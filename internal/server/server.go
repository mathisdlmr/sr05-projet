// Package server - Gère le serveur HTTP et la connexion WebSocket avec le navigateur.
//
// Le serveur sert deux directions :
//   - Navigateur -> application : les messages reçus sur la WebSocket sont
//     publiés sur le channel renvoyé par Inbox(). C'est l'application qui
//     les consomme et décide quoi en faire.
//   - Application -> navigateur : l'application appelle Send() pour pousser
//     un message texte sur la WebSocket courante (en pratique une chaîne JSON).

package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sr05-projet/pkg/logger"
)

type Server struct {
	addr string
	port string
	web  string
	log  *logger.Logger

	mu       sync.Mutex
	ws       *websocket.Conn
	inbox    chan string
	connects chan struct{}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func New(addr string, port string, web string, log *logger.Logger) *Server {
	return &Server{
		addr:     addr,
		port:     port,
		web:      web,
		log:      log,
		inbox:    make(chan string, 16),
		connects: make(chan struct{}, 1),
	}
}

// Connects - retourne un channel qui reçoit un signal à chaque nouvelle connexion WebSocket
func (s *Server) Connects() <-chan struct{} { return s.connects }

// Inbox - canal des messages reçus du navigateur (lecture seule pour l'app).
func (s *Server) Inbox() <-chan string { return s.inbox }

// Send - écrit un message texte sur la WebSocket connectée.
// Si aucun navigateur n'est connecté, on log un warn et on droppe.
func (s *Server) Send(msg string) error {
	s.mu.Lock()
	ws := s.ws
	s.mu.Unlock()

	if ws == nil {
		s.log.Warn("Send", "pas de WebSocket ouverte, message perdu : "+msg)
		return nil
	}
	if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		s.log.Error("Send", "erreur d'envoi: "+err.Error())
		return err
	}
	return nil
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(s.web)))
	mux.HandleFunc("/ws", s.handleWS)

	addr := s.addr + ":" + s.port
	s.log.Info("Run", "serveur lancé sur http://"+addr)
	return http.ListenAndServe(addr, mux)
}

// handleWS - upgrade HTTP -> WebSocket puis lit en boucle les messages
// du navigateur et les pousse sur le channel inbox pour l'application.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	cnx, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("handleWS", "upgrade: "+err.Error())
		return
	}

	s.mu.Lock()
	s.ws = cnx
	s.mu.Unlock()
	s.log.Info("handleWS", "WebSocket ouverte depuis "+r.RemoteAddr)

	select {
	case s.connects <- struct{}{}:
	default:
	}

	for {
		_, msg, err := cnx.ReadMessage()
		if err != nil {
			s.log.Warn("handleWS", "WebSocket fermée: "+err.Error())
			s.mu.Lock()
			s.ws = nil
			s.mu.Unlock()
			return
		}
		line := string(msg)
		s.log.Debug("handleWS", "nav->app : "+line)
		s.inbox <- line
	}
}

type BrowserMessage struct {
	Action string            `json:"action"`
	Data   map[string]string `json:"data"`
}

func (s *Server) PushMessageToBrowser(msg BrowserMessage) {
	out, err := json.Marshal(msg)
	if err != nil {
		s.log.Error("pushMessageToBrowser", "marshal: "+err.Error())
		return
	}
	if err := s.Send(string(out)); err != nil {
		s.log.Warn("pushMessageToBrowser", "send: "+err.Error())
	}
}
