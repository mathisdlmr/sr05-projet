package control

import (
	"fmt"
	"strconv"

	"github.com/sr05-projet/pkg/transport"
)

// --- Helpers pour recaler les horloges ---

// updateClock - recale l'horloge de Lamport lors de la récéption de message : c = max(c, ts) + 1
func (c *Control) updateClock(msg *transport.Message) {
	if *msg.Timestamp > c.clock {
		c.clock = *msg.Timestamp + 1
	} else {
		c.clock = c.clock + 1
	}
	c.log.Debug("Run", fmt.Sprintf("horloge mise à jour: %d", c.clock))
}

// updateVectorClock - recale l'horloge vectorielle lors de la récéption de message : Vi <- Max(Vi, Vm) + 1
func (c *Control) updateVectorClock(vc map[int]int) {
	// Vi <- Max(Vi, Vm) + 1
	for k, v := range vc {
		if v > c.vectorClock[k] {
			c.vectorClock[k] = v
		}
	}
	c.vectorClock[c.myID]++
}

// --- Handlers pour les messages de l'application (locale) et autres centres de contrôle ---

// handleApplicationMessage - traite les messages venant de l'application (locale)
func (c *Control) handleApplicationMessage(msg *transport.Message) {

	// Ignore les message destinés à l'application provenant du control précédent dans l'anneau
	// Les messages de types application sont seulement a destination interne aux sites
	if msg.Sender != c.myID {
		return
	}

	c.log.Info("Run", fmt.Sprintf("message de l'application locale: data=%v", msg.Data))

	if msg.Action == transport.ActionRequestCS { // application request la section critique
		// envoie msg de request : augmente aussi la clock
		c.sendMessage(transport.Message{
			Type:   transport.TypeControl,
			Action: transport.ActionRequestCS,
			// empty data
		})
		// stock l'état correspondant
		c.queue[c.myID] = queueEntry{
			Status:    statusRequest,
			Timestamp: c.clock,
		}
	}

	if msg.Action == transport.ActionEndCS { // application libère la section critique
		// envoie msg de release : augmente aussi la clock
		c.sendMessage(transport.Message{
			Type:   transport.TypeControl,
			Action: transport.ActionReleaseCS,
			// ici on transmet la data pour synchroniser l'état
			Data: msg.Data,
		})
		// stock l'état correspondant
		c.queue[c.myID] = queueEntry{
			Status:    statusRelease,
			Timestamp: c.clock,
		}
	}

	if msg.Action == transport.ActionStartSnapshot { // déclencheur snapshot depuis le navigateur
		// Si on est déjà rouge un snapshot tourne (le nôtre ou celui d'un
		// autre initiateur reçu via Wakeup). On refuse : sinon deux initiateurs
		// simultanés se deadlockent, le premier intercepte les [état] avant
		// qu'ils n'atteignent le second.
		if c.couleur == transport.ColorRed {
			c.log.Warn("handleApplicationMessage", "snapshot refusé : déjà en cours")
			c.sendMessage(transport.Message{
				Type:   transport.TypeApplication,
				Action: transport.ActionSnapshotRejected,
				Data:   map[string]string{"reason": "snapshot already in progress"},
			})
			return
		}
		c.log.Info("handleApplicationMessage", "déclenchement de l'algo 11")
		c.triggerSnapshot(true)
	}

	// fermeture de l'app - départ du site
	if msg.Action == transport.ActionDepart {
		// Message pour prevenir les autre contrôles
		c.sendMessage(transport.Message{
			Type:   transport.TypeControl,
			Action: transport.ActionDepart,
		})
	}
}

// handleControlMessage - traite les messages venant des autres sites
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

	// met a jour la liste des derniers messages reçus de chaque site
	c.LamportClocks[msg.Sender] = *msg.Timestamp

	// Recale Lamport : c = max(c, ts) + 1
	c.updateClock(msg)
	c.updateVectorClock(msg.VectorClock)

	// Règles de couleur de l'algo 11.
	// Déviation du textbook (ligne 19) : on fait le bilan-- APRÈS le check de
	// bascule, pas avant. Pour un message rouge qui déclenche la bascule, le
	// -1 ne doit pas apparaître dans bilan_at_bascule, sinon Σ bilan dérive :
	// le sender est post-sa-bascule (son +1 n'est pas compté non plus), donc
	// un -1 orphelin empêche NbMsgAttendus de revenir à 0. Avec cet ordre, le
	// message bascule-trigger contribue 0 à Σ.
	if isApplicativeRingMessage(*msg) {
		// Bascule rouge (algo 11 lignes 20-23)
		isBasculeTrigger := msg.Color == transport.ColorRed && c.couleur == transport.ColorWhite
		if isBasculeTrigger {
			c.triggerSnapshot(false)
		}

		// Décrément du bilan (algo 11 ligne 19, mais reporté après la bascule).
		// Pour les messages bascule-trigger, on ne décrémente PAS (le receive est
		// conceptuellement "à" la bascule, donc post-snapshot pour le récepteur).
		if !isBasculeTrigger {
			c.bilan--
		}

		// Détection de prépost (algo 11 lignes 25-27) : message blanc reçu
		// alors qu'on est rouge, donc envoyé préclic et reçu postclic. À
		// retransmettre à l'initiateur.
		if msg.Color == transport.ColorWhite && c.couleur == transport.ColorRed {
			c.sendPrepost(msg)
		}
	}

	// Traitement dans chaque handler.
	switch msg.Action {
	case transport.ActionNewSiteAdded:
		c.handleNewSiteAdded(msg)
	case transport.ActionRequestNewSiteInit:
		c.handleRequestNewSiteInit(msg)
	case transport.ActionDepart:
		// quand un site quitte, au niveau contrôle on le retire de la liste
		c.RemoveSite(msg.Sender)
		// et on préviens son net au cas où il y est qq chose à faire
		c.sendMessage(transport.Message{
			Type:   transport.TypeNet,
			Action: transport.ActionDepart,
			Data:   map[string]string{"id": strconv.Itoa(msg.Sender)},
		})

	case transport.ActionRequestCS:
		c.handleRequestCS(msg)
	case transport.ActionAcknowlegeCS:
		c.handleAcknowledgeCS(msg)
	case transport.ActionReleaseCS:
		c.handleReleaseCS(msg)
	case transport.ActionState:
		c.handleSnapshotState(msg)
	case transport.ActionPrepost:
		c.handleSnapshotPrepost(msg)
	case transport.ActionSnapshotComplete:
		c.handleSnapshotComplete(msg)
	case transport.ActionWakeup:
		// Bascule si on est encore blanc. Le wakeup ne passe pas par le
		// lestage/bilan pour ne pas polluer le bilan entre snapshots.
		if c.couleur == transport.ColorWhite {
			c.triggerSnapshot(false)
		}
	default:
		c.log.Warn("Run", fmt.Sprintf("action inconnue dans message de contrôle: %s", msg.Action))
	}
}
