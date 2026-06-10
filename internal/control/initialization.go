package control

import (
	"github.com/sr05-projet/pkg/transport"
)

// Un site vient de terminer de s'initialiser, il nous en informe
func (c *Control) handleNewSiteAdded(msg *transport.Message) {

	c.AddSite(msg.Sender)

}

// On est le parrain, on doit initialiser un nouveau site
func (c *Control) handleRequestNewSiteInit(msg *transport.Message) {

	id := msg.Data["id"]

	c.log.Info("handleRequestNewSiteInit", "initialisation du nouveau site "+id)

	// recuperer l'etat (snapshot) de l'app et control pour le site local

}
