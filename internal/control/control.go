package control

import (
	"fmt"
	"io"

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
	vectorClock map[int]int
	// file d'attente section critique
	// map d'ID du site vers son état en SC
	queue map[int]queueEntry

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
}

// Retrait d'un site
func (c *Control) RemoveSite(id int) {
	if id <= 0 {
		return // invalide
	}

	// clear sa case de la queue - mettre en released
	delete(c.queue, id)

	// mettre sa case de vector clock à -1 pour indiquer qu'elle n'est plus active
	c.vectorClock[id] = -1

	// on retire la case de la map pour cleaner, et on compense dans le comptage
	delete(c.vectorClock, id)

	// on compense juste dans le comptage du nombre de sites.
	c.nbSites--
}
