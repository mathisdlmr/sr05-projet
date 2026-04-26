// Binaire server : Etabli une websocket avec un client web pour permettre de joueur au jeu sur un navigateur
//
// Lancement :
//	./server -addr localhost -port 4444 -n server
//
package main

import (
	"flag"
	"log"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/internal/server"
)

func main() {
	addr   := flag.String("addr", "localhost", "adresse d'écoute")
	port   := flag.String("port", "4444", "port HTTP/WS")
	name   := flag.String("n", "server", "nom du processus (pour les logs)")
	flag.Parse()

	log := logger.New(*name)

	srv := server.New(*addr, *port, log)
	if err := srv.Run(); err != nil {
		log.Fatal("main", "erreur serveur: "+err.Error())
	}

	_ = log // évite l'import inutilisé si Run ne retourne jamais
}

// Silence l'import log standard si non utilisé
var _ = log.Println