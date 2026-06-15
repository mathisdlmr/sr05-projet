package control

import (
	"encoding/json"
	"strconv"

	"github.com/sr05-projet/pkg/transport"
)

// Trigger quand on reçois la snapshot de l'application
func (c *Control) sendInitToSite(appState string, siteID int) {

	siteState := SiteState{
		ControlState: c.snapshotControlState(),
		AppState:     appState,
		VectorClock:  c.vectorClock,
	}

	siteStateJSON, err := json.Marshal(siteState)
	if err != nil {
		c.log.Error("handleSnapshotStateResponse", "marshal siteState: "+err.Error())
		return
	}

	c.sendMessage(transport.Message{
		Type:   transport.TypeControl,
		Action: transport.ActionNewSiteInit,
		Data: map[string]string{
			"siteState": string(siteStateJSON),
			"target":    strconv.Itoa(siteID),
		},
	})
}

// Un site vient de terminer de s'initialiser, il nous en informe
func (c *Control) handleNewSiteAdded(msg *transport.Message) {
	c.AddSite(msg.Sender)
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionSiteAjoute,
		Data:   map[string]string{"id": "J" + strconv.Itoa(msg.Sender)},
	})
}

// On est le parrain, on doit initialiser un nouveau site
func (c *Control) handleRequestNewSiteInit(msg *transport.Message) {

	id := msg.Data["id"]

	// On parse l'id du site avant d'envoyer la requête à l'app :
	// un id invalide doit avorter proprement sans lancer le round-trip.
	siteID, err := strconv.Atoi(id)
	if err != nil {
		c.log.Error("handleRequestNewSiteInit", "id de site invalide: "+id)
		return
	}
	c.awaitingInitSnapshotForSite = siteID

	c.log.Info("handleRequestNewSiteInit", "initialisation du nouveau site "+id)

	// requete a app locale pour qu'elle nous envoie son snapshot (etat de l'app + horloge)
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionSnapshotState,
		Data:   map[string]string{"role": "request"},
	})
}
