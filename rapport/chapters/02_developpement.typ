= Développement


== L'arrivée d'un participant <partie-arrivee>


=== Découverte et demande d'intégration


=== Élection du site parrain


=== Insertion dans l'anneau


=== Mise à cohérence du nouveau réplicat

Pour que le site puisse initialiser son application (pour afficher l'état du jeu à jour), mais aussi son control (pour être à jour sur les exclusions mutuelles et autres communications de controle), il faut transmettre un état valide à ce nouveau site. Pour cela nous utilisons le résultat de l'élection du site parrain. 
#quote(block: true)[Nous notons que si cela fonctionne actuellement car nous augmentons les id des sites au fur et a mesure des ajouts et que nous selectionnons le site d'id le plus bas, un changement du mecanisme d'election qui n'assurerai pas que le site parrain ait un control déjà initialisé pourrait mener à des conflits en cas d'ajouts successifs de sites.]

L'obstacle principal à ce stade est d'assurer qu'aucun message impactant l'état du site ne soit perdu entre la prise de la snapshot locale sur le site du parrain et la fin de l'initialisation du nouveau site. Nous avons exploré les pistes suivantes :
- Le nouveau site met en queue tous les messages qu'il reçois jusqu'a recevoir son initialisation, puis les traite en excluant ceux qui sont antérieurs à la snapshot du parrain, requérant une nouvelle horloge qui contient les horloges de lamport de tous les sites, pour ignorer uniquement les messages non reçus même si un message causalement plus récent à bien été reçu.
- Un freeze global au niveau control pendant l'ajout du nouveau site, pour éviter complètement les problèmes de messages perdus.

=== Impact sur l'exclusion mutuelle et les horloges


== Le départ d'un participant <partie-depart>


=== Départ annoncé


=== Départ subi (panne)


== Synthèse

