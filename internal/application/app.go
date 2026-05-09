// Package application - Contient la logique du jeu et le filtrage entre le
// client (server) et le système réparti (control).
//
// app.go - dispatcher central :
//   - lit stdin pour les messages venant du contrôle local (format /=type=...)
//   - lit le canal Inbox du serveur pour les messages JSON venant du navigateur
//   - écrit sur stdout pour parler au contrôle local
//   - appelle Server.Send pour parler au navigateur

package application

import (
	"encoding/json"
	stdio "io"
	"sort"

	"github.com/sr05-projet/internal/server"
	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

const (
	NullPlayerID string = "NULLPLAYERID"
)

type pendingData struct {
	data map[string]string
}

type App struct {
	myID    string
	siteID  int
	myRole  Role
	state   GameState
	io      *transport.IO
	log     *logger.Logger
	addr    string
	port    string
	web     string
	srv     *server.Server
	pending *pendingData
}

func New(myID string, io *transport.IO, log *logger.Logger, addr string, port string, web string) *App {
	return &App{
		myID:   myID,
		siteID: parseSiteID(myID),
		myRole: RoleUnknown,
		state:  NewGameState(myID),
		io:     io,
		log:    log,
		addr:   addr,
		port:   port,
		web:    web,
	}
}

// parseSiteID - extrait le numéro à partir d'un id du style "J3" -> 3.
func parseSiteID(id string) int {
	n := 0
	for _, c := range id {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func (a *App) Run() {
	a.log.Info("Run", "démarrage application, joueur="+a.myID)

	a.srv = server.New(a.addr, a.port, a.web, a.log)
	go func() {
		if err := a.srv.Run(); err != nil {
			a.log.Error("srv.Run", err.Error())
		}
	}()

	stdinCh := make(chan string, 16)
	go func() {
		for {
			line, err := a.io.ReadLine()
			if err == stdio.EOF {
				close(stdinCh)
				return
			}
			if err != nil {
				a.log.Error("Run", "lecture stdin: "+err.Error())
				close(stdinCh)
				return
			}
			stdinCh <- line
		}
	}()

	for {
		select {
		case line, ok := <-stdinCh:
			if !ok {
				a.log.Info("Run", "stdin fermé, arrêt")
				return
			}
			a.handleFromControl(line)
		case raw := <-a.srv.Inbox():
			a.handleFromBrowser(raw)
		case <-a.srv.Connects():
			a.handleBrowserConnect()
		}
	}
}

// --- Connexion navigateur ---

// handleBrowserConnect - appelé à chaque nouvelle connexion WebSocket depuis le navigateur
// Ajoute le joueur à l'état local s'il n'existe pas déjà, envoie l'init et notifie les autres joueurs via une SC "join"
func (a *App) handleBrowserConnect() {
	a.log.Info("handleBrowserConnect", "navigateur connecté")
	if _, ok := a.state.Players[a.myID]; !ok {
		a.state.Players[a.myID] = Player{ID: a.myID, Role: RoleUnknown, Alive: true}
	}
	a.sendInit()
	a.requestCS(map[string]string{
		"cmd":    "join",
		"player": a.myID,
	})
}

// sendInit - envoie l'état du jeu au navigateur
func (a *App) sendInit() {
	p, ok := a.state.Players[a.myID]
	if !ok {
		p = Player{ID: a.myID, Alive: true}
	}
	evt := map[string]interface{}{
		"type":    "init",
		"phase":   string(a.state.Phase),
		"myId":    a.myID,
		"myRole":  string(a.myRole),
		"myAlive": p.Alive,
		"players": a.buildFilteredPlayers(),
		"votes":   a.getVisibleVotes(),
	}
	if a.state.Phase == PhaseWitch && a.myRole == RoleWitch && a.state.KillWolf != "" {
		evt["killWolf"] = a.state.KillWolf
	}
	a.pushEvent(evt)
}

// buildFilteredPlayers - construit la vue filtrée des joueurs selon le rôle local
func (a *App) buildFilteredPlayers() map[string]interface{} {
	result := make(map[string]interface{})
	for id, p := range a.state.Players {
		visibleRole := "?"
		switch {
		case id == a.myID:
			visibleRole = string(p.Role)
		case a.myRole == RoleWolf && p.Role == RoleWolf:
			visibleRole = string(p.Role)
		case a.state.Phase == PhaseEnd:
			visibleRole = string(p.Role)
		}
		result[id] = map[string]interface{}{
			"id":    id,
			"role":  visibleRole,
			"alive": p.Alive,
		}
	}
	return result
}

// getVisibleVotes - retourne les votes visibles pour ce joueur (sans les vides)
func (a *App) getVisibleVotes() map[string]string {
	result := map[string]string{}
	switch a.state.Phase {
	case PhaseNight:
		if a.myRole == RoleWolf {
			for k, v := range a.state.Votes {
				if v != "" {
					result[k] = v
				}
			}
		}
	case PhaseVote:
		for k, v := range a.state.Votes {
			if v != "" {
				result[k] = v
			}
		}
	}
	return result
}

// pushEvent - envoie un event (une map convertie en JSON) au navigateur
func (a *App) pushEvent(evt map[string]interface{}) {
	out, err := json.Marshal(evt)
	if err != nil {
		a.log.Error("pushEvent", "marshal: "+err.Error())
		return
	}
	if err := a.srv.Send(string(out)); err != nil {
		a.log.Warn("pushEvent", "send: "+err.Error())
	}
}

// --- Gestion de la section critique

// requestCS - demande l'entrée en section critique
func (a *App) requestCS(data map[string]string) {
	if a.pending != nil {
		a.log.Warn("requestCS", "déjà en attente de CS, action ignorée: "+data["cmd"])
		return
	}
	a.pending = &pendingData{data: data}
	if err := a.io.Send(transport.Message{
		Type:   transport.Application,
		Action: transport.ActionRequestCS,
		Sender: a.siteID,
	}.String()); err != nil {
		a.log.Error("requestCS", "envoi RequestCS: "+err.Error())
	}
	a.log.Info("requestCS", "demande de SC pour: "+data["cmd"])
}

// --- Messages venant du contrôle (format /=type=...) ---

func (a *App) handleFromControl(line string) {
	a.log.Debug("ctrl->app", line)

	msg, err := transport.ParseMessage(line)
	if err != nil {
		a.log.Warn("handleFromControl", "parse: "+err.Error())
		return
	}

	switch {
	case msg.Type == transport.Application && msg.Action == transport.ActionBeginCS:
		if a.pending == nil {
			a.log.Warn("handleFromControl", "BeginCS reçu sans action en attente")
			return
		}
		pending := a.pending
		a.pending = nil
		if err := a.io.Send(transport.Message{
			Type:   transport.Application,
			Action: transport.ActionEndCS,
			Sender: a.siteID,
			Data:   pending.data,
		}.String()); err != nil {
			a.log.Error("handleFromControl", "envoi EndCS: "+err.Error())
		}
		a.log.Info("handleFromControl", "SC accordée, EndCS envoyé: "+pending.data["cmd"])

	case msg.Type == transport.Control && msg.Action == transport.ActionReleaseCS:
		a.log.Info("handleFromControl", "ReleaseCS reçu, action: "+msg.Data["cmd"])
		a.handleDistributedAction(msg.Data)
	}
}

// --- Actions propres au jeu (distribuées) --- //

func (a *App) handleDistributedAction(data map[string]string) {
	voterID := data["voter"]
	targetID := data["target"]

	switch data["cmd"] {
	// case "join":
	// 	a.applyJoin(data["player"])

	case "start":
		a.applyStart()

	case "wolfkill":
		if a.state.Phase != PhaseNight {
			a.log.Warn("handleDistributedAction", "wolfkill ignoré hors phase NIGHT")
			return
		}
		a.state.Votes[voterID] = targetID
		evt := map[string]interface{}{
			"type":  "wolfVoted",
			"voter": voterID,
		}
		if a.myRole == RoleWolf {
			evt["target"] = targetID
		}
		a.pushEvent(evt)
		if a.checkAllVotesCompleted() {
			target, valid := a.computeVoteResults()
			if valid {
				a.state.KillWolf = target
			}
			a.transitionToWitch()
		}

	case "witchsave":
		if a.state.Phase != PhaseWitch {
			a.log.Warn("handleDistributedAction", "witchsave ignoré hors phase WITCH")
			return
		}
		a.state.KillWolf = ""
		a.transitionToVote()

	case "witchkill":
		if a.state.Phase != PhaseWitch {
			a.log.Warn("handleDistributedAction", "witchkill ignoré hors phase WITCH")
			return
		}
		a.state.KillWitch = targetID
		a.transitionToVote()

	case "witchskip":
		if a.state.Phase != PhaseWitch {
			a.log.Warn("handleDistributedAction", "witchskip ignoré hors phase WITCH")
			return
		}
		a.transitionToVote()

	case "vote":
		if a.state.Phase != PhaseVote {
			a.log.Warn("handleDistributedAction", "vote ignoré hors phase VOTE")
			return
		}
		a.state.Votes[voterID] = targetID
		a.pushEvent(map[string]interface{}{
			"type":   "voted",
			"voter":  voterID,
			"target": targetID,
		})
		if a.checkAllVotesCompleted() {
			a.applyVoteResult()
		}

	default:
		a.log.Warn("handleDistributedAction", "action inconnue: "+data["cmd"])
	}
}

// applyJoin - ajoute un joueur à l'état local, notifie le navigateur et les autres joueurs via une SC "join"
// func (a *App) applyJoin(playerID string) {
// 	if _, ok := a.state.Players[playerID]; ok {
// 		return
// 	}
// 	a.state.Players[playerID] = Player{ID: playerID, Role: RoleUnknown, Alive: true}
// 	a.pushEvent(map[string]interface{}{
// 		"type":     "playerJoined",
// 		"playerId": playerID,
// 	})
// 	a.log.Info("applyJoin", "joueur rejoint: "+playerID)
// }

// applyStart - distribue les rôles, initialise les votes et passe en phase NIGHT
func (a *App) applyStart() {
	if a.state.Phase != PhaseLobby {
		a.log.Warn("applyStart", "start ignoré hors phase LOBBY")
		return
	}

	playerIDs := make([]string, 0, len(a.state.Players))
	for id := range a.state.Players {
		playerIDs = append(playerIDs, id)
	}
	sort.Strings(playerIDs)

	n := len(playerIDs)
	nWolves := n / 3 // 1/3 des joueurs sont des loups
	if nWolves == 0 {
		nWolves = 1
	}

	for i, id := range playerIDs {
		p := a.state.Players[id]
		switch {
		case i < nWolves:
			p.Role = RoleWolf
		case i == nWolves:
			p.Role = RoleWitch
		default:
			p.Role = RoleVillager
		}
		a.state.Players[id] = p
	}

	a.myRole = a.state.Players[a.myID].Role
	a.createStartingVoteMap(PhaseNight)
	a.state.Phase = PhaseNight

	a.pushEvent(map[string]interface{}{
		"type":    "gameStart",
		"myRole":  string(a.myRole),
		"players": a.buildFilteredPlayers(),
	})
	a.log.Info("applyStart", "partie démarrée, rôle local: "+string(a.myRole))
}

// transitionToWitch - passe en phase WITCH et notifie le navigateur
func (a *App) transitionToWitch() {
	a.state.Phase = PhaseWitch
	a.state.Votes = make(map[string]string)

	evt := map[string]interface{}{
		"type":  "phaseChange",
		"phase": "WITCH",
	}
	if a.myRole == RoleWitch && a.state.KillWolf != "" {
		evt["killWolf"] = a.state.KillWolf
	}
	a.pushEvent(evt)

	// Si la sorcière est morte, on passe directement au vote du village
	witchAlive := false
	for _, p := range a.state.Players {
		if p.Role == RoleWitch && p.Alive {
			witchAlive = true
			break
		}
	}
	if !witchAlive {
		a.log.Info("transitionToWitch", "sorcière morte, auto-passage à VOTE")
		a.transitionToVote()
	}
}

// transitionToVote - applique les kills de la nuit et passe en phase VOTE
func (a *App) transitionToVote() {
	killed := []string{}
	if a.state.KillWolf != "" {
		a.killPlayer(a.state.KillWolf)
		killed = append(killed, a.state.KillWolf)
	}
	if a.state.KillWitch != "" {
		a.killPlayer(a.state.KillWitch)
		killed = append(killed, a.state.KillWitch)
	}
	a.state.KillWolf = ""
	a.state.KillWitch = ""

	if a.checkEndOfGame() {
		a.pushEvent(map[string]interface{}{
			"type":      "nightKills",
			"killed":    killed,
			"nextPhase": "END",
		})
		a.sendGameEnd()
		return
	}

	a.createStartingVoteMap(PhaseVote)
	a.state.Phase = PhaseVote

	a.pushEvent(map[string]interface{}{
		"type":      "nightKills",
		"killed":    killed,
		"nextPhase": "VOTE",
	})
	a.log.Info("transitionToVote", "passage en phase VOTE")
}

// applyVoteResult - élimine le joueur le plus voté et passe à la phase suivante
func (a *App) applyVoteResult() {
	target, valid := a.computeVoteResults()

	if valid {
		a.killPlayer(target)
	}

	if a.checkEndOfGame() {
		if valid {
			a.pushEvent(map[string]interface{}{
				"type":      "voteEliminated",
				"playerId":  target,
				"nextPhase": "END",
			})
		}
		a.sendGameEnd()
		return
	}

	if valid {
		a.pushEvent(map[string]interface{}{
			"type":      "voteEliminated",
			"playerId":  target,
			"nextPhase": "NIGHT",
		})
	}

	a.createStartingVoteMap(PhaseNight)
	a.state.Phase = PhaseNight
	a.log.Info("applyVoteResult", "vote résolu, passage en phase NIGHT")
}

func (a *App) sendGameEnd() {
	allWolvesDead := true
	for _, p := range a.state.Players {
		if p.Alive && p.Role == RoleWolf {
			allWolvesDead = false
			break
		}
	}

	winner := "VILLAGERS"
	if !allWolvesDead {
		winner = "WOLVES"
	}

	a.state.Phase = PhaseEnd
	a.state.Winner = winner

	allPlayers := make(map[string]interface{})
	for id, p := range a.state.Players {
		allPlayers[id] = map[string]interface{}{
			"id":    id,
			"role":  string(p.Role),
			"alive": p.Alive,
		}
	}

	a.pushEvent(map[string]interface{}{
		"type":    "gameEnd",
		"winner":  winner,
		"players": allPlayers,
	})
	a.log.Info("sendGameEnd", "fin de partie, vainqueur: "+winner)
}

func (a *App) killPlayer(targetID string) {
	if p, ok := a.state.Players[targetID]; ok {
		p.Alive = false
		a.state.Players[targetID] = p
		a.log.Info("killPlayer", "joueur éliminé: "+targetID)
	}
}

func (a *App) computeVoteResults() (string, bool) {
	scores := map[string]int{}
	for _, target := range a.state.Votes {
		if target == "" {
			continue
		}
		scores[target]++
	}

	maxValue := 0
	maxTarget := ""
	for target, value := range scores {
		if value > maxValue {
			maxValue = value
			maxTarget = target
		}
	}

	if maxTarget == NullPlayerID || maxTarget == "" {
		return "", false
	}
	return maxTarget, true
}

func (a *App) checkEndOfGame() bool {
	allWolvesDead := true
	for _, p := range a.state.Players {
		if p.Alive && p.Role == RoleWolf {
			allWolvesDead = false
			break
		}
	}
	if allWolvesDead {
		return true
	}

	nbWolves, nbVillagers := 0, 0
	for _, p := range a.state.Players {
		if !p.Alive {
			continue
		}
		if p.Role == RoleWolf {
			nbWolves++
		} else {
			nbVillagers++
		}
	}
	return nbWolves >= nbVillagers
}

func (a *App) checkAllVotesCompleted() bool {
	if len(a.state.Votes) == 0 {
		return false
	}
	for _, vote := range a.state.Votes {
		if vote == "" {
			return false
		}
	}
	return true
}

func (a *App) createStartingVoteMap(phase Phase) {
	var votersRole Role
	switch phase {
	case PhaseNight:
		votersRole = RoleWolf
	case PhaseVote:
		votersRole = RoleAny
	case PhaseWitch:
		votersRole = RoleWitch
	}

	a.state.Votes = make(map[string]string)
	for playerID, player := range a.state.Players {
		if !player.Alive {
			continue
		}
		if player.Role == votersRole || votersRole == RoleAny {
			a.state.Votes[playerID] = ""
		}
	}
}

// --- Messages venant du navigateur --- //

type browserAction struct {
	Action string `json:"action"`
	Target string `json:"target,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

func (a *App) handleFromBrowser(raw string) {
	a.log.Debug("browser->app", raw)

	var action browserAction
	if err := json.Unmarshal([]byte(raw), &action); err != nil {
		a.log.Warn("handleFromBrowser", "parse: "+err.Error())
		return
	}
	if action.Action == "" {
		a.log.Warn("handleFromBrowser", "action vide, ignoré")
		return
	}
	if action.Action == "init" {
		a.sendInit()
		return
	}
	data := map[string]string{
		"cmd":   action.Action,
		"voter": a.myID,
	}
	if action.Target != "" {
		data["target"] = action.Target
	}
	a.requestCS(data)
	a.log.Info("handleFromBrowser", "SC demandée pour: "+action.Action)
}
