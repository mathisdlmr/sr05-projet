= Développement


== L'arrivée d'un participant <partie-arrivee>


=== Découverte et demande d'intégration


=== Élection du site parrain


=== Insertion dans l'anneau


=== Mise à cohérence du nouveau réplicat

Pour que le site puisse initialiser son application (pour afficher l'état du jeu à jour), mais aussi son contrôle (pour être à jour sur les exclusions mutuelles et autres communications de contrôle), il faut transmettre un état valide à ce nouveau site. Pour cela, nous utilisons le résultat de l'élection du site parrain. 
#quote(block: true)[Nous notons qu'il n'y a pour le moment pas garanti que le site sélectionné soit déjà initialisé lorsqu'il est sélectionné, pouvant mener à un deadlock selon le type d'élection choisi.]

L'obstacle principal à ce stade est d'assurer qu'aucun message impactant l'état du site ne soit perdu entre la sauvegarde de l'état local sur le site du parrain et la fin de l'initialisation du nouveau site (cf. @cas_pb).

#figure(
  image("../assets/cas_pb.png", width: 70%),
  caption: [Cas de perte d'un message (A)],
) <cas_pb>

Pour parer à ce problème, nous écartons la solution d'un freeze global, trop lourde inutilement à notre sens, pour privilégier une solution plus fine décrite ci-après.

Le nouveau site _met en queue_ tous les messages qu'il reçoit jusqu'à recevoir son initialisation, puis les traite _une fois initialisé_ en excluant ceux qui sont _antérieurs à la snapshot_ du parrain, requérant une nouvelle horloge qui contient les estampilles des derniers messages reçus de tous les sites, pour ignorer uniquement les messages non reçus même si un message causalement plus récent a bien été reçu #footnote()[En effet, si nous utilisions une horloge vectorielle pour filtrer les messages, il serait possible qu'un message envoyé par un site A ne soit jamais traité, car le site local a reçu avant un message d'un site B ayant déjà reçu A (causalement ultérieur). C'est bien sûr impossible dans un anneau, mais le contrôle doit être agnostique à l'architecture net.].

Pour permettre d'initialiser le premier site sans qu'il se bloque en attendant une initialisation externe, on ajoute l'option d'un lancement en mode "initiateur".

=== Impact sur l'exclusion mutuelle et les horloges

L'adaptation de notre système d'exclusion mutuelle est relativement simple : lorsqu'un site quitte, il doit être retiré de la liste des sites et le nombre de sites adapté. 
Il faut également s'assurer que le site peut être en section critique, auquel cas on peut le mettre en release et on effectue la vérification pour savoir si on peut entrer en SC.

A l'ajout d'un site, une fois initialisé, il suffit aux autres sites d'ajouter le nouveau dans leurs listes et les méthodes implémentées avant sont fonctionnelles.

== Le départ d'un participant <partie-depart>



=== Départ annoncé



=== Départ subi (panne)

Le cas du départ subi n'a pas été implémenté car le projet ne semblait pas demander une forme de résilience du système, et nous ne souhaitions pas rajouter de la complexité au projet sur des notions qui ne concernent pas directement le cours. Tout de même, nous avons effectué quelques recherches sur le sujet, et une solution assez facilement implémentable semblait sortir du lot : la documentation sur les FIFOs indique que lorsqu'un processus essaye d'écrire dans une FIFO dans laquelle aucun processus ne lit, alors une erreur `SIGPIPE` est soulevée (#link("https://www.man7.org/linux/man-pages/man7/fifo.7.html#NOTES")[Source]). Ainsi, il suffirait d'ajouter un try/catch dans le package `io.go` lorsque l'on envoie un message, et considérer que si l'on reçoit une erreur `SIGPIPE` lorsque l'on écrit à notre successeur, alors c'est que ce dernier n'est plus joignable et qu'il doit être exclu du réseau. Afin de reformer l'anneau il faudrait donc maintenir un tableau ordonnée avec l'identifiant de chacun des sites, et essayer de contacter successivement chacun de ses successeurs jusqu'à en trouver un joignable. On pourrait ensuite reprendre notre méthode classique pour exclure un site de la couche contrôle et applicative pour demander d'exclure notre ancien successeur.

== La sauvegarde répartie dans un réseau dynamique <partie-sauvegarde>

La sauvegarde répartie évoquée en introduction est un instantané au sens du cours (chapitre 6, algorithme 11) : à la bascule un site passe au rouge, leste de sa couleur les messages applicatifs qu'il émet, et l'initiateur reconstitue un état global cohérent, les états locaux plus les messages en transit, jusqu'à une terminaison explicite. Chaque site y fige son état local de façon atomique, par un bref gel de son control et de son application ; ce mécanisme appartient à l'implémentation de base de l'algorithme, que l'étude réutilise sans le modifier.

De tous les mécanismes du projet, c'est l'un de ceux qui supportent le plus mal un groupe qui évolue. Sa terminaison repose sur un nombre d'états attendus fixé à $N - 1$ à la bascule et sur un bilan de messages en transit, deux quantités qui supposent $N$ constant pendant la capture ; une arrivée ou un départ en cours de route les fait dériver, et l'instantané ne termine plus ou mêle deux compositions du groupe. Il suppose aussi une topologie, puisque l'algorithme du cours diffuse sa collecte sur l'anneau, que l'étude rend justement variable. C'est donc ici que le membership pèse le plus, ce qui répond à la question du sujet : oui, le projet appelle une solution particulière. Nos corrections suivent ces deux fragilités.

=== Indépendance vis-à-vis de la topologie

L'algorithme 11 suppose un anneau : le message de collecte fait le tour, relayé de site en site. Nous remplaçons cette hypothèse par un contrat avec la couche net, notre raccordement côté sauvegarde : tout message de contrôle émis est livré exactement une fois à chacun des autres membres de la vue, sans écho (l'émetteur ne reçoit pas son propre message). La sauvegarde ne voit plus que ce contrat, et non la forme du réseau.

Deux conséquences. Le relais disparaît : un état ou un prépost destiné à l'initiateur lui parvient directement, donc un site non initiateur ignore ces messages au lieu de les retransmettre. Et le comptage change : une de nos émissions vaut $N - 1$ livraisons, nous incrémentons donc le bilan de $N - 1$ par message applicatif émis, ce qui maintient l'invariant voulant que la somme des bilans égale le nombre de messages en transit. La terminaison reste explicite : l'initiateur attend exactement $N - 1$ états et un bilan nul avant de diffuser l'état global final, condition fermée et non un délai ou une heuristique.

=== Robustesse au membership : les vues

Rendre l'instantané indépendant de la topologie ne suffit pas, il faut encore le protéger du changement de composition lui-même. Nous attachons pour cela un numéro de vue à chaque site, incrémenté à chaque modification du groupe. Tout message de contrôle porte la vue de son émetteur (champ `view` du format wire), ce qui permet à un récepteur de reconnaître un message émis sous un membership périmé.

Ce tag sert de filtre. Un message applicatif reçu dans une vue différente de la nôtre est appliqué mais pas compté : on exécute bien son effet sur le jeu (file d'attente, exclusion mutuelle) pour ne pas désynchroniser l'application, mais on saute tout le traitement l'algorithme 11 (bascule, décrément du bilan, détection de prépost). C'est cohérent avec la remise à zéro du bilan, puisque l'émission du message a été effacée du compte au changement de vue, sa réception ne doit pas y figurer non plus. Les messages de collecte d'une autre vue (état, prépost, état global final) sont, eux, purement ignorés, pour ne jamais mélanger deux memberships dans un même état global.

Le changement de vue est concentré dans une seule opération, déclenchée par l'ajout ou le retrait d'un site. Elle incrémente la vue, remet le bilan à zéro et, si une capture est en cours, l'avorte : compteurs remis à blanc, et notification à l'application du côté de l'initiateur. Le filtrage fait le reste, un message de collecte resté en vol et marqué d'une vue périmée est ignoré à l'arrivée, il ne peut pas ressusciter une capture déjà jetée. Un nouveau snapshot repartira proprement dans la nouvelle vue.

=== Articulation, arrivée, départ et limites

Le mécanisme de vue se branche sur la gestion des participants (parties #ref(<partie-arrivee>, supplement: none) et #ref(<partie-depart>, supplement: none)). À l'arrivée, le nouveau site hérite de la vue de son parrain quand celui-ci lui transmet son état de référence, puis son ajout incrémente la vue ; il démarre donc dans la même vue que les autres. Au départ, le chemin est symétrique : le joueur signale son départ à son control, qui le diffuse ; chaque site retire l'émetteur de sa vue et prévient son application. Ce retrait emprunte la même opération de changement de vue, il avorte donc une capture en cours. La couche net, prévenue par ce même message, défait les liens (partie réseau, non traitée ici).

Tout cela tient à une hypothèse : les départs sont annoncés. Un site qui tombe en panne sans prévenir reste un membre de la vue, et une capture lancée ensuite attendrait sans fin son état. C'est le rôle d'un détecteur de pannes, comme dans les systèmes à synchronie virtuelle @birman-joseph-1987 ; nous ne l'implémentons pas, notre vue suppose des départs coopératifs. Détecter de façon fiable le départ subi dépasse notre contribution et rejoint la partie #ref(<partie-depart>, supplement: none)