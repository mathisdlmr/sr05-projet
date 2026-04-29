// Package control - s'occupe de la communication entre nos sites, comment partager l'état du jeu, choisir les phases, etc.
// control.go - coeur du centre de contrôle : maintient le réplica local de l'état du jeu, reçoit les messages de l'application 
// locale et des autres centres de contrôle, applique la logique de jeu, et coordonne les transitions de phase avec les autres sites
//
// Communication :
// * Messages de l'application locale  -> stdin (sans préfixe)
// * Messages des autres controles     -> stdin (préfixés "BROADCAST:")
// * Messages vers l'application locale -> stdout (sans préfixe)
// * Broadcast vers les autres sites   -> stdout (préfixés "BROADCAST:")

package control

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

type Control struct {
	myID       string
	nbSites    int
	state      GameState
	io         *transport.IO
	log        *logger.Logger

	lobbyJoined readySet // sites ayant rejoint
	lobbyReady  readySet // sites prêts à démarrer

	wolfVotes map[string]string // joueur_loup_id: cible_id
	villageVotes map[string]string // joueur_voteur_id: cible_id

	witchDone bool

	// File d'attente répartie (Algo 28 du poly) — cf. queue.go
	lamport int
	tab     []tabEntry
	myIndex int
	pending *pendingSC
}

func New(myID string, nbSites int, io *transport.IO, log *logger.Logger) *Control {
	tab := make([]tabEntry, nbSites)
	for i := range tab {
		tab[i] = tabEntry{Kind: entryRelease, Date: 0}
	}
	return &Control{
		myID:         myID,
		nbSites:      nbSites,
		state:        newGameState(myID),
		io:           io,
		log:          log,
		lobbyJoined:  make(readySet),
		lobbyReady:   make(readySet),
		wolfVotes:    make(map[string]string),
		villageVotes: make(map[string]string),
		lamport:      0,
		tab:          tab,
		myIndex:      siteIndex(myID),
	}
}

func (c *Control) Run() {
	c.log.Info("Run", fmt.Sprintf("démarrage controle id=%s nbSites=%d", c.myID, c.nbSites))
	for {
		line, err := c.io.ReadLine()
		if err == io.EOF {
			c.log.Info("Run", "stdin fermé, arrêt")
			return
		}
		if err != nil {
			c.log.Error("Run", "lecture stdin: "+err.Error())
			return
		}

		if strings.HasPrefix(line, "BROADCAST:") {
			c.handleFromControl(strings.TrimPrefix(line, "BROADCAST:"))
		} else {
			c.handleFromApp(line)
		}
	}
}

// ========================= Envoi ========================= //

// sendToApp - envoie l'état courant (non filtré) à l'application locale
func (c *Control) sendToApp() {
	data, err := json.Marshal(c.state)
	if err != nil {
		c.log.Error("sendToApp", err.Error())
		return
	}
	msg := transport.Build("type", "state", "data", string(data))
	c.io.MustSend(msg)
}

// broadcast - envoie un message à tous les autres centres de contrôle
func (c *Control) broadcast(msg string) {
	full := "BROADCAST:" + transport.Build("from", c.myID) + msg // Le message commence par "BROADCAST" pour différencier des msgs de l'appli locale, et on ajoute "from" pour ne pas lire ses propres broadcasts
	c.log.Debug("broadcast", full)
	c.io.MustSend(full)
}

// =============== Transitions de phase =============== //

func (c *Control) transitionToNight() {
	c.state.Phase = PhaseNight
	c.state.KillWolf = ""
	c.state.KillWitch = ""
	c.wolfVotes = make(map[string]string)
	c.witchDone = false
	c.log.Info("phase", "-> NIGHT")
	c.sendToApp()
	c.broadcast(transport.Build("type", "state", "data", c.marshalState()))
}

func (c *Control) transitionToWitch() {
	c.state.Phase = PhaseWitch
	c.log.Info("phase", "-> WITCH (cible loups: "+c.state.KillWolf+")")
	c.sendToApp()
	c.broadcast(transport.Build("type", "state", "data", c.marshalState()))
}

func (c *Control) transitionToVote() {
	c.state.Phase = PhaseVote
	c.state.Votes = make(map[string]string)
	c.villageVotes = make(map[string]string)
	c.log.Info("phase", "-> VOTE")
	c.sendToApp()
	c.broadcast(transport.Build("type", "state", "data", c.marshalState()))
}

func (c *Control) transitionToEnd(winner string) {
	c.state.Phase = PhaseEnd
	c.state.Winner = winner
	c.log.Success("phase", "-> END, gagnant: "+winner)
	c.sendToApp()
	c.broadcast(transport.Build("type", "state", "data", c.marshalState()))
}

// ================= Logique de jeu ================= //

func (c *Control) countAliveWolves() int {
	n := 0
	for _, p := range c.state.Players {
		if p.Alive && p.Role == RoleWolf {
			n++
		}
	}
	return n
}

func (c *Control) countAlive() int {
	n := 0
	for _, p := range c.state.Players {
		if p.Alive {
			n++
		}
	}
	return n
}

func (c *Control) checkWin() string {
	wolves, others := 0, 0
	for _, p := range c.state.Players {
		if !p.Alive {
			continue
		}
		if p.Role == RoleWolf {
			wolves++
		} else {
			others++
		}
	}
	if wolves == 0 {
		return "VILLAGERS"
	}
	if wolves >= others {
		return "WOLVES"
	}
	return ""
}

func (c *Control) killPlayer(id string) {
	if p, ok := c.state.Players[id]; ok {
		p.Alive = false
		c.state.Players[id] = p
		c.log.Info("kill", id+" est mort")
	}
}

// assignRoles - distribue les rôles
//
// TODO : pour l'instant on distribue en fonction de l'id
// Mais à termes il faudrait faire ça de manière répartie
func (c *Control) assignRoles() {
	ids := make([]string, 0, len(c.state.Players))
	for id := range c.state.Players {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids)-1; i++ {
		for j := i + 1; j < len(ids); j++ {
			if ids[i] > ids[j] {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}

	nWolves := len(ids) / 3
	if nWolves < 1 {
		nWolves = 1
	}

	for i, id := range ids {
		var role Role
		switch {
		case i < nWolves:
			role = RoleWolf
		case i == nWolves:
			role = RoleWitch
		default:
			role = RoleVillager
		}
		p := c.state.Players[id]
		p.Role = role
		c.state.Players[id] = p
		c.log.Info("assignRoles", id+" -> "+string(role))
	}
}

// resolveNight - applique les morts de la nuit, puis passe au vote
func (c *Control) resolveNight() {
	if c.state.KillWolf != "" && c.state.KillWitch != "save:"+c.state.KillWolf {
		c.killPlayer(c.state.KillWolf)
	}
	if c.state.KillWitch != "" && !strings.HasPrefix(c.state.KillWitch, "save:") {
		c.killPlayer(c.state.KillWitch)
	}

	if winner := c.checkWin(); winner != "" {
		c.transitionToEnd(winner)
		return
	}
	c.transitionToVote()
}

// resolveVote - élimine le joueur le plus voté, puis passe à la nuit suivante
func (c *Control) resolveVote() {
	eliminated := topVoted(c.state.Votes)
	if eliminated != "" {
		c.killPlayer(eliminated)
		c.log.Info("vote", eliminated+" éliminé par le village")
	}

	if winner := c.checkWin(); winner != "" {
		c.transitionToEnd(winner)
		return
	}
	c.transitionToNight()
}

// topWolfTarget - retourne la cible la plus votée par les loups
func (c *Control) topWolfTarget() string {
	return topVoted(c.wolfVotes)
}

// topVoted retourne la cible ayant reçu le plus de votes.
// En cas d'égalité, l'ID lexicographiquement minimal est choisi pour garantir
// que tous les sites prennent la même décision (départage déterministe).
func topVoted(votes map[string]string) string {
	if len(votes) == 0 {
		return ""
	}
	count := make(map[string]int)
	for _, target := range votes {
		count[target]++
	}
	candidates := make([]string, 0, len(count))
	for id := range count {
		candidates = append(candidates, id)
	}
	sort.Strings(candidates)
	winner := ""
	max := 0
	for _, id := range candidates {
		if count[id] > max {
			max = count[id]
			winner = id
		}
	}
	return winner
}

// ========= Traitement des messages de l'application (locale) ========= //

func (c *Control) handleFromApp(raw string) {
	c.log.Debug("app->ctrl", raw)
	msgType := transport.Get(raw, "type")
	player := transport.Get(raw, "player")

	switch msgType {
	case "join":
		if _, exists := c.state.Players[player]; !exists {
			c.state.Players[player] = Player{ID: player, Alive: true}
			c.lobbyJoined[player] = true
			c.log.Info("lobby", fmt.Sprintf("%s a rejoint (%d/%d)", player, len(c.lobbyJoined), c.nbSites))
		}
		c.broadcast(transport.Build("type", "join", "player", player))
		c.sendToApp()

	case "ready":
		c.lobbyReady[player] = true
		c.log.Info("lobby", fmt.Sprintf("%s est prêt (%d/%d)", player, len(c.lobbyReady), c.nbSites))
		c.broadcast(transport.Build("type", "ready", "player", player))
		c.tryStartGame()

	case "wolfkill":
		if c.state.Phase != PhaseNight {
			return
		}
		target := transport.Get(raw, "target")
		c.wolfVotes[player] = target
		c.log.Info("wolfkill", fmt.Sprintf("%s -> %s (%d/%d loups)", player, target, len(c.wolfVotes), c.countAliveWolves()))
		c.broadcast(transport.Build("type", "wolfkill", "player", player, "target", target))
		c.tryResolveWolves()

	case "witchsave":
		if c.state.Phase != PhaseWitch || c.witchDone {
			return
		}
		c.state.KillWitch = "save:" + c.state.KillWolf
		c.witchDone = true
		c.log.Info("witch", "sorcière sauve "+c.state.KillWolf)
		c.broadcast(transport.Build("type", "witchsave", "player", player))
		c.tryResolveNight()

	case "witchkill":
		if c.state.Phase != PhaseWitch || c.witchDone {
			return
		}
		target := transport.Get(raw, "target")
		c.state.KillWitch = target
		c.witchDone = true
		c.log.Info("witch", "sorcière empoisonne "+target)
		c.broadcast(transport.Build("type", "witchkill", "player", player, "target", target))
		c.tryResolveNight()

	case "witchpass":
		if c.state.Phase != PhaseWitch || c.witchDone {
			return
		}
		c.witchDone = true
		c.log.Info("witch", "sorcière passe")
		c.broadcast(transport.Build("type", "witchpass", "player", player))
		c.tryResolveNight()

	case "vote":
		if c.state.Phase != PhaseVote {
			return
		}
		target := transport.Get(raw, "target")
		c.state.Votes[player] = target
		c.villageVotes[player] = target
		c.log.Info("vote", fmt.Sprintf("%s -> %s (%d/%d)", player, target, len(c.villageVotes), c.countAlive()))
		c.broadcast(transport.Build("type", "vote", "player", player, "target", target))
		c.tryResolveVote()

	default:
		c.log.Warn("handleFromApp", "type inconnu: "+msgType)
	}
}

// ======== Traitement des messages des autres centres de contrôle ======== //

func (c *Control) handleFromControl(raw string) {
	from := transport.Get(raw, "from")
	if from == c.myID { // on ignore nos propres broadcasts
		return
	}

	msgType := transport.Get(raw, "type")
	c.log.Debug("ctrl->ctrl", fmt.Sprintf("de %s : type=%s", from, msgType))

	// Messages de la file d'attente répartie (cf. queue.go).
	switch msgType {
	case "req":
		h, err := strconv.Atoi(transport.Get(raw, "h"))
		if err != nil {
			c.log.Warn("handleFromControl", "req sans h valide: "+raw)
			return
		}
		c.handleReq(from, h)
		return
	case "rel":
		h, err := strconv.Atoi(transport.Get(raw, "h"))
		if err != nil {
			c.log.Warn("handleFromControl", "rel sans h valide: "+raw)
			return
		}
		c.handleRel(from, h)
		return
	case "ack":
		if transport.Get(raw, "to") != c.myID {
			return // accusé destiné à un autre site
		}
		h, err := strconv.Atoi(transport.Get(raw, "h"))
		if err != nil {
			c.log.Warn("handleFromControl", "ack sans h valide: "+raw)
			return
		}
		c.handleAck(from, h)
		return
	}

	switch msgType {
	case "state":
		// État autoritaire envoyé par un site qui vient de sortir de SC :
		// remplacement direct (pas de merge), en préservant notre MyID local.
		data := transport.Get(raw, "data")
		var remote GameState
		if err := json.Unmarshal([]byte(data), &remote); err != nil {
			c.log.Error("handleFromControl", "unmarshal: "+err.Error())
			return
		}
		remote.MyID = c.myID
		// Synchroniser les trackers locaux (hors GameState) avec la nouvelle phase.
		// Le SC-winner les met à jour dans transitionToX ; les autres sites doivent
		// le faire ici à l'adoption de l'état pour ne pas garder de données stale.
		if remote.Phase != c.state.Phase {
			switch remote.Phase {
			case PhaseNight:
				c.wolfVotes = make(map[string]string)
				c.witchDone = false
			case PhaseVote:
				c.villageVotes = make(map[string]string)
			}
		}
		c.state = remote
		c.sendToApp()

	case "join":
		player := transport.Get(raw, "player")
		if _, exists := c.state.Players[player]; !exists {
			c.state.Players[player] = Player{ID: player, Alive: true}
			c.lobbyJoined[player] = true
			c.log.Info("lobby", "distant: "+player+" a rejoint")
		}
		c.sendToApp()

	case "ready":
		player := transport.Get(raw, "player")
		c.lobbyReady[player] = true
		c.log.Info("lobby", "distant: "+player+" est prêt")
		c.tryStartGame()

	case "wolfkill":
		player := transport.Get(raw, "player")
		target := transport.Get(raw, "target")
		c.wolfVotes[player] = target
		c.tryResolveWolves()

	case "witchsave":
		if !c.witchDone {
			c.state.KillWitch = "save:" + c.state.KillWolf
			c.witchDone = true
			c.tryResolveNight()
		}

	case "witchkill":
		if !c.witchDone {
			target := transport.Get(raw, "target")
			c.state.KillWitch = target
			c.witchDone = true
			c.tryResolveNight()
		}

	case "witchpass":
		if !c.witchDone {
			c.witchDone = true
			c.tryResolveNight()
		}

	case "vote":
		player := transport.Get(raw, "player")
		target := transport.Get(raw, "target")
		c.state.Votes[player] = target
		c.villageVotes[player] = target
		c.tryResolveVote()
	}
}

// ========= Gestion des différentes phases et transitions ========= //

// tryStartGame - démarre la partie si tous les sites sont connectés et prêts.
// Demande la section critique pour que UN seul site exécute assignRoles + transitionToNight.
func (c *Control) tryStartGame() {
	if len(c.lobbyJoined) < c.nbSites || len(c.lobbyReady) < c.nbSites {
		return
	}
	c.requestSC(pendingSC{
		name: "startGame",
		fn: func() {
			// Re-vérification dans la SC : un autre site a peut-être déjà démarré.
			if c.state.Phase != PhaseLobby {
				return
			}
			if len(c.lobbyJoined) < c.nbSites || len(c.lobbyReady) < c.nbSites {
				return
			}
			c.log.Success("lobby", "tous les sites sont prêts -> démarrage !")
			c.assignRoles()
			c.transitionToNight()
		},
	})
}

// tryResolveWolves - passe à la phase sorcière quand tous les loups ont voté.
func (c *Control) tryResolveWolves() {
	alive := c.countAliveWolves()
	if alive == 0 || len(c.wolfVotes) < alive {
		return
	}
	c.requestSC(pendingSC{
		name: "resolveWolves",
		fn: func() {
			if c.state.Phase != PhaseNight {
				return
			}
			n := c.countAliveWolves()
			if n == 0 || len(c.wolfVotes) < n {
				return
			}
			c.state.KillWolf = c.topWolfTarget()
			c.log.Info("wolfkill", "consensus loups -> cible: "+c.state.KillWolf)
			c.transitionToWitch()
		},
	})
}

// tryResolveVote - passe à la nuit suivante quand tous les vivants ont voté.
func (c *Control) tryResolveVote() {
	if len(c.villageVotes) < c.countAlive() {
		return
	}
	c.requestSC(pendingSC{
		name: "resolveVote",
		fn: func() {
			if c.state.Phase != PhaseVote {
				return
			}
			if len(c.villageVotes) < c.countAlive() {
				return
			}
			c.resolveVote()
		},
	})
}

// tryResolveNight - applique les morts de la nuit (transition WITCH → VOTE/END).
// La sorcière a déjà donné son verdict (witchDone == true) avant l'appel.
func (c *Control) tryResolveNight() {
	if c.state.Phase != PhaseWitch || !c.witchDone {
		return
	}
	c.requestSC(pendingSC{
		name: "resolveNight",
		fn: func() {
			if c.state.Phase != PhaseWitch || !c.witchDone {
				return
			}
			c.resolveNight()
		},
	})
}

func (c *Control) marshalState() string {
	data, err := json.Marshal(c.state)
	if err != nil {
		c.log.Error("marshalState", err.Error())
		return "{}"
	}
	return string(data)
}