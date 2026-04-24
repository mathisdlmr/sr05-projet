# SR05 - Projet

## Proposition d'architecture du système

Navigateur <(Websocket)> server.go <(chan/std)>  application.go <(chan/std)> control.go
                                                                               ^  |
                                                                               |  ⌄
Navigateur <(Websocket)> server.go <(chan/std)>  application.go <(chan/std)> control.go                                                                         

## Proposition d'organisation du projet

Pour info : j'ai commencé à créer des fichiers un peu au piff histoire de pouvoir push les dossiers, mais c'est à revoir

```bash
* cmd/
  * application/
    * main.go
  * control/
    * main.go
  * server/
    * main.go
* internal/
  * application/
    * <à définir mais sûrement la logique du jeu etc.>
    * ...
  * control/
    * <à définir mais sûrement la logique   horloge etc.>
    * ...
  * server/
    * <à définir mais sûrement la logique des websockets etc.>
    * ...
  * transport/
    * io.go # Communication entre les processus en local
    * messages.go # Communication control <-> control
* web/
  * index.html
  * ...
* scripts/
  * 4-ring.sh
  * 5-mesh.sh
  * 7-ring_with_ctl.sh
  * ...
```

## Consignes à garder en tête (pas encore implémentées)
* Les programmes communiquent par stdin et stdout.... Pour tout le reste on utilise stderr
* On sépare le code de "controle" du code de "l'application"
    * Le code de "controle" intercèpte les messages de l'application et réalise un contrôle (par exemple l'estampillage)

## Modélisation d'un état du jeu

```
  phase: LG/SORCIERE/VOTE
  votes: J1:J3,J2:J3,J3:J1
  joueurs: J1:198.0.0.1:40067, J2:IP:PORT
  kills: J1, J2
  alive: J3,J4,J5
```