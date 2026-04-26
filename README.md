# SR05 - Projet

## TODO : suppr une fois le projet terminé

### Proposition d'architecture du système

Navigateur <(Websocket)> server.go <(chan/std)> application.go <(chan/std)> control.go
^ |
| ⌄
Navigateur <(Websocket)> server.go <(chan/std)> application.go <(chan/std)> control.go

### Proposition d'organisation du projet

Pour info : j'ai commencé à créer des fichiers un peu au piff histoire de pouvoir push les dossiers, mais c'est à revoir

```bash
* cmd/
  * application/
    * main.go       # Lance l'application (créé un logger, un io, et on lance internal/application)
  * control/
    * main.go       # Lance le contrôle (créé un logger, un io, et on lance internal/control)
  * server/
    * main.go       # Lance le serveur (créé un logger, un io, et on lance internal/server)
* internal/
  * application/
    * app.go        # Contient la logique principale de l'application
    * filter.go     # Forme un filtre sur ce que l'on souhaite forward ou non au server (ce que le joueur a le droit de voir ou pas)
    * state.go      # Contient les structures de données représentant l'état du jeu côté application
  * control/
    * control.go    # Contient la logique principale du centre de contrôle
    * state.go      # Contient les structures de données représentant l'état du jeu côté centre de contrôle
    * <horloge.go, snapshot.go...> Enfin bref ce qui est demandé pour le projet
  * server/
    * server.go     # Gère le serveur (pour communiquer via ws avec un navigateur)
* pkg/
  * logger/
    * logger.go     # Logger mis en forme (couleur, nom des processus, PID, etc.)
  * transport/
    * io.go         # Gère la lecture sur stdin et écriture sur stdout
    * messages.go   # Gère la construction et lectures des messages envoyés/reçus
* web/
  * index.html
  * ...
* scripts/
  * 4-ring.sh
  * 5-mesh.sh
  * 7-ring_with_ctl.sh
  * ...
```

## Prémisses

> _Choix étrange d'avoir choisi le jeu du loup-garou pour modéliser une application répartie..._

Effectivement, de prime abord le jeu du loup-garou est un jeu extrêmement centralisé (difficile de faire plus centralisé même) :

- un.e maître.sse du jeu ("MJ") a la totale connaissance du jeu,
- les joueurs.euses n'ont cependant aucune connaissance,
- seul.e le.a MJ s'occupe d'appliquer les règles
- et iel chosit quand est-ce que l'on passe d'une phase du jeu à une autre

![centralized_game](./docs/centralized_game.png "Un jeu de loup-garou classique")

Cependant, si on change un peu notre façon de voir les choses, on peut imaginer un système réparti.

### Une nouvelle façon de voir le jeu

Au lieu de visualiser le jeu comme on peut avoir l'habitude d'y jouer, avec tou.te.s les participant.e.s en cercle et le.a MJ au centre, on va plutôt se placer dans une autre configurations :

- Chacun.e des participant.e.s est placé.e dans une pièce isolée (1 participant.e par pièce)
- On possède autant de MJ que de participant.e.s
- A chaque phase, les MJ se réunissent et décident de l'état actuel du jeu et de quelle règle appliquer
- Ensuite, chacun.e retourne voir son.a participant.e pour l'informer de l'état du jeu
- A chaque changement dans l'état du jeu (un.e participant.e a vôté pour quelqu'un, fin du phase du jeu, etc.), tou.te.s les MJ se réunissent pour se partager l'information et décider de quoi faire.

![distributed_game](./docs/distributed_game.png "Un jeu de loup-garou distribué !")

On se retrouve alors dans un cadre très particulier du loup-garou : un loup-garou décentralisé entre $n$ MJ

### D'un point de vue implémentation

D'un point de vue implémentation, cette vision du jeu colle plutôt bien avec l'architecture proposée dans l'énoncé, à savoir que chaque machine fait un "centre de controle", une application, ainsi qu'un frontend.

- Le centre de controle fait alors parti du système réparti (c'est l'ensemble des MJ qui communiquent),
- L'application s'occupe de mettre en forme le jeu et filtrer les informations (chaque MJ met en forme son discours pour communqiuer avec son.a participant.e attribué.e et ne pas révéler le rôle d'un.e autre participant.e),
- Et finalement notre frontend représente notre participant.e qui joue

#### Architecture du système d'un point de vue physique

![physical_architecture](./docs/physical_system.png "Architecture physique de notre système")

#### Architecture du système d'un point de vue virtuel (code)

![virtual_architecture](./docs/virtual_system.png "Architecture Virtuelle de notre système")

## Rappel des consignes

Le projet porte sur la création d'une application répartie respectant les contraintes suivantes :

- L'application répartie utilise une donnée partagée entre les sites
  - Définir un scénario qui nécessite le partage d'au moins une donnée entre plusieurs "sites" : les instances de l'application réparties s'exécutant sur chaque site travaillent sur des réplicats qui sont des copies locales de la donnée partagée.
- Les réplicats restent cohérents
  - N'autoriser qu'une seule modification de réplicat à la fois et propager les modifications aux autres réplicats.
  - Implémenter pour cela l'algorithme de la file d'attente répartie qui organise une exclusion mutuelle. La section critique correspond à l'accès exclusif à la donnée. À vous de voir s'il faut une exclusion mutuelle pour l'écriture et la lecture de la donnée partagée. À vous de voir comment adapter l'algorithme pour diffuser la mise à jour de la donnée partagée.
  - Cet algorithme utilise lui-même les estampilles, qu'il est donc nécessaire d'implémenter.
- L'application répartie inclut une fonctionnalité de sauvegarde répartie datée
  - Implémenter pour cela un algorithme de calcul d'instantanés du cours.
  - Pour dater la sauvegarde, utiliser des horloges vectorielles.
- L'application répartie est clairement structurée
  - Utiliser une architecture qui distingue les fonctionnalités applicatives des fonctionnalités de contrôle.
  - Définir au moins un réseau convaincant pour les tests.

## Modélisation d'un état du jeu

```go
GameState{
    Phase: "VOTE", // "LOBBY" | "NIGHT" | "WITCH" | "VOTE" | "END"
    Players: {
        "J1": { ID: "J1", Role: "WOLF", Alive: false }, // TODO : sûrement une section IP+port à ajouter ici par la suite
        "J2": { ID: "J2", Role: "WOLF", Alive: true  },
        "J3": { ID: "J3", Role: "WITCH", Alive: true  },
        "J4": { ID: "J4", Role: "VILLAGER", Alive: true  },
        "J5": { ID: "J5", Role: "VILLAGER", Alive: true  },
    },
    Votes: {
        "J2": "J4",
        "J3": "J2",
        "J4": "J2",
        "J5": "J2",
    },
    KillWolf: "J4",
    KillWitch: "save:J4", // ou "poison:J<id>" ou "" (rien)
    Winner: "", // "" | "WOLVES" | "VILLAGERS"
    MyID: "J3", // identifiant du joueur local (ajouté par application avant envoi)
}
```

## Communication entre les processus

```
Navigateur ---- JSON ----> Server
Server     --- "/=type=state/=data={}" ---> Navigateur # TODO : c'est à revoir ça

Server ---- Transparent ----> Application
Application ---- Transparent ----> Server

Application --- "/=type=.../=..." ---> Control
Control --- "/=type=state/=data={}" ---> Application

Control --- "BROADCAST:/=from=.../" ---> [réseau] -> autres Controls # Le from ici permet d'éviter de lire son broadcast
```

### Actions Navigateur -> Serveur

```json
// Rejoindre le lobby
{ "action": "join" }

// Indiquer qu'on est prêt à démarrer
{ "action": "ready" }

// Vote du village (phase VOTE)
{ "action": "vote", "target": "J2" }

// Les loups choisissent leur victime (phase NIGHT, loups seulement)
{ "action": "wolfkill", "target": "J4" }

// La sorcière sauve la victime des loups (phase WITCH, sorcière seulement)
{ "action": "witchsave" }

// La sorcière empoisonne quelqu'un (phase WITCH, sorcière seulement)
{ "action": "witchkill", "target": "J5" }

// La sorcière ne fait rien (phase WITCH, sorcière seulement)
{ "action": "witchpass" }
```

### Serveur -> Navigateur

On forward tout de l'application de manière bête et méchante

Soit un état complet du jeu : `/=type=state/=data={"phase":"VOTE","players":{...},"votes":{...},"myId":"J3"}`

Soit une erreur : `/=type=error/=msg=action interdite pendant cette phase`

### Application -> Control

Même format que Navigateur -> Serveur mais au format imposé dans l'énoncé (comme `/=type=join/=player=J3` par exemple)

### Control -> Control

Même format que Application -> Control, mais on ajoute "BROADCAST:/=from=J<id_joueur>/=type=..."
