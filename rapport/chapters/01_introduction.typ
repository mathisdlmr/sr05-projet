= Introduction

== Contexte initial

Le projet de base implémente le fameux jeu de société à rôle caché "Les Loups-garous de Thiercelieux". Étant un jeu initialement très centralisé avec la présence d'un maître du jeu, nous avons réfléchi à une architecture distribuée répondant aux critères du projet :

#figure(
  image("../assets/distributed_game.png", width: 80%),
  caption: [Schématisation de l'architecture avec 5 joueurs],
) <archi-init>

L'idée montrée par @archi-init est que chaque joueur (en blanc) possède un maître du jeu (en vert) qui lui communique l'état de la partie. Ainsi les maîtres du jeu de chaque joueur forment un réseau sous la forme d'un anneau orienté. Cette architecture est davantage détaillée dans le _README.md_ à la racine du projet.

Côté fonctionnalités, le projet gère la cohérence des réplicats
et implémente une sauvegarde répartie horodatée. Cette étude s'intéresse à la question de l'évolution de la liste des participants au fil de la partie.

== Objectifs de l'étude

La question posée par l'étude est la suivante : *comment faire en sorte que la liste de participants puisse évoluer ?* Autrement dit, comment intégrer de nouveaux arrivants et supporter le départ de participants sans perturber l'application développée dans le cadre du projet ? Pour y répondre, nous avons exploré deux idées que nous présentons puis comparons ci-dessous.

=== Première idée : la topologie gérée par le script de lancement

La première approche que nous pensions exploiter se base sur le script #link("https://moodle.utc.fr/mod/page/view.php?id=179855")[dynamich.sh sur Moodle] : c'est le même script qui créé le réseau (avec des `tee`) et qui se charge de son évolution (en manipulant les processus `tee`).
Notre approche pour utiliser cette méthode était que chaque message transitant dans le réseau serait également envoyé à notre script en charge du réseau. Le script ayant créé le réseau pourrait alors parser tous les messages pour y détecter des demandes de modification de topologie au format :

```
/=type=net/=action=addLink/=sender=<i>/=target=<j>
/=type=net/=action=removeLink/=sender=<i>/=target=<j>
```

À la réception d'un tel message, le script met à jour la liste des destinations du site concerné puis relance le `tee` correspondant. L'ajout ou le retrait d'un participant se ramène alors à quelques messages `addLink` / `removeLink` que le script applique lui-même.
Pour mieux visualiser cette approche, le script initialement imaginé est toujours disponible dans le projet, dans #link("../scripts/local_net.sh")[scripts/local_net.sh]

// TODO : insérer un schéma illustrant le routage centralisé par le script
//        (tee pilotés par le script via le canal de commande).

Cette solution a le mérite d'être simple, mais elle présente un défaut rédhibitoire : *le réseau n'est plus du tout décentralisé*. Une entité centrale (le script) observe l'intégralité du trafic et détient seule le pouvoir de reconfigurer la topologie. Cela contredit directement l'esprit du projet, où le contrôle est réparti entre les sites.

=== Seconde idée : chaque site gère lui-même ses liens

Pour rétablir la décentralisation, l'idée est que *chaque site manipule lui-même ses propres entrées et sorties*, sans intermédiaire central. Le script ne sert plus qu'à mettre en place l'anneau initial ; toute évolution ultérieure est prise en charge par les sites eux-mêmes.

Lorsqu'un site souhaite rejoindre le réseau, il ne peut pas s'y greffer de lui-même : il demande à un membre du réseau de l'ajouter. Cette demande déclenche un *algorithme d'élection* qui désigne le site chargé de le parrainer. Faire passer l'intégration par une élection présente un avantage : elle donne la main sur les critères d'admission et les restrictions que l'on souhaite imposer (par exemple écarter de l'élection un site dont la latence est trop élevée, limiter le parrainage à une certaine proximité, etc.).

Une fois le parrain élu, celui-ci reconfigure lui-même ses liens : il interrompt son `tee` courant et en recrée un nouveau pointant vers son contrôle local et vers le nouveau site, puis envoie un message à l'arrivant pour que ce dernier établisse à son tour ses liens vers son propre contrôle et vers le site qui était jusque-là le successeur du parrain. Le nouveau site est ainsi inséré dans l'anneau comme un maillon dans une liste chaînée.

// TODO : insérer un schéma illustrant l'insertion décentralisée
//        (parrain qui recrée son tee, puis branchement du nouveau site).

On obtient alors un fonctionnement *complètement réparti*, à l'exception du script qui crée l'anneau de départ. Cette propriété rend d'ailleurs la solution indépendante de la topologie initiale : elle fonctionnerait vraisemblablement quel que soit le format du réseau (à confirmer), pourvu que deux conditions soient réunies : des canaux *FIFO* (pour ne perdre aucun message) et un mécanisme permettant à un site extérieur de *demander* à rejoindre le réseau.

Cette approche impose toutefois un certain nombre de précautions, que l'implémentation devra traiter :

- garantir qu'*aucun message n'est perdu* pendant la reconfiguration
- *mettre à jour les contrôleurs* avec le nouveau nombre de sites, afin que les mécanismes qui en dépendent (capture d'instantanés, horloges vectorielles, file d'attente répartie) restent corrects
- à l'arrivée, le nouveau venu *spectate* jusqu'à la partie suivante (il n'entre pas dans une partie déjà commencée)
- au départ volontaire, le joueur correspondant est retiré (_kill_) proprement
- gérer la *déconnexion brutale* d'un site : être capable d'intercepter un signal d'arrêt (`SIGKILL`/`SIGTERM`) pour émettre un message de `leave` avant de disparaître

=== Choix retenu

Nous retenons la *seconde idée*. La première, bien que plus simple à mettre en œuvre, réintroduit un point de centralisation (le script) incompatible avec un système réparti : elle reviendrait à confier à un tiers omniscient un rôle que le projet s'attache précisément à distribuer. La seconde approche, en confiant à chaque site la gestion de ses propres liens et en passant par une élection pour l'intégration, préserve la décentralisation et offre en prime une maîtrise fine des conditions d'admission. C'est cette solution que nous détaillons dans le chapitre suivant.