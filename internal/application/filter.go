// Package application - Contient la logique du jeu, et le filtrage entre le client (server) et le système réparti (control)
// filtre.go — masque les informations que le joueur local ne doit pas voir : 
// Règles de visibilité :
//   - Chaque joueur connaît son propre rôle.
//   - Les joueurs morts ont leur rôle révélé pour tout le monde.
//   - Les loups se connaissent entre eux.
//   - La cible des loups (KillWolf) n'est visible que par les loups et la sorcière (phase WITCH).
//   - Les votes sont publics pendant la phase VOTE et END.

package application

// BuildView - construit la vue filtrée de l'état à envoyer au navigateur
func BuildView(state GameState, myID string, myRole Role) GameState {
	view := GameState{
		Phase:     state.Phase,
		Players:   make(map[string]Player),
		Votes:     make(map[string]string),
		KillWolf:  state.KillWolf,
		KillWitch: state.KillWitch,
		Winner:    state.Winner,
		MyID:      myID,
	}

	for id, p := range state.Players {
		visible := Player{ID: id, Alive: p.Alive, Role: RoleUnknown}

		switch {
		case id == myID:
			visible.Role = p.Role // on connait son propre rôle
		case !p.Alive:
			visible.Role = p.Role // les morts sont révélés
		case myRole == RoleWolf && p.Role == RoleWolf:
			visible.Role = p.Role // les loups se connaissent
		}

		view.Players[id] = visible
	}

	if state.Phase == PhaseVote || state.Phase == PhaseEnd { // les votes ne sont publics que pendant VOTE et END
		for voter, target := range state.Votes {
			view.Votes[voter] = target
		}
	}

	canSeeKill := myRole == RoleWolf || (myRole == RoleWitch && state.Phase == PhaseWitch) // la cible des loups : visible uniquement par les loups et la sorcière en phase WITCH
	if !canSeeKill {
		view.KillWolf = ""
	}

	view.KillWitch = "" // la décision de la sorcière n'est jamais transmise au navigateur

	return view
}