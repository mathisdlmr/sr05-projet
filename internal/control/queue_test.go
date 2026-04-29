package control

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

// makeQueueControl crée un Control avec un buffer de sortie capturé.
func makeQueueControl(id string, nbSites int) (*Control, *bytes.Buffer) {
	out := &bytes.Buffer{}
	io := transport.NewIOFromReadWriter(&bytes.Buffer{}, out)
	log := logger.New(id)
	return New(id, nbSites, io, log), out
}

// containsBroadcast vérifie qu'au moins une ligne du buffer commence par
// "BROADCAST:" et a le type attendu.
func containsBroadcast(s, msgType string) bool {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "BROADCAST:") {
			continue
		}
		if transport.Get(strings.TrimPrefix(line, "BROADCAST:"), "type") == msgType {
			return true
		}
	}
	return false
}

// TestRequestSCBroadcastsRequest : vérifie qu'un appel à requestSC
// (i) incrémente l'horloge Lamport, (ii) marque tab[myIndex] = requête,
// (iii) broadcast un message "type=req", (iv) enregistre la pending action.
// Avec nbSites=2 (pas seul), l'action ne doit PAS s'exécuter immédiatement.
func TestRequestSCBroadcastsRequest(t *testing.T) {
	ctrl, out := makeQueueControl("J1", 2)
	executed := false
	ctrl.requestSC(pendingSC{
		name: "test",
		fn:   func() { executed = true },
	})

	if ctrl.lamport != 1 {
		t.Errorf("lamport attendu = 1, obtenu %d", ctrl.lamport)
	}
	if ctrl.tab[0].Kind != entryRequest || ctrl.tab[0].Date != 1 {
		t.Errorf("tab[0] attendu = (req, 1), obtenu (%s, %d)", ctrl.tab[0].Kind, ctrl.tab[0].Date)
	}
	if ctrl.pending == nil {
		t.Errorf("pending devrait être défini après requestSC")
	}
	if !containsBroadcast(out.String(), "req") {
		t.Errorf("broadcast 'type=req' attendu, sortie:\n%s", out.String())
	}
	if executed {
		t.Errorf("l'action ne devrait pas s'exécuter avant l'entrée en SC (nbSites > 1)")
	}
}

// TestSingleSiteEntersSCImmediately : avec un seul site, la condition
// d'entrée en SC est triviale. requestSC doit exécuter l'action puis libérer.
func TestSingleSiteEntersSCImmediately(t *testing.T) {
	ctrl, out := makeQueueControl("J1", 1)
	executed := false
	ctrl.requestSC(pendingSC{
		name: "test",
		fn:   func() { executed = true },
	})

	if !executed {
		t.Errorf("avec nbSites=1, l'action devrait s'exécuter immédiatement")
	}
	if ctrl.pending != nil {
		t.Errorf("pending devrait être à nil après exécution")
	}
	if ctrl.tab[0].Kind != entryRelease {
		t.Errorf("tab[0] attendu = libération, obtenu %s", ctrl.tab[0].Kind)
	}
	if !containsBroadcast(out.String(), "req") || !containsBroadcast(out.String(), "rel") {
		t.Errorf("broadcasts attendus 'req' puis 'rel', sortie:\n%s", out.String())
	}
}

// TestEntryConditionTieBreakByID : si deux sites ont la même date dans Tab,
// le site avec l'index inférieur a priorité. Un site J2 (index 1) avec
// tab[0] = (req, 5) et tab[1] = (req, 5) ne doit PAS entrer en SC.
func TestEntryConditionTieBreakByID(t *testing.T) {
	ctrl, _ := makeQueueControl("J2", 2)
	executed := false
	ctrl.pending = &pendingSC{name: "test", fn: func() { executed = true }}
	ctrl.tab[0] = tabEntry{Kind: entryRequest, Date: 5}
	ctrl.tab[1] = tabEntry{Kind: entryRequest, Date: 5}

	ctrl.tryEnterSC()

	if executed {
		t.Errorf("J2 ne doit pas entrer en SC : J1 a un index inférieur à date égale")
	}

	// Maintenant on simule la libération de J1 → J2 doit pouvoir entrer.
	ctrl.tab[0] = tabEntry{Kind: entryRelease, Date: 6}
	ctrl.tryEnterSC()

	if !executed {
		t.Errorf("J2 doit entrer en SC après libération de J1")
	}
}

// TestLamportClockMonotonic : à la réception d'un message avec une estampille
// supérieure à l'horloge locale, celle-ci doit avancer à max(local, reçue) + 1.
func TestLamportClockMonotonic(t *testing.T) {
	ctrl, _ := makeQueueControl("J1", 2)
	ctrl.lamport = 2

	ctrl.handleReq("J2", 10)

	if ctrl.lamport < 11 {
		t.Errorf("lamport doit être >= 11 (max(2, 10)+1), obtenu %d", ctrl.lamport)
	}
}

// ============ Harness multi-contrôles pour tests d'intégration ============ //

type testRing struct {
	ids      []string
	controls map[string]*Control
	outs     map[string]*bytes.Buffer
	history  map[string][]string // tous les messages broadcastés par chaque site
}

func newTestRing(ids []string) *testRing {
	r := &testRing{
		ids:      ids,
		controls: make(map[string]*Control),
		outs:     make(map[string]*bytes.Buffer),
		history:  make(map[string][]string),
	}
	for _, id := range ids {
		ctrl, out := makeQueueControl(id, len(ids))
		r.controls[id] = ctrl
		r.outs[id] = out
	}
	return r
}

// pump draine les buffers, dispatche les BROADCAST aux autres contrôles,
// et boucle jusqu'à stabilisation. Enregistre l'historique des messages.
func (r *testRing) pump(t *testing.T) {
	for iter := 0; iter < 1000; iter++ {
		any := false
		for _, from := range r.ids {
			data := r.outs[from].String()
			r.outs[from].Reset()
			for _, line := range strings.Split(data, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				r.history[from] = append(r.history[from], line)
				if !strings.HasPrefix(line, "BROADCAST:") {
					continue // message app-bound, ignoré
				}
				payload := strings.TrimPrefix(line, "BROADCAST:")
				for _, to := range r.ids {
					if to == from {
						continue
					}
					r.controls[to].handleFromControl(payload)
					any = true
				}
			}
		}
		if !any {
			return
		}
	}
	t.Fatalf("pump: pas de stabilisation après 1000 itérations")
}

// countTypeBroadcasts retourne le nombre de broadcasts de type donné émis par from.
func (r *testRing) countTypeBroadcasts(from, msgType string) int {
	n := 0
	for _, msg := range r.history[from] {
		if !strings.HasPrefix(msg, "BROADCAST:") {
			continue
		}
		if transport.Get(strings.TrimPrefix(msg, "BROADCAST:"), "type") == msgType {
			n++
		}
	}
	return n
}

// setupVotePhase met les 3 sites en PhaseVote avec les mêmes votes (tous → J3).
func setupVotePhase(r *testRing) {
	players := map[string]Player{
		"J1": {ID: "J1", Role: RoleVillager, Alive: true},
		"J2": {ID: "J2", Role: RoleVillager, Alive: true},
		"J3": {ID: "J3", Role: RoleWolf, Alive: true},
	}
	votes := map[string]string{"J1": "J3", "J2": "J3", "J3": "J3"}
	for _, id := range r.ids {
		c := r.controls[id]
		c.state.Phase = PhaseVote
		c.state.Players = make(map[string]Player, len(players))
		for k, v := range players {
			c.state.Players[k] = v
		}
		c.state.Votes = make(map[string]string, len(votes))
		c.villageVotes = make(map[string]string, len(votes))
		for k, v := range votes {
			c.state.Votes[k] = v
			c.villageVotes[k] = v
		}
	}
}

// TestQueueIntegrationThreeSites : test end-to-end avec 3 contrôles connectés.
// Tous les sites détectent la condition de fin de vote simultanément.
// Avec 3 joueurs (J1/J2 villageois, J3 loup) et l'élimination unanime de J3,
// la partie se termine immédiatement (END, victoire villageois).
// Exigences :
//   - exactement UN site exécute la transition (un seul broadcast 'state')
//   - les trois sites convergent vers le même état final
//   - aucun site ne reste bloqué avec une demande en SC en suspens
func TestQueueIntegrationThreeSites(t *testing.T) {
	r := newTestRing([]string{"J1", "J2", "J3"})
	setupVotePhase(r)

	// Les 3 sites détectent la même condition et demandent la SC.
	for _, id := range r.ids {
		r.controls[id].tryResolveVote()
	}

	r.pump(t)

	// Compter combien de sites ont broadcasté un état (= ont exécuté la transition).
	stateBroadcasters := 0
	for _, id := range r.ids {
		if r.countTypeBroadcasts(id, "state") > 0 {
			stateBroadcasters++
		}
	}
	if stateBroadcasters != 1 {
		t.Errorf("exactement 1 site doit broadcaster l'état (transition unique), obtenu %d", stateBroadcasters)
		for _, id := range r.ids {
			t.Logf("  %s : %d broadcasts state", id, r.countTypeBroadcasts(id, "state"))
		}
	}

	// Tous les sites doivent converger : Phase=END, J3 mort, victoire villageois,
	// pending vidé (aucune demande en suspens).
	for _, id := range r.ids {
		c := r.controls[id]
		if c.state.Phase != PhaseEnd {
			t.Errorf("site %s : phase attendue END, obtenu %s", id, c.state.Phase)
		}
		if c.state.Players["J3"].Alive {
			t.Errorf("site %s : J3 devrait être mort", id)
		}
		if c.state.Winner != "VILLAGERS" {
			t.Errorf("site %s : winner attendu VILLAGERS, obtenu %q", id, c.state.Winner)
		}
		if c.pending != nil {
			t.Errorf("site %s : pending non nil après stabilisation (%s)", id, c.pending.name)
		}
	}
}
