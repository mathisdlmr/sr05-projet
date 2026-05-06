// Binaire application : Contient la logique du jeu, et s'occupe de faire évoluer l'état du jeu à partir des actions reçues
//                       sur le navigateur web, et via le système réparti (transmis par le centre de contrôle).
//
// Lancement :
//  ./application -id J1 -n application_J1
//
// Infos :
// * Pipeline de communication avec les autres processus :
//   * Notre joueur.euse fait une action que l'on doit propager au système : action sur le navigateur -> application -> control
//   * L'état du système a changé, on doit remonter l'information au joueur.euse : control -> application -> afficher sur le navigateur

package main

import (
	"flag"

	"github.com/sr05-projet/internal/application"
	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

func main() {
	id := flag.String("id", "J1", "identifiant du joueur local (ex: J1)")
	name := flag.String("n", "app", "nom du processus (pour les logs)")
	addr := flag.String("addr", "localhost", "adresse d'écoute")
	port := flag.String("port", "4444", "port HTTP/WS")
	web := flag.String("web", "./web", "chemin du dossier web statique")
	flag.Parse()

	log := logger.New(*name)
	io := transport.NewIO()

	app := application.New(*id, io, log, *addr, *port, *web)
	app.Run()
}
