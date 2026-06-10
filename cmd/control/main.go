// Binaire control : Permet de gérer la communication entre les différents sites de sorte à ce que le système réparti reste cohérent
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
	id := flag.Int("id", 1, "identifiant de ce site (ex: J1)")
	nbSites := flag.Int("sites", 5, "nombre total de sites dans le système")
	name := flag.String("n", "ctrl", "nom du processus (pour les logs)")
	initiateur := flag.Bool("isInitiator", false, "True si ce site est le premier créé, i.e. n'attend pas d'être initialisé.")
	flag.Parse()

	log := logger.New(*name)
	io := transport.NewIO()
	ctrl := control.New(*id, *nbSites, *initiateur, io, log)
	ctrl.Run()
}
