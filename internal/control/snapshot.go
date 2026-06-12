// Algorithme 11 d'instantané (Lai-Yang avec reconstitution + terminaison)
// Référence : docs/cours/poly-instantanes.pdf chapitre 6, Algorithme 11.
//
// Ce fichier regroupe :
//   - les types EG, SiteState, ControlSnapshot
//   - la procédure de bascule (triggerSnapshot)
//   - les handlers des messages snapshot (state, prepost, snapshotComplete)
//   - la condition de terminaison + la diffusion finale
//   - la file d'attente pendant le round-trip Control↔App

package control

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sr05-projet/pkg/transport"
)

// ControlSnapshot capture l'état du Control au moment de la bascule.
type ControlSnapshot struct {
	Queue         map[int]queueEntry `json:"queue"`
	Bilan         int                `json:"bilan"`
	VectorClock   map[int]int        `json:"vectorClock"`
	LamportClocks map[int]int        `json:"lamportClocks"`
	View          int                `json:"view"`
}

// SiteState représente l'état complet d'un site (Control + App + horloge)
// à l'instant de la prise d'instantané. C'est l'unité collectée par l'initiateur.
type SiteState struct {
	ControlState ControlSnapshot `json:"controlState"`
	AppState     string          `json:"appState"` // JSON sérialisé du GameState (opaque côté Control)
	VectorClock  map[int]int     `json:"vectorClock"`
}

// EG (État Global) est la structure collectée par l'initiateur de l'algo 11
// puis diffusée à tous les sites via ActionSnapshotComplete.
type EG struct {
	States   map[int]SiteState `json:"states"`
	Preposts []string          `json:"preposts"` // messages préposts au format wire
}

func newEG() *EG {
	return &EG{
		States:   make(map[int]SiteState),
		Preposts: []string{},
	}
}

// snapshotControlState deep-copie l'état du Control au moment t.
// Appelé à la bascule pour figer ce qu'on capture (queue, bilan, vectorClock).
func (c *Control) snapshotControlState() ControlSnapshot {
	// deep copy de la map queue
	q := make(map[int]queueEntry, len(c.queue))
	for k, v := range c.queue {
		q[k] = v
	}
	// deep copy de la map vectorClock
	vc := make(map[int]int, len(c.vectorClock))
	for k, v := range c.vectorClock {
		vc[k] = v
	}
	return ControlSnapshot{
		Queue:         q,
		Bilan:         c.bilan,
		VectorClock:   vc,
		LamportClocks: c.LamportClocks,
		View:          c.view,
	}
}

// triggerSnapshot exécute la bascule de couleur et déclenche la collecte.
// initiateur=true quand le déclencheur vient du navigateur (via ActionStartSnapshot
// reçu de l'App locale), initiateur=false quand la bascule est déclenchée par la
// réception d'un message rouge (algo 11 lignes 20-23).
func (c *Control) triggerSnapshot(initiateur bool) {
	if c.couleur == transport.ColorRed {
		c.log.Warn("triggerSnapshot", "déjà rouge, bascule ignorée")
		return
	}

	// 1. Flip couleur (avant toute autre opération, cf. design § "Procédure exacte")
	c.couleur = transport.ColorRed

	// 2. Capture de l'état Control en mémoire (deep-copy synchrone)
	ctrlSnap := c.snapshotControlState()
	c.pendingControlSnap = &ctrlSnap

	// 3. Init des variables propres à l'initiateur
	c.initiateur = initiateur
	if initiateur {
		c.EG = newEG()
		c.nbEtatsAttendus = c.nbSites - 1
		c.nbMsgAttendus = ctrlSnap.Bilan
		c.log.Info("triggerSnapshot",
			fmt.Sprintf("INITIATEUR rouge, nbEtatsAttendus=%d, nbMsgAttendus=bilan=%d",
				c.nbEtatsAttendus, c.nbMsgAttendus))
	} else {
		c.log.Info("triggerSnapshot", "bascule rouge (non-initiateur)")
	}

	// 4. Demande de l'état applicatif à l'App locale
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionSnapshotState,
		Data:   map[string]string{"role": "request"},
	})

	// 5. Mode pending. La suite se fait dans handleSnapshotStateResponse.
	c.globalSnapshotPending = true
}

// handleSnapshotStateResponse est appelée quand l'App locale répond à notre
// requête ActionSnapshotState. C'est ici qu'on finalise EG_i, qu'on envoie
// [état] sur l'anneau (si non-initiateur), et qu'on rejoue la file d'attente.
func (c *Control) handleSnapshotStateResponse(msg *transport.Message) {

	if c.awaitingInitSnapshotForSite != -1 {
		c.sendInitToSite(msg.Data["state"], c.awaitingInitSnapshotForSite)
	}

	if !c.globalSnapshotPending || c.pendingControlSnap == nil {
		c.log.Warn("handleSnapshotStateResponse", "reçu hors mode pending, ignoré")
		return
	}

	appState := msg.Data["state"]
	bilanAtBascule := c.pendingControlSnap.Bilan

	siteState := SiteState{
		ControlState: *c.pendingControlSnap,
		AppState:     appState,
		VectorClock:  c.pendingControlSnap.VectorClock,
	}

	if c.initiateur {
		// L'initiateur stocke directement dans son EG (il n'envoie pas [état])
		c.EG.States[c.myID] = siteState
		c.log.Info("handleSnapshotStateResponse", "EG_i initiateur capturé localement")

		// Envoi d'un Wakeup rouge sur l'anneau pour garantir que tous les sites
		// finiront par recevoir un message rouge et basculer (cf. exo 127/128 du
		// poly). Sans ça, si l'application n'émet plus de messages post-bascule,
		// les autres sites ne basculeraient jamais et l'algo ne terminerait pas.
		c.sendMessage(transport.Message{
			Type:   transport.TypeControl,
			Action: transport.ActionWakeup,
		})
		c.log.Info("handleSnapshotStateResponse", "Wakeup envoyé sur l'anneau")
	} else {
		// Non-initiateur : envoie [état] sur l'anneau vers l'initiateur (algo 11 ligne 23)
		siteStateJSON, err := json.Marshal(siteState)
		if err != nil {
			c.log.Error("handleSnapshotStateResponse", "marshal siteState: "+err.Error())
			return
		}
		c.sendMessage(transport.Message{
			Type:   transport.TypeControl,
			Action: transport.ActionState,
			Data: map[string]string{
				"siteState": string(siteStateJSON),
				"bilan":     strconv.Itoa(bilanAtBascule),
			},
		})
		c.log.Info("handleSnapshotStateResponse", "[état] envoyé sur l'anneau")
	}

	// Sortie du mode pending
	c.globalSnapshotPending = false
	c.pendingControlSnap = nil

	// Rejouer la file d'attente accumulée pendant le freeze
	c.replayPending()

	// Cas dégénéré : l'initiateur peut déjà avoir atteint la terminaison
	// (par ex. bilan=0 et N=1, ou tous les états reçus pendant le freeze).
	if c.initiateur {
		c.checkSnapshotTermination()
	}
}

// replayPending re-feed les lignes accumulées pendant le freeze à travers le
// dispatch normal. Les règles de couleur s'appliquent : les messages blancs
// reçus pendant qu'on est rouge sont détectés comme préposts à ce moment-là.
func (c *Control) replayPending() {
	queue := c.pendingQueue
	c.pendingQueue = nil
	if len(queue) == 0 {
		return
	}
	c.log.Info("replayPending", fmt.Sprintf("rejeu de %d message(s)", len(queue)))
	for _, msg := range queue {
		c.HandleMessage(msg)
	}
}

// handleSnapshotState traite la réception d'un message [état] diffusé par le réseau
// (algo 11 lignes 30-40). Si on est l'initiateur : on collecte. Sinon : forward.
func (c *Control) handleSnapshotState(msg *transport.Message) {
	if msg.View != c.view {
		c.log.Warn("handleSnapshotState", "message d'une autre vue, ignoré")
		return
	}
	if !c.initiateur {
		return
	}

	var siteState SiteState
	if err := json.Unmarshal([]byte(msg.Data["siteState"]), &siteState); err != nil {
		c.log.Error("handleSnapshotState", "unmarshal siteState: "+err.Error())
		return
	}
	bilan, _ := strconv.Atoi(msg.Data["bilan"])

	if c.EG == nil {
		c.EG = newEG()
	}
	c.EG.States[msg.Sender] = siteState
	c.nbEtatsAttendus--
	c.nbMsgAttendus += bilan

	c.log.Info("handleSnapshotState",
		fmt.Sprintf("[état] de site %d intégré, nbEtatsAttendus=%d, nbMsgAttendus=%d",
			msg.Sender, c.nbEtatsAttendus, c.nbMsgAttendus))

	c.checkSnapshotTermination()
}

// handleSnapshotPrepost traite la réception d'un message [prépost] diffusé par le réseau
// (algo 11 lignes 41-51). Si on est l'initiateur : on collecte. Sinon : forward.
func (c *Control) handleSnapshotPrepost(msg *transport.Message) {
	if msg.View != c.view {
		c.log.Warn("handleSnapshotPrepost", "message d'une autre vue, ignoré")
		return
	}
	if !c.initiateur {
		return
	}

	wrappedMsg := msg.Data["msg"]
	if c.EG == nil {
		c.EG = newEG()
	}
	c.EG.Preposts = append(c.EG.Preposts, wrappedMsg)
	c.nbMsgAttendus--

	c.log.Info("handleSnapshotPrepost",
		fmt.Sprintf("[prépost] reçu, nbMsgAttendus=%d", c.nbMsgAttendus))

	c.checkSnapshotTermination()
}

// handleSnapshotComplete traite la réception du message de diffusion finale.
// Tous les sites (sauf l'initiateur via self-detect) : sauvegarde locale +
// push à l'App locale + forward. C'est une extension hors algo 11 strict
// (cf. design § "Diffusion finale").
func (c *Control) handleSnapshotComplete(msg *transport.Message) {
	// Un complete d'une ancienne vue ne doit pas écraser l'EG ni remettre à
	// blanc un site qui a déjà basculé dans la nouvelle vue.
	if msg.View != c.view {
		c.log.Warn("handleSnapshotComplete", "message d'une autre vue, ignoré")
		return
	}
	egJSON := msg.Data["eg"]
	var eg EG
	if err := json.Unmarshal([]byte(egJSON), &eg); err != nil {
		c.log.Error("handleSnapshotComplete", "unmarshal: "+err.Error())
		return
	}
	c.EG = &eg

	// Push à l'App locale pour affichage navigateur
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionRestoreSnapshot,
		Data:   map[string]string{"eg": egJSON},
	})

	c.log.Success("handleSnapshotComplete", "snapshot reçu, sauvegardé et poussé à l'App")

	// Reset couleur pour permettre un futur snapshot
	c.resetSnapshotState()
}

// checkSnapshotTermination vérifie la condition de fin chez l'initiateur.
// L'algo 11 la teste à la fois lignes 35 et 46, donc on appelle ça depuis
// les handlers state et prepost.
func (c *Control) checkSnapshotTermination() {
	if !c.initiateur {
		return
	}
	if c.nbEtatsAttendus == 0 && c.nbMsgAttendus == 0 {
		c.log.Success("checkSnapshotTermination",
			"terminaison atteinte, diffusion finale de l'EG")
		c.broadcastSnapshotComplete()
	}
}

// broadcastSnapshotComplete envoie l'EG final sur l'anneau et le pousse aussi
// à notre App locale (l'initiateur ne reçoit pas son propre snapshotComplete
// du fait du self-detect, donc il faut le pousser explicitement).
func (c *Control) broadcastSnapshotComplete() {
	egJSON, err := json.Marshal(c.EG)
	if err != nil {
		c.log.Error("broadcastSnapshotComplete", "marshal: "+err.Error())
		return
	}

	// Diffusion sur l'anneau
	c.sendMessage(transport.Message{
		Type:   transport.TypeControl,
		Action: transport.ActionSnapshotComplete,
		Data:   map[string]string{"eg": string(egJSON)},
	})

	// Push à notre propre App (on ne recevra pas notre propre msg de retour)
	c.sendMessage(transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionRestoreSnapshot,
		Data:   map[string]string{"eg": string(egJSON)},
	})

	c.log.Info("broadcastSnapshotComplete", "EG diffusé sur l'anneau")

	// Reset couleur pour permettre un futur snapshot
	c.resetSnapshotState()
}

// sendPrepost gère un message reçu blanc alors qu'on est rouge.
//   - Si on n'est pas l'initiateur : on emballe dans un ActionPrepost et on envoie
//     sur l'anneau vers l'initiateur.
//   - Si on EST l'initiateur : on traite localement (raccourci). Sinon le message
//     ferait le tour et reviendrait à nous, mais serait droppé par le self-detect
//     au top de handleControlMessage avant d'atteindre handleSnapshotPrepost,
//     ce qui causerait un deadlock (nbMsgAttendus ne décrémenterait pas pour
//     nos propres préposts).
func (c *Control) sendPrepost(originalMsg *transport.Message) {
	if c.initiateur {
		if c.EG == nil {
			c.EG = newEG()
		}
		c.EG.Preposts = append(c.EG.Preposts, originalMsg.String())
		c.nbMsgAttendus--
		c.log.Info("sendPrepost",
			fmt.Sprintf("prépost local (initiateur), nbMsgAttendus=%d", c.nbMsgAttendus))
		c.checkSnapshotTermination()
		return
	}
	c.sendMessage(transport.Message{
		Type:   transport.TypeControl,
		Action: transport.ActionPrepost,
		Data:   map[string]string{"msg": originalMsg.String()},
	})
	c.log.Info("sendPrepost", "envoi d'un prépost sur l'anneau")
}

// resetSnapshotState remet les compteurs spécifiques au snapshot et la couleur
// à blanc pour permettre un futur snapshot. Ne PAS toucher à `bilan` : c'est
// un cumul depuis le démarrage du site, et la cohérence de Σ bilan = nb_msg_en_transit
// est préservée tant qu'on ne le reset jamais (les sends/receives suivants
// s'équilibrent normalement).
func (c *Control) resetSnapshotState() {
	c.couleur = transport.ColorWhite
	c.initiateur = false
	c.nbEtatsAttendus = 0
	c.nbMsgAttendus = 0
	// EG conservé pour consultation ultérieure si besoin
	// snapshotPending et pendingControlSnap déjà à false/nil à ce stade
}
