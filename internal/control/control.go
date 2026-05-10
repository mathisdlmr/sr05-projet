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
	Status    queueStatus `json:"status"`
	Timestamp int         `json:"timestamp"`
}

type Control struct {
	myID    int
	nbSites int
	io      *transport.IO
	log     *logger.Logger

	clock       int
	vectorClock []int
	// file d'attente section critique
	// tableau de taille nbSites qui contient des
	queue []queueEntry

	// Variables de l'algo 11 d'instantané (Lai-Yang avec reconstitution)
	couleur            string           // ColorWhite par défaut, ColorRed après bascule
	initiateur         bool             // vrai sur le site qui a déclenché le snapshot
	bilan              int              // nb_emissions - nb_receptions de messages applicatifs
	nbEtatsAttendus    int              // utilisé chez l'initiateur uniquement
	nbMsgAttendus      int              // utilisé chez l'initiateur uniquement
	snapshotPending    bool             // vrai pendant l'attente de réponse SnapshotState de l'App
	pendingQueue       []string         // lignes stdin mises de côté pendant le freeze
	pendingControlSnap *ControlSnapshot // snapshot du Control figé à la bascule, en attente de l'app state
	EG                 *EG              // état global collecté (initiateur) ou reçu (autres)
}

func New(myID int, nbSites int, io *transport.IO, log *logger.Logger) *Control {

	queue := make([]queueEntry, nbSites+1)
	for i := 1; i <= nbSites; i++ {
		queue[i] = queueEntry{
			Status:    statusRelease,
			Timestamp: 0,
		}
	}

	// initialisation de l'horloge vectorielle
	vectorClock := make([]int, nbSites+1)
	for i := 1; i <= nbSites; i++ {
		vectorClock[i] = 0
	}

	return &Control{
		myID:        myID,
		nbSites:     nbSites,
		io:          io,
		log:         log,
		clock:       0,
		vectorClock: vectorClock,
		queue:       queue,
		couleur:     transport.ColorWhite,
	}
}

// isApplicativeRingMessage - vrai pour les messages de "l'application de base"
// au sens de l'algo 11, ceux qui doivent être lestés d'une couleur (règle R2)
// et comptés dans le bilan. Le Wakeup n'en fait pas partie : il déclenche la
// bascule via son handler, sans lestage, pour ne pas polluer le bilan entre
// snapshots successifs.
func isApplicativeRingMessage(m transport.Message) bool {
	if m.Type != transport.TypeControl {
		return false
	}
	switch m.Action {
	case transport.ActionRequestCS, transport.ActionReleaseCS, transport.ActionAcknowlegeCS:
		return true
	}
	return false
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

// Message envoyé pour lancer la section critique dans l'app
func (c *Control) localStartCriticalSection() {
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionBeginCS,
	})
}

// -- Helpers pour l'horloge vectorielle --

func (c *Control) updateVectorClock(vec []int) {
	// Vi <- Max(Vi, Vm) + 1
	for i := 0; i < len(vec); i++ {
		if vec[i] > c.vectorClock[i] {
			c.vectorClock[i] = vec[i]
		}
	}
	c.vectorClock[c.myID]++
}

// Wrapper pour l'envoie de message qui :
// 1. (si applicatif) leste avec la couleur courante et ajuste le bilan
// 2. incrémente l'horloge
// 3. ajoute le timestamp, vectorClock et sender au message
// 4. envoie le message
func (c *Control) sendMessage(m transport.Message) {
	if isApplicativeRingMessage(m) {
		// Lestage Lai-Yang : on tague avec la couleur courante (algo 11 ligne 15).
		m.Color = c.couleur
		// L'algo ajoute +1 par envoi. Notre anneau avec tee diffuse à nbSites-1
		// sites en un seul envoi physique (chacun décrémentera de 1), donc on
		// compense pour garder Σ bilan = nb_msg_en_transit.
		c.bilan += c.nbSites - 1
	}

	// incremente l'horloge vectorielle aussi
	c.vectorClock[c.myID]++

	// Incremente aussi l'horloge de Lamport
	c.clock++

	// Ajoute les deux horloges au message
	ts := c.clock
	m.Timestamp = &ts

	vc := make([]int, len(c.vectorClock))
	copy(vc, c.vectorClock)
	m.VectorClock = vc

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

		// Pendant le freeze de l'instantané on n'accepte que la réponse de
		// l'App à notre requête ActionSnapshotState. Le reste va en file
		// d'attente, rejoué après la bascule.
		if c.snapshotPending {
			if msg.Type == transport.TypeApplication &&
				msg.Action == transport.ActionSnapshotState &&
				msg.Data["role"] == "response" &&
				msg.Sender == c.myID {
				c.handleSnapshotStateResponse(msg)
				continue
			}
			c.pendingQueue = append(c.pendingQueue, line)
			continue
		}

		// Gestion des messages selon le type :
		switch msg.Type {
		case transport.TypeApplication: // message de l'application locale
			c.handleApplicationMessage(msg)
		case transport.TypeControl: // message de contrôle d'un autre site
			c.handleControlMessage(msg)
		default:
			c.log.Warn("Run", fmt.Sprintf("message avec type inconnu: type=%s data=%v", msg.Type, msg.Data))
		}
	}
}

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

	// Le forward sur l'anneau est fait dans chaque handler.
	switch msg.Action {
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
		c.io.Send(msg.String()) // forward sur l'anneau
	default:
		c.log.Warn("Run", fmt.Sprintf("action inconnue dans message de contrôle: %s", msg.Action))
		// On forward quand même pour ne pas casser la propagation.
		c.io.Send(msg.String())
	}
}

// --- Handlers pour les messages de file d'attente répartie ---

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
