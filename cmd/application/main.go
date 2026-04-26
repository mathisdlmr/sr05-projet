// Binaire application : Contient la logique du jeu, et le filtrage entre le client (server) et le système réparti (control)
//
// Lancement : 
//  ./application -id J1 -n application_J1
//
// Infos :
// * Pipeline de communication avec les autres processus :
//   * Notre joueur.euse fait une action que l'on doit propager au système : server -> application -> control
//   * L'état du système a changé, on doit remonter l'information au joueur.euse : control -> application -> server

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
	flag.Parse()

	log := logger.New(*name)
	io := transport.NewIO()
	app := application.New(*id, io, log)

	app.Run()
}