// Binaire server : Etabli une websocket avec un client web pour permettre de joueur au jeu sur un navigateur
//
// Lancement :
//	./server -addr localhost -port 4444 -n server

package main

import (
	"flag"

	"github.com/sr05-projet/internal/server"
	"github.com/sr05-projet/pkg/logger"
)

func main() {
	addr := flag.String("addr", "localhost", "adresse d'écoute")
	port := flag.String("port", "4444", "port HTTP/WS")
	web := flag.String("web", "./web", "chemin du dossier web statique")
	name := flag.String("n", "server", "nom du processus (pour les logs)")
	flag.Parse()

	log := logger.New(*name)
	srv := server.New(*addr, *port, *web, log)
	if err := srv.Run(); err != nil {
		log.Fatal("main", "erreur serveur: "+err.Error())
	}
}
