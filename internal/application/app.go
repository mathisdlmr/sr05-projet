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

type App struct {
	myID   string
	siteID int
	myRole Role
	state  GameState
	io     *transport.IO
	log    *logger.Logger
	addr   string
	port   string
	web    string
	srv    *server.Server
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

// ========= Messages venant du contrôle (format /=type=...) ========= //

func (a *App) handleFromControl(line string) {
	a.log.Debug("ctrl->app", line)

	msg, err := transport.ParseMessage(line)
	if err != nil {
		a.log.Warn("handleFromControl", "parse: "+err.Error())
		return
	}

	switch msg.Type {
	case transport.ControlMessage:
		// Message applicatif estampillé qui circule entre les sites :
		// on le pousse au navigateur pour visualisation.
		a.pushEventToBrowser(msg)
	default:
		// Tout le reste (CriticalSection, types inconnus) ne concerne pas l'app.
		a.log.Debug("handleFromControl", "type ignoré: "+msg.Type)
	}
}

// pushEventToBrowser - sérialise un événement en JSON et le pousse via la WS.
func (a *App) pushEventToBrowser(msg *transport.Message) {
	payload := struct {
		Type      string            `json:"type"`
		From      int               `json:"from"`
		Timestamp *int              `json:"timestamp,omitempty"`
		Data      map[string]string `json:"data"`
	}{
		Type:      "event",
		From:      msg.Sender,
		Timestamp: msg.Timestamp,
		Data:      msg.Data,
	}
	out, err := json.Marshal(payload)
	if err != nil {
		a.log.Error("pushEventToBrowser", "marshal: "+err.Error())
		return
	}
	if err := a.srv.Send(string(out)); err != nil {
		a.log.Warn("pushEventToBrowser", "send: "+err.Error())
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
		Type:   transport.DataMessage,
		Sender: a.siteID,
		Data:   data,
	}
	if err := a.io.Send(msg.String()); err != nil {
		a.log.Error("handleFromBrowser", "envoi contrôle: "+err.Error())
		return
	}
	a.log.Info("handleFromBrowser", "envoyé au contrôle: "+msg.String())
}
