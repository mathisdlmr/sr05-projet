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
// Ce numéro est utilisé pour le champ Sender de transport.Message qui est int.
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

	// 1. On démarre server.go qui va gérer les connexions WebSocket avec le navigateur
	a.srv = server.New(a.addr, a.port, a.web, a.log)
	go func() {
		if err := a.srv.Run(); err != nil {
			a.log.Error("srv.Run", err.Error())
		}
	}()

	// 2. On démarre une goroutine qui lit stdin en continu et envoie les lignes dans un canal stdinCh
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

	// 3. Boucle principale : on écoute à la fois les messages du contrôle (stdinCh), du
	// navigateur (a.srv.Inbox()) et les nouvelles connexions WebSocket (a.srv.Connects())
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

// --- Gestion de la section critique

// requestCS - demande l'entrée en section critique
func (a *App) requestCS(data map[string]string) {
	if a.pending != nil { // On est déjà en attente d'une SC, on ignore la nouvelle demande
		a.log.Warn("requestCS", "déjà en attente de CS, action ignorée: "+data["cmd"])
		return
	}

	// On stocke les données de l'action en attente, qui seront utilisées à la réception du BeginCS
	a.pending = &pendingData{data: data}
	if err := a.io.Send(transport.Message{
		Type:   transport.TypeApplication,
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
	// Quand on reçoit un BeginCS, ça veut dire que notre demande de section critique a été acceptée,
	// et qu'on peut appliquer l'action en attente (pending) et envoyer un EndCS
	case msg.Type == transport.TypeApplication && msg.Action == transport.ActionBeginCS:
		if a.pending == nil {
			a.log.Warn("handleFromControl", "BeginCS reçu sans action en attente")
			return
		}
		pending := a.pending
		a.pending = nil
		if err := a.io.Send(transport.Message{
			Type:   transport.TypeApplication,
			Action: transport.ActionEndCS,
			Sender: a.siteID,
			Data:   pending.data,
		}.String()); err != nil {
			a.log.Error("handleFromControl", "envoi EndCS: "+err.Error())
		}
		a.log.Info("handleFromControl", "SC accordée, EndCS envoyé: "+pending.data["cmd"])

	// Quand on reçoit une ReleaseCS, ça veut dire qu'une action a été validée par le contrôle (après accord de tous les joueurs),
	// et qu'on peut l'appliquer localement (handleDistributedAction)
	case msg.Type == transport.TypeControl && msg.Action == transport.ActionReleaseCS:
		a.log.Info("handleFromControl", "ReleaseCS reçu, action: "+msg.Data["cmd"])
		a.handleDistributedAction(msg.Data)
	}
}

// --- Actions propres au jeu (distribuées) --- //

func (a *App) handleDistributedAction(data map[string]string) {
	voterID := data["voter"]
	targetID := data["target"]

	switch data["cmd"] {
	case "join":
		a.applyJoin(data["player"])

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
			a.transitionToNight()
		}

	default:
		a.log.Warn("handleDistributedAction", "action inconnue: "+data["cmd"])
	}
}

// applyJoin - ajoute un joueur à l'état local, notifie le navigateur et les autres joueurs via une SC "join"
func (a *App) applyJoin(playerID string) {
	if _, ok := a.state.Players[playerID]; ok {
		return
	}
	a.state.Players[playerID] = Player{ID: playerID, Role: RoleUnknown, Alive: true}
	a.pushEvent(map[string]interface{}{
		"type":     "playerJoined",
		"playerId": playerID,
	})
	a.log.Info("applyJoin", "joueur rejoint: "+playerID)
}

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

// Used only at Vote Phase
// applyVoteResult - élimine le joueur le plus voté
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

	a.log.Info("applyVoteResult", "vote résolu")
}

// sendGameEnd - calcule le vainqueur, met à jour l'état et notifie le navigateur
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

// killPlayer - met à jour l'état pour marquer un joueur comme éliminé
func (a *App) killPlayer(targetID string) {
	if p, ok := a.state.Players[targetID]; ok {
		p.Alive = false
		a.state.Players[targetID] = p
		a.log.Info("killPlayer", "joueur éliminé: "+targetID)
	}

}

// computeVoteResults - calcule le joueur le plus voté à partir de a.state.Votes
// Retourne le playerID du plus voté et un booléen indiquant si le résultat est valide (non nul)
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

// checkEndOfGame - vérifie si la partie est terminée (tous les loups sont morts ou les loups ont l'avantage numérique)
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

// checkAllVotesCompleted - vérifie si tous les joueurs qui doivent voter ont voté (aucune valeur vide dans a.state.Votes)
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

// createStartingVoteMap - initialise a.state.Votes avec les joueurs qui doivent voter pour la phase donnée, avec des votes vides
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

// ========= Messages venant du navigateur ========= //

type browserAction struct {
	Action string `json:"action"`
	Target string `json:"target,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

// handleFromBrowser - appelé à chaque message JSON reçu du navigateur
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
