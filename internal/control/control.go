// Gère les informations de controle dans les messages, et la logique de section critique

package control

import (
	"fmt"
	"io"
	"strconv"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

// enum : status en file d'attente
type queueStatus int

const (
	statusRequest queueStatus = iota + 1
	statusAcknowledge
	statusRelease
)

type queueEntry struct {
	Status    queueStatus
	Timestamp int
}

type Control struct {
	myID    int
	nbSites int
	io      *transport.IO
	log     *logger.Logger

	clock int
	// file d'attente section critique
	// tableau de taille nbSites qui contient des
	queue []queueEntry
}

func New(myID int, nbSites int, io *transport.IO, log *logger.Logger) *Control {

	// initialisation de la file d'attente avec (release, 0)
	queue := make([]queueEntry, nbSites)
	for i := 0; i < nbSites; i++ {
		queue[i] = queueEntry{
			Status:    statusRelease,
			Timestamp: 0,
		}
	}

	return &Control{
		myID:    myID,
		nbSites: nbSites,
		io:      io,
		log:     log,
		clock:   0,
		queue:   queue,
	}
}

// Méthode pour vérifier si on peut entrer en SC
// si queue[myID] == request et que tous les autres ont une estampille plus grande
func (c *Control) checkCriticalSection() {
	if c.queue[c.myID].Status != statusRequest {
		return
	}
	for i := 0; i < c.nbSites; i++ {
		if i != c.myID && c.queue[i].Status == statusRequest && c.queue[i].Timestamp < c.queue[c.myID].Timestamp {
			return
		}
	}
	c.log.Info("checkCriticalSection", "entrée en section critique")
	c.localStartCriticalSection()
}

// Message envoyé pour lancer la section critique dans l'app
func (c *Control) localStartCriticalSection() {
	c.sendMessage(transport.Message{
		Type:   transport.Application,
		Action: "begin_cs",
	})
}

// Wrapper pour l'envoie de message qui :
// 1. incrémente l'horloge
// 2. ajoute le timestamp et sender au message
// 3. envoie le message
func (c *Control) sendMessage(m transport.Message) {
	c.clock++
	ts := c.clock
	m.Timestamp = &ts
	m.Sender = c.myID
	c.io.Send(m.String())
}

func (c *Control) updateClock(msg *transport.Message) {
	if *msg.Timestamp > c.clock {
		c.clock = *msg.Timestamp + 1
	} else {
		c.clock = c.clock + 1
	}
	c.log.Debug("Run", fmt.Sprintf("horloge mise à jour: %d", c.clock))
}

func (c *Control) Run() {
	c.log.Info("Run", fmt.Sprintf("démarrage controle id=%d nbSites=%d", c.myID, c.nbSites))
	for {
		line, err := c.io.ReadLine()
		if err == io.EOF {
			c.log.Info("Run", "stdin fermé, arrêt")
			return
		}
		if err != nil {
			c.log.Error("Run", "lecture stdin: "+err.Error())
			return
		}

		// parse message
		msg, err := transport.ParseMessage(line)
		if err != nil {
			c.log.Error("Run", "parse message: "+err.Error())
			continue
		}

		// Gestion des messages selon le type :
		switch msg.Type {
		case transport.Application: // message de l'application locale
			c.handleApplicationMessage(msg)
		case transport.Control: // message de contrôle d'un autre site
			c.handleControlMessage(msg)
		default:
			c.log.Warn("Run", fmt.Sprintf("message avec type inconnu: type=%s data=%v", msg.Type, msg.Data))
		}
	}
}

func (c *Control) handleApplicationMessage(msg *transport.Message) {
	c.log.Info("Run", fmt.Sprintf("message de l'application locale: data=%v", msg.Data))

	if msg.Action == "request_cs" { // application request la section critique
		// envoie msg de request : augmente aussi la clock
		c.sendMessage(transport.Message{
			Type:   transport.Control,
			Action: "request_cs",
			// empty data
		})
		// stock l'état correspondant
		c.queue[c.myID] = queueEntry{
			Status:    statusRequest,
			Timestamp: c.clock,
		}
	}

	if msg.Action == "end_cs" { // application libère la section critique
		// envoie msg de release : augmente aussi la clock
		c.sendMessage(transport.Message{
			Type:   transport.Control,
			Action: "release_cs",
			// ici on transmet la data pour synchroniser l'état
			Data: msg.Data,
		})
		// stock l'état correspondant
		c.queue[c.myID] = queueEntry{
			Status:    statusRelease,
			Timestamp: c.clock,
		}
	}
}

func (c *Control) handleControlMessage(msg *transport.Message) {
	if msg.Timestamp == nil {
		c.log.Warn("Run", "control_message reçu sans timestamp, ignoré")
		return
	}
	if msg.Sender == c.myID {
		c.log.Debug("Run", fmt.Sprintf("control_message propre de retour, ignoré (anneau) timestamp=%d", *msg.Timestamp))
		return
	}
	c.log.Info("Run", fmt.Sprintf("message de contrôle reçu: sender=%d timestamp=%d data=%v", msg.Sender, *msg.Timestamp, msg.Data))

	// Recale Lamport : c = max(c, ts) + 1
	c.updateClock(msg)

	switch msg.Action {
	case "request_cs":
		c.handleRequestCS(msg)
	case "acknowledge_cs":
		c.handleAcknowledgeCS(msg)
	case "release_cs":
		c.handleReleaseCS(msg)
	default:
		c.log.Warn("Run", fmt.Sprintf("action inconnue dans message de contrôle: %s", msg.Action))
	}

	// Forward sur l'anneau (timestamp conservé pour les autres sites)
	c.io.Send(msg.String())
}

// --- Handlers pour les messages de file d'attente répartie ---

func (c *Control) handleRequestCS(msg *transport.Message) {
	c.queue[msg.Sender] = queueEntry{
		Status:    statusRequest,
		Timestamp: *msg.Timestamp,
	}
	// envoyer un message d'acquittement à senderID
	ackMsg := transport.Message{
		Type:   transport.Control,
		Action: "acknowledge_cs",
		// on indique le destinataire dans data
		Data: map[string]string{
			"target": fmt.Sprintf("%d", msg.Sender),
		},
	}
	c.io.Send(ackMsg.String())
	c.log.Info("Run", fmt.Sprintf("envoi d'un message d'acquittement à %d", msg.Sender))
	c.checkCriticalSection()
}

func (c *Control) handleAcknowledgeCS(msg *transport.Message) {
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
	c.queue[msg.Sender] = queueEntry{
		Status:    statusAcknowledge,
		Timestamp: *msg.Timestamp,
	}
	c.checkCriticalSection()
}

func (c *Control) handleReleaseCS(msg *transport.Message) {
	c.queue[msg.Sender] = queueEntry{
		Status:    statusRelease,
		Timestamp: *msg.Timestamp,
	}
	c.checkCriticalSection()
}
