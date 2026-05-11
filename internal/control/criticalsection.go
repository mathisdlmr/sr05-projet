package control

import (
	"fmt"
	"strconv"

	"github.com/sr05-projet/pkg/transport"
)

// Message envoyé pour lancer la section critique dans l'app
func (c *Control) localStartCriticalSection() {
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionBeginCS,
	})
}

// checkCriticalSection - vérifie si ce site peut entrer en SC :
// aucun autre site ne doit avoir une requête avec une estampille plus grande
// (au sens de la relation <_K vue dans le cours)
func (c *Control) checkCriticalSection() {
	if c.queue[c.myID].Status != statusRequest {
		return
	}
	mine := c.queue[c.myID]
	for i := 1; i <= c.nbSites; i++ {
		if i == c.myID {
			continue
		}
		other := c.queue[i]
		if other.Status == statusRequest &&
			(other.Timestamp < mine.Timestamp ||
				(other.Timestamp == mine.Timestamp && i < c.myID)) {
			return
		}
	}
	c.log.Info("checkCriticalSection", "entrée en section critique")
	c.localStartCriticalSection()
}

// handleRequestCS - traite les messages de type RequestCS (requete de section critique d'un site)
func (c *Control) handleRequestCS(msg *transport.Message) {
	c.queue[msg.Sender] = queueEntry{
		Status:    statusRequest,
		Timestamp: *msg.Timestamp,
	}
	// envoyer un message d'acquittement à senderID
	ackMsg := transport.Message{
		Type:   transport.TypeControl,
		Action: transport.ActionAcknowlegeCS,
		// on indique le destinataire dans data
		Data: map[string]string{
			"target": fmt.Sprintf("%d", msg.Sender),
		},
	}
	c.sendMessage(ackMsg)
	c.log.Info("Run", fmt.Sprintf("envoi d'un message d'acquittement à %d", msg.Sender))
	c.checkCriticalSection()
	// Forward sur l'anneau : la requête doit faire le tour pour que tous les
	// sites la voient dans leur queue locale.
	c.io.Send(msg.String())
}

// handleAcknowledgeCS - traite les messages de type AcknowledgeCS (acquittement d'une requete de section critique d'un site)
func (c *Control) handleAcknowledgeCS(msg *transport.Message) {
	// Forward sur l'anneau dans tous les cas. Même si l'ack ne nous est pas
	// destiné, il doit continuer à circuler pour atteindre son destinataire
	// et boucler jusqu'à l'émetteur (auto-détection).
	defer c.io.Send(msg.String())

	// vérifie que le message d'acquittement nous est bien destiné
	targetStr, ok := msg.Data["target"]
	if !ok {
		c.log.Warn("Run", "message d'acquittement reçu sans target, ignoré")
		return
	}
	target, err := strconv.Atoi(targetStr)
	if err != nil {
		c.log.Warn("Run", fmt.Sprintf("message d'acquittement reçu avec target non entier: %s, ignoré", targetStr))
		return
	}
	if target != c.myID {
		c.log.Debug("Run", fmt.Sprintf("message d'acquittement reçu pour %d, pas pour nous (%d), ignoré", target, c.myID))
		return
	}
	// ne pas écraser une requete par un acknowledgement
	if c.queue[msg.Sender].Status == statusRequest {
		c.log.Debug("Run", fmt.Sprintf("message d'acquittement reçu de %d, mais on a déjà une requete de ce site, ignoré", msg.Sender))
		return
	}
	c.queue[msg.Sender] = queueEntry{
		Status:    statusAcknowledge,
		Timestamp: *msg.Timestamp,
	}
	c.checkCriticalSection()
}

// handleReleaseCS - traite les messages de type ReleaseCS (libération de la section critique d'un site)
func (c *Control) handleReleaseCS(msg *transport.Message) {
	c.queue[msg.Sender] = queueEntry{
		Status:    statusRelease,
		Timestamp: *msg.Timestamp,
	}

	// Transmettre l'info reçue des autres sites vers l'application
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionReleaseCS,
		Data:   msg.Data,
	})

	c.checkCriticalSection()
	// Forward sur l'anneau (broadcast du release pour que tous les sites
	// mettent à jour leur queue locale).
	c.io.Send(msg.String())
}