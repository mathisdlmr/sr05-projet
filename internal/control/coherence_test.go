package control

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

// makeTestControl crée un Control dont les sorties sont capturées dans un buffer.
func makeTestControl(id string, nbSites int) (*Control, *bytes.Buffer) {
	out := &bytes.Buffer{}
	io := transport.NewIOFromReadWriter(&bytes.Buffer{}, out)
	log := logger.New(id)
	return New(id, nbSites, io, log), out
}

// voteState retourne un GameState en phase VOTE avec 4 joueurs vivants.
func voteState() GameState {
	return GameState{
		Phase: PhaseVote,
		Players: map[string]Player{
			"J1": {ID: "J1", Role: RoleWolf, Alive: true},
			"J2": {ID: "J2", Role: RoleVillager, Alive: true},
			"J3": {ID: "J3", Role: RoleVillager, Alive: true},
			"J4": {ID: "J4", Role: RoleWitch, Alive: true},
		},
		Votes: map[string]string{},
	}
}

// TestDoubleTransition démontre que deux sites exécutent tous les deux la transition
// de phase quand leur condition locale est satisfaite — sans aucune coordination.
// Ce comportement est déterministe : il se produit à chaque exécution.
//
// Ce test ÉCHOUE dans la version actuelle (bug démontré).
// Il doit PASSER après l'implémentation de la file d'attente répartie
// (un seul site exécutera la transition).
func TestDoubleTransition(t *testing.T) {
	votes := map[string]string{
		"J1": "J2", "J2": "J2", "J3": "J2", "J4": "J2",
	}

	ctrl1, out1 := makeTestControl("J1", 4)
	ctrl2, out2 := makeTestControl("J2", 4)

	for _, c := range []*Control{ctrl1, ctrl2} {
		c.state = voteState()
		c.state.Votes = votes
		c.villageVotes = votes
	}

	ctrl1.tryResolveVote()
	ctrl2.tryResolveVote()

	transitioned1 := strings.Contains(out1.String(), string(PhaseNight))
	transitioned2 := strings.Contains(out2.String(), string(PhaseNight))

	t.Logf("Site J1 a transitionné : %v", transitioned1)
	t.Logf("Site J2 a transitionné : %v", transitioned2)

	if transitioned1 && transitioned2 {
		t.Errorf(
			"BUG: double transition — les deux sites ont exécuté resolveVote() sans coordination.\n"+
				"Attendu (avec file d'attente) : un seul site transitionne.\n"+
				"Observé : J1=%v  J2=%v",
			transitioned1, transitioned2,
		)
	}
}

// TestTiedVoteNonDeterminism démontre que resolveVote() est non-déterministe
// en cas d'égalité de votes : deux sites avec le même état peuvent éliminer
// des joueurs différents selon l'ordre d'itération de la map Go.
//
// Ce test ÉCHOUE dans la version actuelle (bug démontré).
// Il doit PASSER après l'implémentation d'un départage déterministe.
func TestTiedVoteNonDeterminism(t *testing.T) {
	// J2 et J3 ont chacun 2 votes : égalité parfaite.
	tiedVotes := map[string]string{
		"J1": "J2",
		"J2": "J3",
		"J3": "J2",
		"J4": "J3",
	}

	eliminated := make(map[string]int)
	const runs = 200

	for i := 0; i < runs; i++ {
		c, _ := makeTestControl("J1", 4)
		c.state = voteState()
		c.state.Votes = tiedVotes
		c.villageVotes = tiedVotes
		c.resolveVote()

		for id, p := range c.state.Players {
			if !p.Alive {
				eliminated[id]++
			}
		}
	}

	t.Logf("Résultats sur %d appels à resolveVote() avec égalité :", runs)
	for id, count := range eliminated {
		t.Logf("  %s éliminé %d fois (%.0f%%)", id, count, float64(count)/float64(runs)*100)
	}

	if len(eliminated) > 1 {
		t.Errorf(
			"BUG: résultat non-déterministe — %d joueurs différents éliminés selon l'ordre d'itération de la map.\n"+
				"Deux sites avec le même état peuvent prendre des décisions différentes.",
			len(eliminated),
		)
	}
}
