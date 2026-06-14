= Développement


== L'arrivée d'un participant <partie-arrivee>


=== Découverte et demande d'intégration


=== Élection du site parrain


=== Insertion dans l'anneau


=== Mise à cohérence du nouveau réplicat

Pour que le site puisse initialiser son application (pour afficher l'état du jeu à jour), mais aussi son control (pour être à jour sur les exclusions mutuelles et autres communications de controle), il faut transmettre un état valide à ce nouveau site. Pour cela, nous utilisons le résultat de l'élection du site parrain. 
#quote(block: true)[Nous notons que si cela fonctionne actuellement, car nous augmentons les id des sites au fur et à mesure des ajouts et que nous sélectionnons le site d'id le plus bas, un changement du mécanisme d'élection qui n'assurerait pas que le site parrain ait un control déjà initialisé pourrait mener à des conflits en cas d'ajouts successifs de sites.]

L'obstacle principal à ce stade est d'assurer qu'aucun message impactant l'état du site ne soit perdu entre la prise de la snapshot locale sur le site du parrain et la fin de l'initialisation du nouveau site. Nous avons exploré les pistes suivantes :
- Un freeze global au niveau control pendant l'ajout du nouveau site, pour éviter complètement les problèmes de messages perdus.
- Le nouveau site met en queue tous les messages qu'il reçoit jusqu'à recevoir son initialisation, puis les traite en excluant ceux qui sont antérieurs à la snapshot du parrain, requérant une nouvelle horloge qui contient les horloges de Lamport de tous les sites, pour ignorer uniquement les messages non reçus même si un message causalement plus récent a bien été reçu.

Nous choisissons cette seconde approche, qui permet de traiter le problème plus finement, puisqu'il ne nécessite pas de freeze tous les sites, tout en ne requérant pas plus de développement. Pour permettre d'initialiser le premier site sans qu'il se bloque en attendant une initialisation externe, on ajoute l'option d'un lancement en mode "initisateur".

=== Impact sur l'exclusion mutuelle et les horloges

L'adaptation de notre système d'exclusion mutuelle est relativement simple : lorsqu'un site quitte, il doit être retiré de la liste des sites et le nombre de sites adapté. 
Il faut également s'assurer que le site peut être en section critique, auquel cas on peut le mettre en release et on effectue la vérification pour savoir si on peut entrer en SC.

A l'ajout d'un site, une fois initialisé, il suffit aux autres sites d'ajouter le nouveau dans leurs listes et les méthodes implémentées avant sont fonctionnelles.




== Le départ d'un participant <partie-depart>


=== Départ annoncé


=== Départ subi (panne)


== Synthèse

