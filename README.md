# SR05 - Projet

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

## Organisation du projet

```bash
├── 0-old
│   └── ... # Contient les différentes étapes de l'activité "Projet" sur Moodle
├── cmd
│   ├── application
│   │   └── main.go # Lance l'application (internal/application/app.go) qui embarque le server (internal/server/server.go) maintenant la WebSocket avec le frontend
│   ├── control
│   │   └── main.go # Lance le centre de contrôle (internal/control/control.go) qui communique dans notre système réparti
│   └── net
│       └── main.go # Lance le centre de contrôle (internal/net/net.go) qui permet de gérer le réseau de notre système réparti
├── docs
│   └── ... # Contient les images utilisées par le README
├── internal
│   ├── application
│   │   ├── app.go             # Dispatcher central : stdin <-> control, WebSocket <-> navigateur
│   │   ├── browser.go         # Envoi d'événements vers le navigateur (pushEvent, sendInit)
│   │   ├── game.go            # Gère la logique du jeu : gestion des évènements, gestion de l'état du jeu, etc.
│   │   ├── roleattribution.go # Attribution aléatoire distribuée des rôles
│   │   ├── snapshot.go        # Gestion des demandes de snapshot et restoration à partir d'une snapshot
│   │   ├── state.go           # Structures de données (GameState, Player, Phase, Role)
│   │   └── transitions.go     # Transitions de phase (→WITCH, →VOTE, →NIGHT)
│   ├── control
│   │   ├── control.go         # Dispatcher central : reçoit les message + les redirige vers les handles, et gère l'envoie de messages
│   │   ├── criticalsection.go # Gestion des sections critiques (demande de SC, Acknowledge, fin de SC)
│   │   ├── handlers.go        # Handlers pour gérer les requêtes de l'application locale et des controler des autres sites
│   │   └── snapshot.go        # Implémentation des snapshots réalisées dans notre système
│   ├── net
│   │   └── net.go             # Permet de gérer un réseau dynamique pour notre système réparti
│   └── server
│       └── server.go          # Serveur HTTP + WebSocket (gorilla/websocket)
├── pkg
│   ├── logger
│   │   └── logger.go          # Logger coloré sur stderr (DEBUG/INFO/WARN/ERROR)
│   └── transport
│       ├── io.go              # Lecture stdin / écriture stdout
│       └── message.go         # Sérialisation/désérialisation des messages inter-processus
├── frontend
│   └── src                    # Sources React+TypeScript (compilées vers web/)
├── web                        # Frontend compilé, servi par l'application
├── k8s                        # Chart Helm pour déploiement Kubernetes
├── scripts
│   ├── local.sh               # Déploiement local (FIFOs Unix + anneau)
│   └── reset-game.sh          # Réinitialisation d'une partie en cours sur Kubernetes
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── TODO
```

## Modélisation d'un état du jeu

```go
GameState{
    Phase: "VOTE", // "LOBBY" | "NIGHT" | "WITCH" | "VOTE" | "END"
    Players: {
        "J1": { ID: "J1", Role: "WOLF", Alive: false },
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
    KillWitch: "J5",       // playerID de la cible de la sorcière, ou "" si non utilisé
    Winner: "", // "" | "WOLVES" | "VILLAGERS"
    MyID: "J3", // identifiant du joueur local (ajouté par application avant envoi)
}
```

## Communication entre les processus

### Routage physique (cf. `scripts/local.sh`)

Le routage entre processus se fait via des FIFO Unix mises en place par le script de lancement :

```
out_app_i  ->  in_ctl_i                       (app -> contrôle local)
out_ctl_i  ->  in_app_i  +  in_ctl_{i+1}      (contrôle -> app locale + suivant sur l'anneau)
```

Le `tee` qui duplique `out_ctl_i` envoie donc chaque sortie de contrôle à la fois à l'application locale et au contrôle suivant. C'est le **type** du message qui détermine ce que le récepteur en fait.

### Format des messages stdin/stdout

Tous les messages échangés entre `application` et `control` (et entre contrôles) suivent le format défini dans `pkg/transport/message.go` :

```
/=type=<type>/=action=<action>/=timestamp=<n>/=vectorClock=<v1,v2,...>/=sender=<id>/=cle1=val1/=cle2=val2/...
```

Les types reconnus sont :

| Type            | Action                | Émetteur         | Timestamp | Sens / Description                                     |
| --------------- | --------------------- | ---------------- | :-------: | ------------------------------------------------------ |
| `application`   | `requestCS`           | app locale       |    non    | app -> contrôle local : demande de section critique     |
| `application`   | `beginCS`             | contrôle local   |    non    | contrôle -> app locale : SC accordée                    |
| `application`   | `endCS`               | app locale       |    non    | app -> contrôle local : fin de SC + données             |
| `application`   | `releaseCS`           | contrôle local   |    non    | contrôle -> app locale : données d'un releaseCS distant |
| `control`       | `requestCS`           | contrôle local   |    oui    | diffusion de la demande sur l'anneau                   |
| `control`       | `acknowlegeCS`        | contrôle local   |    oui    | acquittement envoyé au demandeur (`data.target`)       |
| `control`       | `releaseCS`           | contrôle local   |    oui    | libération + diffusion des données sur l'anneau        |

L'évitement de boucle sur l'anneau se fait en comparant `sender` à l'ID local du contrôle (cf. `internal/control/control.go`).

### Transmission des données

Toutes les actions de jeu (rejoindre, démarrer, voter, actions de la sorcière…) passent par la section critique. La SC garantit qu'un seul site à la fois modifie l'état partagé. Le flux est le suivant :

1. Le navigateur envoie une action à l'application locale
2. L'application demande la SC au contrôle local (`requestCS`)
3. Le contrôle accorde la SC (`beginCS`) -> l'application exécute l'action localement et envoie `endCS` avec les données de l'action
4. Le contrôle diffuse un `releaseCS` (avec les données) sur l'anneau
5. Chaque autre site reçoit le `releaseCS`, transmet les données à son application locale, qui applique la même action -> ce qui permet la convergence des réplicas

L'évitement de boucle sur l'anneau se fait en comparant `sender` à l'ID local du contrôle


### Actions Navigateur -> Application (via WebSocket, JSON)

```json
// Demander l'état complet du jeu
{ "action": "init" }

// Démarrer la partie (phase LOBBY)
{ "action": "start" }

// Vote du village (phase VOTE)
{ "action": "vote", "target": "J2" }

// Les loups choisissent leur victime (phase NIGHT, loups seulement)
{ "action": "wolfkill", "target": "J4" }

// La sorcière sauve la victime des loups (phase WITCH, sorcière seulement)
{ "action": "witchsave" }

// La sorcière empoisonne quelqu'un (phase WITCH, sorcière seulement)
{ "action": "witchkill", "target": "J5" }

// La sorcière ne fait rien (phase WITCH, sorcière seulement)
{ "action": "witchskip" }
```

> _En cas de déconnexion du navigateur, le programme prend en charge la reconnexion : l'application React qui sert de frontend reformule une requête `init` à l'application, qui lui renvoie l'état complet du jeu_

### Application -> Navigateur (via WebSocket, JSON)

L'application pousse des **événements ciblés** (pas de push d'état complet à chaque changement). Le navigateur maintient son état local à partir de ces événements.

| `type`           | Déclencheur                                      | Champs notables                                              |
| ---------------- | ------------------------------------------------ | ------------------------------------------------------------ |
| `init`           | Connexion WS ou action `init`                    | `phase`, `myId`, `myRole`, `myAlive`, `players`, `votes`, `killWolf?` |
| `playerJoined`   | Nouveau joueur dans le lobby                     | `playerId`                                                   |
| `gameStart`      | Partie démarrée, rôles distribués                | `myRole`, `players` (rôles filtrés)                          |
| `wolfVoted`      | Un loup a voté (phase NIGHT)                     | `voter`, `target` (visible seulement pour les loups)         |
| `phaseChange`    | Passage en phase WITCH                           | `phase`, `killWolf?` (visible seulement pour la sorcière)    |
| `nightKills`     | Kills de la nuit appliqués                       | `killed[]`, `nextPhase` (`"VOTE"` ou `"END"`)                |
| `voted`          | Un joueur a voté (phase VOTE)                    | `voter`, `target`                                            |
| `voteEliminated` | Joueur éliminé par vote                          | `playerId`, `nextPhase` (`"NIGHT"` ou `"END"`)               |
| `gameEnd`        | Fin de partie                                    | `winner` (`"WOLVES"` ou `"VILLAGERS"`), `players` (tous rôles révélés) |

Les rôles dans `players` sont filtrés : chaque joueur ne connaît que son propre rôle (et celui des autres loups s'il est loup). Tous les rôles sont révélés en phase `END`.

---

## Algorithmes distribués (`internal/control/control.go`)

### Horloge de Lamport (scalaire)

Chaque site maintient un compteur `clock` : incrémenté à l'envoi (`clock++`), recalé à la réception (`clock = max(clock, ts) + 1`) tel que vu dans le cours. Cette horloge permet d'ordonner les requêtes dans la file d'attente.

### Horloge vectorielle

Tous les messages sont envoyés avec une estampille basée sur l'horloge de Lamport, utilisée pour la file d'attente répartie, mais aussi avec une estampille vectorielle basée sur une horloge vectorielle et utilisée pour la sauvegarde (capture d'instantanées).

### Exclusion mutuelle — File d'attente répartie de Lamport

La file d'attente répartie a été implémentée au plus proche de l'algorithme indiqué dans le cours, au niveau de la couche **Control** Elle attend un message de l'application (type application, action 'requestCS') pour faire une demande d'entrée en section critique. Une fois entré, le contrôle envoie un mesage (action: ' beginCS') à l'application pour lui indiquer le début de l'action. Puisque nous utilisons les files d'attente pour une seule modification à la fois, l'application répond normalement directement avec un 'endSC' pour terminer la section critique, accompagné dans la data du message à transmettre, de manière à synchroniser l'ensemble des états.

### Capture d'instantanés

L'algorithme implémenté est l'algorithme 11 du chapitre sur la capture d'instantanés (variante Lai-Yang avec lestage, reconstitution de configuration et détection de terminaison). Il se distingue des variantes plus simples (algo 5 / Chandy-Lamport) sur trois points :

- la **reconstitution** collecte chez l'initiateur les états locaux et les messages préposts, donc on obtient un EG réellement utilisable et redistribuable, pas seulement une coloration ;
- la **détection de terminaison** (`NbÉtatsAttendus == 0 && NbMsgAttendus == 0`) évite tout timeout côté initiateur ;
- le **lestage** n'exige pas que les canaux applicatifs soient FIFO.

**Datage** : chaque site copie son horloge vectorielle au moment précis de sa bascule rouge. La collection des `vectorClock_i` capturés forme la date globale de l'instantané (un N-vecteur de vecteurs), ce qui ne serait pas possible avec une simple horloge scalaire de Lamport.

**Flot général** (déclenché par un bouton "Sauvegarder" dans le navigateur, qui émet `{action: "startSnapshot"}`) :

1. L'initiateur bascule rouge, capture son état Control (queue, bilan, vectorClock) et demande l'état App via stdio. Pendant le round-trip, les messages entrants sont mis en file et rejoués après (option B du design).
2. Il envoie un `Wakeup` sur l'anneau pour garantir la propagation de la bascule même sans trafic applicatif (cf. exo 127-128 du poly).
3. Les non-initiateurs basculent à la première réception d'un message rouge, capturent leur état, envoient un message `[état]` sur l'anneau vers l'initiateur.
4. Les messages applicatifs reçus blancs alors qu'on est rouge sont détectés comme **préposts** et retransmis à l'initiateur.
5. Quand l'initiateur a reçu les `N-1` messages `[état]` et compensé les préposts en transit, il diffuse l'EG complet sur l'anneau via `ActionSnapshotComplete`. Chaque site sauvegarde l'EG et le pousse à son App locale qui l'affiche dans le navigateur.

Le code se trouve principalement dans `internal/control/snapshot.go` (algorithme distribué) et `internal/application/snapshot.go` (deep-copy du `GameState` + interface navigateur). Quelques déviations par rapport au cours sont commentées, notamment `bilan += N-1` par envoi pour s'adapter au broadcast via `tee`, et le `bilan--` reporté après le check de bascule pour préserver l'invariant `Σ bilan = nb messages en transit`.

---

## Attribution des rôles (distribuée)

Au lancement (`start`), les rôles sont attribués via une chaîne de sections critiques :

1. Le site déclencheur choisit aléatoirement son rôle parmi les rôles disponibles, l'applique localement, et le diffuse via le `releaseCS` de `start`
2. À la réception, chaque autre site applique le rôle annoncé puis demande lui-même une SC (`attribution`) pour déclarer le sien
3. Chaque site choisit aléatoirement dans les rôles encore libres (`listAvailableRoles`), diffuse `applyattribution` via son `releaseCS`
4. Quand tous les joueurs ont un rôle (`checkEveryoneHasRole`), la transition vers NIGHT est déclenchée et `gameStart` est envoyé au navigateur

**Distribution :** `floor(n/3)` loups, 1 sorcière, le reste en villageois

---

## Déploiement local (`scripts/local.sh`)

```sh
make run            # build + lance le script local.sh qui forme un anneau de 3 sites

make build          # permet de uniquement compiler le frontend et les binaires Go
./scripts/local.sh 5000  # permet de former l'anneau entre 3 sites (localhost ports 4444, 4445, 4446), utilise les binaires Go
```

Le script crée 4 FIFOs Unix par site dans `/tmp/`, lance les processus, puis relie les FIFOs :

```
cat  /tmp/out_app$i  >  /tmp/in_ctl$i          (app -> contrôle local)
tee  /tmp/in_app$i   <  /tmp/out_ctl$i  >  /tmp/in_ctl$NEXT (contrôle -> app locale + site suivant)
```

`Ctrl+C` déclenche le nettoyage automatique (kill des processus + suppression des FIFOs).

---

## Déploiement Kubernetes (`k8s/`)

Chart Helm dans `k8s/`. Chaque pod (un par joueur) contient 4 containers :

| Container    | Rôle                                                                                  |
| ------------ | ------------------------------------------------------------------------------------- |
| `init-fifos` | (initContainer) Crée les FIFOs sur le PVC partagé, attend que tous les sites soient prêts |
| `application`| Binaire application                                                                   |
| `control`    | Binaire control                                                                       |
| `router`     | Relie les FIFOs : `cat out_app -> in_ctl` + `tee in_app < out_ctl > in_ctl_next`      |

Un PVC partagé (`werewolf-fifos`, 64 Mi) permet aux containers de tous les pods d'accéder aux mêmes FIFOs. 
Un Service + Ingress Traefik TLS est créé par joueur (`j1-<domainSuffix>`, `j2-<domainSuffix>`…).

```sh
# Déploiement
helm upgrade --install werewolf ./k8s
helm upgrade --install werewolf ./k8s --set players=3  # surcharger le nb de joueurs

# Réinitialiser une partie en cours (scale 0 -> attendre -> scale N)
./scripts/reset-game.sh
./scripts/reset-game.sh werewolf 5
```
