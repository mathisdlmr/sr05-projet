// Binaire control : S'occupe de la communication entre nos sites, comment partager l'état du jeu, choisir les phases, etc.
//
// Lancement :
//  ./control -id J1 -sites 5 -n control_J1
//
// Infos : 
// * Messages entrants sur stdin :
//   * Sans préfixe    -> viennent de l'application locale
//   * "BROADCAST:..." -> viennent d'un autre centre de contrôle
// * Messages sortants sur stdout :
//   * Sans préfixe    -> destinés à l'application locale
//   * "BROADCAST:..." -> à router vers les autres centres de contrôle (par le script réseau)

package main

import (
	"flag"

	"github.com/sr05-projet/internal/control"
	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

func main() {
	id := flag.String("id", "J1", "identifiant de ce site (ex: J1)")
	nbSites := flag.Int("sites", 5, "nombre total de sites dans le système")
	name := flag.String("n", "ctrl", "nom du processus (pour les logs)")
	flag.Parse()

	log := logger.New(*name)
	io := transport.NewIO()
	ctrl := control.New(*id, *nbSites, io, log)
	ctrl.Run()
}