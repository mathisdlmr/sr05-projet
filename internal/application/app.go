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
		}
	}
}

// --- Connexion navigateur ---

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
		//a.handleDistributedAction(msg.Data)
	}
}

func (a *App) computeVoteResults() (string, bool) {
	scores := map[string]int{}

	for _, target := range a.state.Votes {
		if _, ok := scores[target]; ok {
			scores[target] += 1
		} else {
			scores[target] = 1
		}
	}

	max_value := 0
	max_target := ""
	for target, value := range scores {
		// Cas d'égalité possible, à régler
		if value > max_value {
			max_value = value
			max_target = target
		}
	}

	if max_target == NullPlayerID {
		return NullPlayerID, false
	}

	return max_target, true
}

func (a *App) handleVote(voterID string, targetID string) {
	a.state.Votes[voterID] = targetID

	if a.checkAllVotesCompleted() {
		target, validtarget := a.computeVoteResults()
		a.applyVoteResults(target, validtarget) // kills target of vote imediatly after PhaseVote
		if a.state.Phase == PhaseWitch {
			a.applyNightKills()
		}

		if a.checkEndOfGame() {
			// end game here
			return
		}
		a.switchPhase(a.getNextPhase())
	} else {
		//send vote to browser comme ça il peut l'afficher
		a.srv.PushMessageToBrowser(
			server.BrowserMessage{
				Action: "vote",
				Data: map[string]string{
					"voter":  voterID,
					"target": targetID,
				}})
	}
}

func (a *App) switchPhase(next_phase Phase) {

	var message server.BrowserMessage

	switch a.state.Phase {
	case PhaseNight:
		message.Action = "switch_to_witch"
		if a.myRole == RoleWitch {
			message.Data["kill"] = a.state.KillWolf
		}
	case PhaseWitch:
		message.Action = "switch_to_vote"
		message.Data["kill1"] = a.state.KillWitch // Attention trouver un moyen de melanger
		message.Data["kill2"] = a.state.KillWolf
		message.Data["role1"] = string(a.state.Players[a.state.KillWitch].Role)
		message.Data["role2"] = string(a.state.Players[a.state.KillWolf].Role)
	case PhaseVote:
		message.Action = "switch_to_night"
		killed, _ := a.computeVoteResults()
		message.Data["kill"] = killed
	case PhaseLobby:
		message.Action = "switch_to_night"
		message.Data["members"] = "MEMBERS" // A compléter
		message.Data["YourRole"] = string(a.myRole)
	}
	a.srv.PushMessageToBrowser(message)

	//Reset les votes et les kills
	a.createStartingVoteMap(next_phase)

	a.state.KillWitch = ""
	a.state.KillWolf = ""
	a.state.Phase = next_phase

}

func (a *App) applyVoteResults(targetID string, validtarget bool) {
	if !validtarget {
		// No valid target, we do nothing
		return
	}

	switch a.state.Phase {
	case PhaseNight:
		a.state.KillWolf = targetID
	case PhaseWitch:
		if targetID == a.state.KillWolf { // C'est un
			a.state.KillWolf = ""
		} else {
			a.state.KillWitch = targetID
		}
	case PhaseVote:
		a.killPlayer(targetID)
	}
}

func (a *App) applyNightKills() {
	if a.state.KillWitch != "" {
		a.killPlayer(a.state.KillWitch)
	}

	if a.state.KillWolf != "" {
		a.killPlayer(a.state.KillWolf)
	}

}

func (a *App) getNextPhase() Phase {
	switch a.state.Phase {
	case PhaseLobby:
		return PhaseNight
	case PhaseNight:
		return PhaseWitch
	case PhaseWitch:
		return PhaseVote
	default:
		return PhaseNight
	}

}

// Suppose qu'un player ne "meurt" réellement qu'a la fin du vote du village ou a la fin du tour de la sorcière.
// En attendant les morts sont stockés dans les kills
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
	for _, vote := range a.state.Votes {
		if vote != "" {
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

// ========= Messages venant du navigateur ========= //

type browserAction struct {
	Action string `json:"action"`
	Target string `json:"target,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

func (a *App) handleFromBrowser(raw string) {
	a.log.Debug("browser->app", raw)

	var action browserAction
	if err := json.Unmarshal([]byte(raw), &action); err != nil {
		// Saisie texte libre : on la transforme en action "chat" pour la
		// faire transiter via le mécanisme normal (utile pour debug).
		action = browserAction{Action: "chat", Msg: raw}
	}
	if action.Action == "" {
		a.log.Warn("handleFromBrowser", "action vide, ignoré")
		return
	}

	data := map[string]string{"action": action.Action}
	if action.Target != "" {
		data["target"] = action.Target
	}
	if action.Msg != "" {
		data["msg"] = action.Msg
	}

	msg := transport.Message{
		Type:   transport.Application,
		Sender: a.siteID,
		Data:   data,
	}
	if err := a.io.Send(msg.String()); err != nil {
		a.log.Error("handleFromBrowser", "envoi contrôle: "+err.Error())
		return
	}
	a.log.Info("handleFromBrowser", "envoyé au contrôle: "+msg.String())
}
