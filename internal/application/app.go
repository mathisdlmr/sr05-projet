// Package application - Contient la logique du jeu, et le filtrage entre le client (server) et le système réparti (control)
// app.go - contient le processus applicatif, qui reçoit les messages du navigateur (JSON) et du centre de contrôle (type=state, type=error), 
// maintient l'état de jeu local, et envoie les actions du joueur vers le centre de contrôle, ainsi que la vue filtrée vers le navigateur
//
// Communication :
// * Messages du navigateur -> reçus sur stdin en JSON
// * Messages du centre de contrôle -> reçus sur stdin au format "/=type=state/=data={}"
// * Messages vers le centre de contrôle -> envoyés sur stdout au format "/=type=.../=..."
// * Messages vers le navigateur -> envoyés sur stdout au format "/=type=state/=data={...}"

package application

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

type App struct {
	myID   string
	myRole Role
	state  GameState
	io     *transport.IO
	log    *logger.Logger
}

func New(myID string, io *transport.IO, log *logger.Logger) *App {
	return &App{
		myID:   myID,
		myRole: RoleUnknown,
		state:  NewGameState(myID),
		io:     io,
		log:    log,
	}
}

func (a *App) Run() {
	a.log.Info("Run", "démarrage application, joueur="+a.myID)
	for {
		line, err := a.io.ReadLine()
		if err == io.EOF {
			a.log.Info("Run", "stdin fermé, arrêt")
			return
		}
		if err != nil {
			a.log.Error("Run", "lecture stdin: "+err.Error())
			return
		}

		if strings.HasPrefix(line, "{") {
			a.handleFromBrowser(line)
		} else {
			a.handleFromControl(line)
		}
	}
}

// ========= Messages venant du navigateur (JSON) ========= //

type browserAction struct {
	Action string `json:"action"`
	Target string `json:"target,omitempty"`
}

func (a *App) handleFromBrowser(raw string) {
	a.log.Debug("browser->ctrl", raw)

	var action browserAction
	if err := json.Unmarshal([]byte(raw), &action); err != nil {
		a.log.Warn("handleFromBrowser", "JSON invalide: "+err.Error())
		return
	}

	switch action.Action {
	case "join":
		a.io.MustSend(transport.Build("type", "join", "player", a.myID))

	case "ready":
		a.io.MustSend(transport.Build("type", "ready", "player", a.myID))

	case "vote":
		if a.state.Phase != PhaseVote {
			a.log.Warn("handleFromBrowser", "vote ignoré : mauvaise phase ("+string(a.state.Phase)+")")
			return
		}
		a.io.MustSend(transport.Build("type", "vote", "player", a.myID, "target", action.Target))

	case "wolfkill":
		if a.state.Phase != PhaseNight || a.myRole != RoleWolf {
			a.log.Warn("handleFromBrowser", "wolfkill ignoré")
			return
		}
		a.io.MustSend(transport.Build("type", "wolfkill", "player", a.myID, "target", action.Target))

	case "witchsave":
		if a.state.Phase != PhaseWitch || a.myRole != RoleWitch {
			return
		}
		a.io.MustSend(transport.Build("type", "witchsave", "player", a.myID))

	case "witchkill":
		if a.state.Phase != PhaseWitch || a.myRole != RoleWitch {
			return
		}
		a.io.MustSend(transport.Build("type", "witchkill", "player", a.myID, "target", action.Target))

	case "witchpass":
		if a.state.Phase != PhaseWitch || a.myRole != RoleWitch {
			return
		}
		a.io.MustSend(transport.Build("type", "witchpass", "player", a.myID))

	default:
		a.log.Warn("handleFromBrowser", "action inconnue: "+action.Action)
	}
}

// ====== Messages venant du controle (/=type=state/=data={...}) ======= //

func (a *App) handleFromControl(raw string) {
	a.log.Debug("ctrl->browser", raw)

	msgType := transport.Get(raw, "type")

	switch msgType {
	case "state":
		data := transport.Get(raw, "data")
		var newState GameState
		if err := json.Unmarshal([]byte(data), &newState); err != nil {
			a.log.Error("handleFromControl", "unmarshal state: "+err.Error())
			return
		}
		a.state = newState

		if p, ok := a.state.Players[a.myID]; ok {
			a.myRole = p.Role
		}

		a.sendStateToBrowser()

	case "error":
		msg := transport.Get(raw, "msg")
		a.io.MustSend(transport.Build("type", "error", "msg", msg))

	default:
		a.log.Warn("handleFromControl", "type inconnu: "+msgType)
	}
}

// sendStateToBrowser - construit la vue filtrée, la sérialise en JSON et l'envoie au server (stdout)
func (a *App) sendStateToBrowser() {
	view := BuildView(a.state, a.myID, a.myRole)

	data, err := json.Marshal(view)
	if err != nil {
		a.log.Error("sendStateToBrowser", "marshal: "+err.Error())
		return
	}

	msg := transport.Build("type", "state", "data", string(data))
	a.log.Debug("sendStateToBrowser", msg)
	a.io.MustSend(msg)
}