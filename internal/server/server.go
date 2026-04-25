// Package server - Gère le serveur HTTP et la connexion WebSocket avec le navigateur
//
// Infos :
// * Pipeline de communication avec les autres processus : 
//   * Navigateur --WS--> server --stdout--> application
//   * application --stdin--> server --WS--> Navigateur

package server

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sr05-projet/internal/logger"
)

type Server struct {
	addr string
	port string
	log  *logger.Logger
	ws *websocket.Conn
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func New(addr, port, log *logger.Logger) *Server {
	return &Server{
		addr: addr,
		port: port,
		log:  log,
	}
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("../../web")))
	mux.HandleFunc("/ws", s.handleWS)

	go s.readStdinLoop()

	addr := s.addr + ":" + s.port
	s.log.Info("Run", "serveur lancé sur http://"+addr)
	return http.ListenAndServe(addr, mux)
}

// handleWS - handler HTTP qui upgrade la connexion en WebSocket, 
// puis lit en boucle les messages du navigateur et les écrit sur stdout
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	cnx, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("handleWS", "upgrade: "+err.Error())
		return
	}

	s.ws = cnx
	s.log.Info("handleWS", "WebSocket ouverte depuis "+r.RemoteAddr)

	for {
		_, msg, err := cnx.ReadMessage()
		if err != nil {
			s.log.Warn("handleWS", "WebSocket fermée: "+err.Error())
			s.ws = nil
			return
		}
		line := string(msg)
		s.log.Debug("handleWS", "nav->app : "+line)
		fmt.Println(line)
	}
}

// readStdinLoop - lit stdin en continu et pousse chaque ligne vers la WebSocket
func (s *Server) readStdinLoop() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		s.log.Debug("readStdinLoop", "app->nav : "+line)
		s.wsSend(line)
	}
}

// wsSend - envoie un message texte vers le navigateur connecté sur notre WebSocket
func (s *Server) wsSend(msg string) {
	ws := s.ws
	if ws == nil {
		s.log.Warn("wsSend", "pas de WebSocket ouverte, message perdu : "+msg)
		return
	}

	if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		s.log.Error("wsSend", "erreur d'envoi: "+err.Error())
	}
}