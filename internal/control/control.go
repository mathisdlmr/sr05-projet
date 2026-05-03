// Gère les informations de controle dans les messages, et la logique de section critique

// 3 Types de messages :
// * Messages de données : envoyés par l'application locale, avec type "data_message".
// 	Ils sont transformés en messages de contrôle (type "control_message") avec un timestamp et retransmis à tous les centres de contrôle.
// * Messages de contrôle : envoyés par les centres de contrôle, avec type "control_message".
// 	Ils contiennent un timestamp et sont retransmis à tous les centres de contrôle (sans le timestamp).
// * Messages de section critique : envoyés par les centres de contrôle, avec type "critical_section".
// 	Ils contiennent un timestamp et une action (request, acknowledge, release) et sont traités localement pour gérer la file d'attente de la section critique.

// Les messages sans timestamp seront reconnus et traités par l'application locale,
// les autres seront envoyés aux autres
// l'entrée en section critique est reconnu par l'application locale grâce à un message de type "critical_section" avec action "enter".

package control

import (
	"fmt"
	"io"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

// enum : status en file d'attente
const (
	Request     = 1
	Acknowledge = 2
	Release     = 3
)

type queueEntry struct {
	Status    int
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
			Status:    Release,
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

// Méthode pour vérifier et entrer en section critique :
// si queue[myID] == request et que tous les autres ont une estampille plus grande
// si c'est bon on envoie le message de section critique à l'application
func (c *Control) checkCriticalSection() {
	if c.queue[c.myID].Status == Request {
		canEnter := true
		for i := 0; i < c.nbSites; i++ {
			if i != c.myID && c.queue[i].Status == Request && c.queue[i].Timestamp < c.queue[c.myID].Timestamp {
				canEnter = false
				break
			}
		}
		if canEnter {
			c.log.Info("checkCriticalSection", "entrée en section critique")
			// envoyer message de section critique à l'application locale
			msg := transport.Message{
				Type:   transport.CriticalSection,
				Sender: 0, // on peut mettre n'importe quel sender, car le message de section critique est traité localement
				Data: map[string]string{
					"action": "enter",
				},
			}
			c.io.Send(msg.String())
		}
	}
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

		// 1. Seulement message, venant de l'application
		switch msg.Type {
		case transport.DataMessage:
			c.log.Info("Run", fmt.Sprintf("message de l'application locale: data=%v", msg.Data))
			// Lamport : émission -> incrément
			c.clock++
			ts := c.clock
			msg.Sender = c.myID
			msg.Timestamp = &ts
			msg.Type = transport.ControlMessage
			c.io.Send(msg.String())
			continue

		// 2. Message de contrôle, venant d'un autre centre de contrôle
		//    -> recaler l'horloge et retransmettre tel quel sur l'anneau ;
		//       si le message nous revient (Sender == myID), on l'ignore.
		case transport.ControlMessage:
			if msg.Timestamp == nil {
				c.log.Warn("Run", "control_message reçu sans timestamp, ignoré")
				continue
			}
			if msg.Sender == c.myID {
				c.log.Debug("Run", fmt.Sprintf("control_message propre de retour, ignoré (anneau) timestamp=%d", *msg.Timestamp))
				continue
			}
			c.log.Info("Run", fmt.Sprintf("message de contrôle reçu: sender=%d timestamp=%d data=%v", msg.Sender, *msg.Timestamp, msg.Data))
			// Recale Lamport : c = max(c, ts) + 1
			if *msg.Timestamp > c.clock {
				c.clock = *msg.Timestamp + 1
			} else {
				c.clock = c.clock + 1
			}
			c.log.Debug("Run", fmt.Sprintf("horloge mise à jour: %d", c.clock))
			// Forward sur l'anneau (timestamp conservé pour les autres sites)
			c.io.Send(msg.String())
			continue

		// 3. Message de section critique, venant d'un autre centre de contrôle ou de l'application locale
		// TODO : la machine d'état ci-dessous a encore plusieurs bugs (Sender:0 à l'émission,
		// pas de loop-prevention sur l'anneau). À refaire avec la file d'attente Lamport propre.
		case transport.CriticalSection:
			ts := -1
			if msg.Timestamp != nil {
				ts = *msg.Timestamp
			}
			c.log.Info("Run", fmt.Sprintf("message de section critique reçu: sender=%d timestamp=%d data=%v", msg.Sender, ts, msg.Data))
			// Recale Lamport (si timestamp présent) : c = max(c, ts) + 1
			if msg.Timestamp != nil {
				if *msg.Timestamp > c.clock {
					c.clock = *msg.Timestamp + 1
				} else {
					c.clock = c.clock + 1
				}
				c.log.Debug("Run", fmt.Sprintf("horloge mise à jour: %d", c.clock))
			}
			// update queue
			senderID := msg.Sender
			action := msg.Data["action"]
			switch action {
			case "request":
				c.queue[senderID] = queueEntry{
					Status:    Request,
					Timestamp: *msg.Timestamp,
				}
				// envoyer un message d'acquittement à senderID
				ackMsg := transport.Message{
					Type:   transport.CriticalSection,
					Sender: 0, // on peut mettre n'importe quel sender, car le message d'acquittement est traité localement
					Data: map[string]string{
						"action": "acknowledge",
						"target": fmt.Sprintf("%d", senderID),
					},
				}
				c.io.Send(ackMsg.String())
				c.log.Info("Run", fmt.Sprintf("envoi d'un message d'acquittement à %d", senderID))
				c.checkCriticalSection()
			case "acknowledge":
				c.queue[senderID] = queueEntry{
					Status:    Acknowledge,
					Timestamp: *msg.Timestamp,
				}
				c.checkCriticalSection()
			case "release":
				c.queue[senderID] = queueEntry{
					Status:    Release,
					Timestamp: *msg.Timestamp,
				}
				c.checkCriticalSection()
			case "end":
				c.log.Info("Run", "fin de la section critique de l'application locale")
				// augmenter l'horloge
				c.clock++
				// mettre à jour notre propre entrée dans la file d'attente
				c.queue[c.myID] = queueEntry{
					Status:    Release,
					Timestamp: c.clock,
				}
				// envoyer un message de release à tous les autres centres de contrôle
				releaseMsg := transport.Message{
					Type:   transport.CriticalSection,
					Sender: 0, // on peut mettre n'importe quel sender, car le message de release est traité localement
					Data: map[string]string{
						"action": "release",
					},
				}
				c.io.Send(releaseMsg.String())
				c.log.Info("Run", "envoi d'un message de release à tous les autres centres de contrôle")
				c.checkCriticalSection()

			default:
				c.log.Warn("Run", fmt.Sprintf("action inconnue dans message de section critique: %s", action))
			}
			continue

		default:
			c.log.Warn("Run", fmt.Sprintf("message avec type inconnu: type=%s data=%v", msg.Type, msg.Data))
		}
	}
}
