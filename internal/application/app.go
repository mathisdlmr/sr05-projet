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
	stdio "io"

	"github.com/sr05-projet/internal/server"
	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
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

// --- Demande de section critique ---

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

	if msg.Type != transport.TypeApplication {
		return
	}

	switch msg.Action {
	// Snapshot : le Control nous demande notre état pour la prise d'instantané (algo 11)
	case transport.ActionSnapshotState:
		if msg.Data["role"] == "request" {
			a.handleSnapshotStateRequest()
		}
		// Les réponses (role=response) sont émises par nous, pas reçues, donc on ignore.
		return

	// Snapshot : le Control nous transmet l'EG final pour affichage navigateur
	case transport.ActionRestoreSnapshot:
		a.handleSnapshotRestore(msg.Data["eg"])
		return

	// Snapshot refusé par le Control (un autre est déjà en cours)
	case transport.ActionSnapshotRejected:
		a.pushEvent(map[string]interface{}{
			"type":   "snapshot_rejected",
			"reason": msg.Data["reason"],
		})
		a.log.Warn("handleFromControl", "snapshot refusé : "+msg.Data["reason"])
		return
		
	// Quand on reçoit un BeginCS, ça veut dire que notre demande de section critique a été acceptée,
	// et qu'on peut appliquer l'action en attente (pending) et envoyer un EndCS
	case transport.ActionBeginCS:
		if a.pending == nil {
			a.log.Warn("handleFromControl", "BeginCS reçu sans action en attente")
			return
		}

		a.handleDistributedAction(a.pending.data)
		pending := a.pending

		if err := a.io.Send(transport.Message{
			Type:   transport.TypeApplication,
			Action: transport.ActionEndCS,
			Sender: a.siteID,
			Data:   pending.data,
		}.String()); err != nil {
			a.log.Error("handleFromControl", "envoi EndCS: "+err.Error())
		}
		a.log.Info("handleFromControl", "SC accordée, EndCS envoyé: "+pending.data["cmd"])
		a.pending = nil

	// Quand on reçoit une ReleaseCS, ça veut dire qu'une action a été validée par le contrôle (après accord de tous les joueurs),
	// et qu'on peut l'appliquer localement (handleDistributedAction)
	case transport.ActionReleaseCS:
		a.log.Info("handleFromControl", "ReleaseCS reçu, action: "+msg.Data["cmd"])
		a.handleDistributedAction(msg.Data)
	}
}
