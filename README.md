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

```yaml
phase: LG   # LG / SORCIERE / VOTE

joueurs:
  J1:
    ip: 198.0.0.1
    port: 40067
    role: WOLF   # WOLF / VILLAGER / WITCH
    alive: false
  J2:
    ip: 198.0.0.2
    port: 40068
    role: WOLF
    alive: false
  J3:
    ip: 198.0.0.3
    port: 40069
    role: VILLAGER
    alive: true
  J4:
    ip: 198.0.0.4
    port: 40070
    role: VILAGER
    alive: true
  J5:
    ip: 198.0.0.5
    port: 40071
    role: WITCH
    alive: true

votes:
  J1: J3
  J2: J3
  J3: J1

kills:
  wolf: J1
  witch: J2
```
