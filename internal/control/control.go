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
}

func New(myID string, nbSites int, io *transport.IO, log *logger.Logger) *Control {
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
	count := make(map[string]int)
	for _, target := range c.state.Votes {
		count[target]++
	}
	var eliminated string
	max := 0
	for id, n := range count {
		if n > max {
			max = n
			eliminated = id
		}
	}
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
	count := make(map[string]int)
	for _, t := range c.wolfVotes {
		count[t]++
	}
	max := 0
	target := ""
	for t, n := range count {
		if n > max {
			max = n
			target = t
		}
	}
	return target
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
		c.resolveNight()

	case "witchkill":
		if c.state.Phase != PhaseWitch || c.witchDone {
			return
		}
		target := transport.Get(raw, "target")
		c.state.KillWitch = target
		c.witchDone = true
		c.log.Info("witch", "sorcière empoisonne "+target)
		c.broadcast(transport.Build("type", "witchkill", "player", player, "target", target))
		c.resolveNight()

	case "witchpass":
		if c.state.Phase != PhaseWitch || c.witchDone {
			return
		}
		c.witchDone = true
		c.log.Info("witch", "sorcière passe")
		c.broadcast(transport.Build("type", "witchpass", "player", player))
		c.resolveNight()

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

	switch msgType {
	case "state": // TODO : je savais pas quoi faire, donc si on reçoit un state pour l'instant on fusionne
		data := transport.Get(raw, "data")
		var remote GameState
		if err := json.Unmarshal([]byte(data), &remote); err != nil {
			c.log.Error("handleFromControl", "unmarshal: "+err.Error())
			return
		}
		c.mergeState(remote)
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
			c.resolveNight()
		}

	case "witchkill":
		if !c.witchDone {
			target := transport.Get(raw, "target")
			c.state.KillWitch = target
			c.witchDone = true
			c.resolveNight()
		}

	case "witchpass":
		if !c.witchDone {
			c.witchDone = true
			c.resolveNight()
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

// tryStartGame - démarre la partie si tous les sites sont connectés et prêts
func (c *Control) tryStartGame() {
	if len(c.lobbyJoined) == c.nbSites && len(c.lobbyReady) == c.nbSites {
		c.log.Success("lobby", "tous les sites sont prêts -> démarrage !")
		c.assignRoles()
		c.transitionToNight()
	}
}

// tryResolveWolves - passe à la phase sorcière quand tous les loups ont voté
func (c *Control) tryResolveWolves() {
	if len(c.wolfVotes) >= c.countAliveWolves() && c.countAliveWolves() > 0 {
		c.state.KillWolf = c.topWolfTarget()
		c.log.Info("wolfkill", "consensus loups -> cible: "+c.state.KillWolf)
		c.transitionToWitch()
	}
}

// tryResolveVote - passe à la nuit suivante quand tous les vivants ont voté
func (c *Control) tryResolveVote() {
	if len(c.villageVotes) >= c.countAlive() {
		c.resolveVote()
	}
}

// ========= Merge de réplicas ========= //

// mergeState - intègre un état distant dans le réplica local (pour l'instant on n'ajoute que les joueurs et votes inconnus)
//
// TODO : j'ai aucune idée de comment on est censés gérer ça proprement ...
// Peut-être faire comme on avait dit avec aucun merge mais juste l'envoie d'actions ?
func (c *Control) mergeState(remote GameState) {
	for id, p := range remote.Players {
		if _, exists := c.state.Players[id]; !exists {
			c.state.Players[id] = p
			c.lobbyJoined[id] = true
		}
	}
	for voter, target := range remote.Votes {
		if _, exists := c.state.Votes[voter]; !exists {
			c.state.Votes[voter] = target
			c.villageVotes[voter] = target
		}
	}
	phaseOrder := map[Phase]int{
		PhaseLobby: 0,
		PhaseNight: 1,
		PhaseWitch: 2,
		PhaseVote:  3,
		PhaseEnd:   4,
	}
	if phaseOrder[remote.Phase] > phaseOrder[c.state.Phase] { // L'idée est d'adopter la phase distante si elle est plus avancée
		c.state.Phase = remote.Phase
		c.state.KillWolf = remote.KillWolf
		c.state.Winner = remote.Winner
	}
}

func (c *Control) marshalState() string {
	data, err := json.Marshal(c.state)
	if err != nil {
		c.log.Error("marshalState", err.Error())
		return "{}"
	}
	return string(data)
}