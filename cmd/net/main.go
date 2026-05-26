// Binaire net : S'occupe de la gestion du réseau entre les sites (ajout/départ d'un participant, diffusion de message avec terminaison explicite, etc.)
//
// Lancement :
//  ./net

package main

import (
	"github.com/sr05-projet/internal/net"
	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

func main() {
	log := logger.New(*name)
	io := transport.NewIO()
	net := net.New(io, log)
	net.Run()
}
