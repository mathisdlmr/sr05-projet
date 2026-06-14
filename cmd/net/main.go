// Binaire net : S'occupe de la gestion du réseau entre les sites (ajout/départ d'un participant, diffusion de message avec terminaison explicite, etc.)
//
// Lancement :
//  ./net

package main

import (
	"flag"

	"github.com/sr05-projet/internal/net"
	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

func main() {
	id := flag.Int("id", 1, "identifiant de ce site (ex: J1)")
	name := flag.String("n", "net", "nom du processus (pour les logs)")
	nextSiteId := flag.Int("next", 1, "identifiant du site suivant (ex: J2)")
	temporary_tee_in := flag.Int("ttin", 1, "pid of temporary in tee")
	temporary_tee_out := flag.Int("ttout", 1, "pid of temporary out tee")
	flag.Parse()

	log := logger.New(*name)
	io := transport.NewIO()
	net := net.New(*id, io, log, *nextSiteId, *temporary_tee_in, *temporary_tee_out)
	net.Run()
}
