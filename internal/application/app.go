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

	"github.com/gorilla/websocket"
	"github.com/sr05-projet/internal/server"
	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

type App struct {
	myID   string
	myRole Role
	state  GameState
	io     *transport.IO
	log    *logger.Logger
	addr   string
	port   string
	web    string
	ws     *websocket.Conn
	srv    *server.Server
}

func New(myID string, io *transport.IO, log *logger.Logger, addr string, port string, web string) *App {
	return &App{
		myID:   myID,
		myRole: RoleUnknown,
		state:  NewGameState(myID),
		io:     io,
		log:    log,
		addr:   addr,
		port:   port,
		web:    web,
	}
}

func (a *App) Run() {
	a.log.Info("Run", "démarrage application, joueur="+a.myID)

	a.srv = server.New(a.addr, a.port, a.web, a.log)
	//lance l'écoute pour les connexion
	//upgrade les connextion en web socket
	// la connection a.srv.ws est set a la dernière websocket ouverte
	go a.srv.Run()

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

		// Ici la division doit se faire entre
		// - écouter la websocket du serveur (quand on attend un input utilisateur)
		// - écouter les messages de l'exterieur/les changements d'état
		if strings.HasPrefix(line, "{") {
			a.handleFromBrowser(line)
		} else {
			// a.handleFromControl(line)
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

}
