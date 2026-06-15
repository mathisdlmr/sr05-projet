package control

import (
	"encoding/json"
	"fmt"

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
	myID                        int
	nbSites                     int
	io                          *transport.IO
	log                         *logger.Logger
	initialized                 bool
	awaitingInitSnapshotForSite int

	clock         int
	vectorClock   map[int]int // horloge vectorielle
	LamportClocks map[int]int // timestamps des derniers messages reçus de chaque site

	// file d'attente section critique
	// map d'ID du site vers son état en SC
	queue map[int]queueEntry

	// Variables de l'algo 11 d'instantané (Lai-Yang avec reconstitution)
	couleur               string               // ColorWhite par défaut, ColorRed après bascule
	initiateur            bool                 // vrai sur le site qui a déclenché le snapshot
	bilan                 int                  // nb_emissions - nb_receptions de messages applicatifs
	view                  int                  // numéro de vue : version courante de la liste des membres
	nbEtatsAttendus       int                  // utilisé chez l'initiateur uniquement
	nbMsgAttendus         int                  // utilisé chez l'initiateur uniquement
	globalSnapshotPending bool                 // vrai pendant l'attente de réponse SnapshotState de l'App
	pendingQueue          []*transport.Message // messages mis de côté pendant le freeze
	pendingControlSnap    *ControlSnapshot     // snapshot du Control figé à la bascule, en attente de l'app state
	EG                    *EG                  // état global collecté (initiateur) ou reçu (autres)
}

func New(myID int, nbSites int, initiateur bool, io *transport.IO, log *logger.Logger) *Control {

	// initialisation de la file d'attente (map d'ID du site vers son état en SC)
	queue := make(map[int]queueEntry)
	for i := 1; i <= nbSites; i++ {
		queue[i] = queueEntry{
			Status:    statusRelease,
			Timestamp: 0,
		}
	}

	// initialisation de l'horloge vectorielle (map d'ID du site vers sa valeur)
	vectorClock := make(map[int]int)
	for i := 1; i <= nbSites; i++ {
		vectorClock[i] = 0
	}

	// initialisation de la map des timestamps Lamport reçus de chaque site
	lamportClocks := make(map[int]int)
	for i := 1; i <= nbSites; i++ {
		lamportClocks[i] = 0
	}

	return &Control{
		myID:                        myID,
		nbSites:                     nbSites,
		io:                          io,
		log:                         log,
		clock:                       0,
		vectorClock:                 vectorClock,
		LamportClocks:               lamportClocks,
		queue:                       queue,
		couleur:                     transport.ColorWhite,
		initialized:                 initiateur,
		awaitingInitSnapshotForSite: -1,
	}
}

func (c *Control) HandleMessage(msg *transport.Message) {

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

func (c *Control) Initialize(state SiteState) {
	c.log.Info("Initialize", fmt.Sprintf("initialisation du control avec state=%v", state))

	// la queue devient celle de state.ControlState.Queue
	for id, entry := range state.ControlState.Queue {
		c.queue[id] = queueEntry{
			Status:    entry.Status,
			Timestamp: entry.Timestamp,
		}
	}

	// idem pour l'horloge vectorielle state.ControlState.VectorClock
	for id, ts := range state.ControlState.VectorClock {
		c.vectorClock[id] = ts
	}

	// et pour la liste des derniers timestamps reçus de chaque site
	for id, ts := range state.ControlState.LamportClocks {
		c.LamportClocks[id] = ts
	}

	// On hérite de la vue du parrain avant de s'ajouter comme site actif.
	// AddSite va incrémenter la vue de 1 (notre arrivée = nouveau membership).
	c.view = state.ControlState.View

	// On s'ajoute comme site actif
	c.AddSite(c.myID)
	c.initialized = true

	// On transmet l'état reçu du parrain à l'app
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionNewSiteInit,
		Data:   map[string]string{"state": state.AppState},
	})

	// Envoyer à tous qu'on est initialisé
	c.sendMessage(transport.Message{
		Type:   transport.TypeControl,
		Action: transport.ActionNewSiteAdded,
	})
}

func (c *Control) WaitingForInit() {

	// queue fifo des messages non traités
	initQueue := make([]*transport.Message, 0)

	for {
		msg, err := c.ReadNextMessage()
		if err != nil {
			c.log.Error("WaitingForInit", "lecture message: "+err.Error())
			continue
		}

		if msg.Type == transport.TypeControl && msg.Action == transport.ActionNewSiteInit {
			c.log.Info("WaitingForInit", "message d'initialisation reçu, lancement de Run")
			var initializationData SiteState
			if err := json.Unmarshal([]byte(msg.Data["siteState"]), &initializationData); err != nil {
				c.log.Error("handleSnapshotState", "unmarshal siteState: "+err.Error())
				return
			}
			c.Initialize(initializationData)
			break
		} else {
			c.log.Warn("WaitingForInit", fmt.Sprintf("message reçu avant init: type=%s data=%v, mis en attente", msg.Type, msg.Data))
			initQueue = append(initQueue, msg)
		}
	}

	// Rejoue les messages reçus pendant l'attente de l'init
	for _, msg := range initQueue {
		c.log.Info("WaitingForInit", fmt.Sprintf("retraitement message reçu avant init: type=%s data=%v", msg.Type, msg.Data))

		// ignore les messages déjà inclus dans la snapshot
		// cad les messages deja recus par le parrain avant la snapshot
		if *msg.Timestamp <= c.LamportClocks[msg.Sender] { // TODO : J'ai vu passer certains messages sans timestamp du net je crois, à vérifier mais si c'est ça on perd des messages dans les snapshots
			c.log.Info("WaitingForInit", fmt.Sprintf("message ignoré, déjà inclus dans la snapshot: sender=%d timestamp=%d data=%v", msg.Sender, *msg.Timestamp, msg.Data))
			continue
		}

		c.HandleMessage(msg)
	}

}

func (c *Control) ReadNextMessage() (*transport.Message, error) {
	line, err := c.io.ReadLine()
	if err != nil {
		return nil, err
	}

	msg, err := transport.ParseMessage(line)
	if err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}

	return msg, nil
}

func (c *Control) Run() {
	c.log.Info("Run", fmt.Sprintf("démarrage controle id=%d nbSites=%d", c.myID, c.nbSites))

	// Si non initialisé, rentre dans la boucle d'attente d'init
	if !c.initialized {
		c.WaitingForInit()
	}

	for {
		msg, err := c.ReadNextMessage()
		if err != nil {
			c.log.Error("Run", "lecture message: "+err.Error())
			continue
		}
		c.Dispatch(msg)
	}
}

// Dispatch applique la logique de réception de Run à un message : pendant le
// freeze de l'instantané on n'accepte que la réponse de l'App à notre requête
// ActionSnapshotState, le reste est mis en file et rejoué après la bascule.
func (c *Control) Dispatch(msg *transport.Message) {
	if c.globalSnapshotPending {
		if msg.Type == transport.TypeApplication &&
			msg.Action == transport.ActionSnapshotState &&
			msg.Data["role"] == "response" &&
			msg.Sender == c.myID {
			c.handleSnapshotStateResponse(msg)
			return
		}
		c.pendingQueue = append(c.pendingQueue, msg)
		return
	}
	c.HandleMessage(msg)
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

// Wrapper pour l'envoie de message qui :
// 1. (si applicatif) leste avec la couleur courante et ajuste le bilan
// 2. incrémente l'horloge
// 3. ajoute le timestamp, vectorClock et sender au message
// 4. envoie le message
func (c *Control) sendMessage(m transport.Message) {
	if isApplicativeRingMessage(m) {
		// Lestage Lai-Yang : on tague avec la couleur courante (algo 11 ligne 15).
		m.Color = c.couleur
		// L'algo du cours compte +1 par envoi point-à-point. Chez nous une
		// émission = une diffusion (contrat net : nbSites-1 livraisons, peu
		// importe la topologie), donc +nbSites-1 pour garder
		// Σ bilan = nb de livraisons en attente.
		c.bilan += c.nbSites - 1
	}

	// Tagge le numéro de vue sur les messages de contrôle (ring) pour que
	// les récepteurs puissent distinguer les messages d'une vue périmée.
	if m.Type == transport.TypeControl {
		m.View = c.view
	}

	// incremente l'horloge vectorielle aussi
	c.vectorClock[c.myID]++

	// Incremente aussi l'horloge de Lamport
	c.clock++

	// Ajoute les deux horloges au message
	ts := c.clock
	m.Timestamp = &ts

	// Copier la map vectorClock directement
	vc := make(map[int]int, len(c.vectorClock))
	for k, v := range c.vectorClock {
		vc[k] = v
	}
	m.VectorClock = vc

	m.Sender = c.myID
	c.io.Send(m.String())
}

// Ajout d'un nouveau site
// On suppose qu'on ajoutera toujours un site avec un id plus elevé
func (c *Control) AddSite(id int) {
	// Si le site n'est pas encore dans la map, l'ajouter
	if _, ok := c.vectorClock[id]; !ok {
		c.vectorClock[id] = 0
		c.queue[id] = queueEntry{
			Status:    statusRelease,
			Timestamp: 0,
		}
	}
	c.nbSites++
	c.onViewChange()
}

// Retrait d'un site
func (c *Control) RemoveSite(id int) {
	if id <= 0 {
		return // invalide
	}

	CSRecheck := false
	// s'il était en request, il peut être en section critique
	// et il faut ré-évaluer si c'est libéré à la fin
	if c.queue[id].Status == statusRequest {
		CSRecheck = true
	}

	// clear sa case de la queue
	delete(c.queue, id)

	// on libère la case de la map
	delete(c.vectorClock, id)

	// maj le nombre de sites
	c.nbSites--

	// changement de membership : nouvelle vue (reset bilan / abort snapshot).
	c.onViewChange()

	// si le site partant était en request, il pouvait être en section
	// critique : on ré-évalue si elle est désormais libérée.
	if CSRecheck {
		c.checkCriticalSection()
	}
}

// onViewChange - toute modification du membership ouvre une nouvelle vue.
// Les comptages de l'algo 11 repartent de zéro (les messages de l'ancienne
// vue sont identifiables par leur tag et ne seront pas comptés), et un
// snapshot en cours est avorté : son EG mélangerait deux memberships.
func (c *Control) onViewChange() {
	c.view++
	c.bilan = 0
	if c.couleur == transport.ColorRed {
		c.log.Warn("onViewChange", "changement de membership pendant un snapshot : abort")
		if c.initiateur {
			c.sendMessage(transport.Message{
				Type:   transport.TypeApplication,
				Action: transport.ActionSnapshotRejected,
				Data:   map[string]string{"reason": "membership changed during snapshot"},
			})
		}
		c.resetSnapshotState()
		// Si l'abort survient pendant le freeze (entre bascule et réponse
		// App), sortir aussi du mode freeze et rejouer la file, sinon le
		// control resterait à tout mettre en attente.
		c.globalSnapshotPending = false
		c.pendingControlSnap = nil
		// Une éventuelle init de filleul en attente est abandonnée aussi :
		// son état de référence date de l'ancienne vue.
		c.awaitingInitSnapshotForSite = -1
		c.replayPending()
	}
}
