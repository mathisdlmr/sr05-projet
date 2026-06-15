package application

import "encoding/json"

const (
	NullPlayerID string = "NULLPLAYERID"
)

// handleDistributedAction - traite les actions reçues des autres joueurs via la communication SC,
// met à jour l'état local et notifie le navigateur
func (a *App) handleDistributedAction(data map[string]string) {
	voterID := data["voter"]
	targetID := data["target"]

	switch data["cmd"] {
	case "join":
		a.applyJoin(data["player"])

	case "start":
		a.applyStart(data)

	case "wolfkill":
		if a.state.Phase != PhaseNight {
			a.log.Warn("handleDistributedAction", "wolfkill ignoré hors phase NIGHT")
			return
		}
		a.state.Votes[voterID] = targetID
		evt := map[string]interface{}{
			"type":  "wolfVoted",
			"voter": voterID,
		}
		if a.myRole == RoleWolf {
			evt["target"] = targetID
		}
		a.pushEvent(evt)
		if a.checkAllVotesCompleted() {
			target, valid := a.computeVoteResults()
			if valid {
				a.state.KillWolf = target
			}
			a.transitionToWitch()
		}

	case "witchsave":
		if a.state.Phase != PhaseWitch {
			a.log.Warn("handleDistributedAction", "witchsave ignoré hors phase WITCH")
			return
		}
		a.state.KillWolf = ""
		a.transitionToVote()

	case "witchkill":
		if a.state.Phase != PhaseWitch {
			a.log.Warn("handleDistributedAction", "witchkill ignoré hors phase WITCH")
			return
		}
		a.state.KillWitch = targetID
		a.transitionToVote()

	case "witchskip":
		if a.state.Phase != PhaseWitch {
			a.log.Warn("handleDistributedAction", "witchskip ignoré hors phase WITCH")
			return
		}
		a.transitionToVote()

	case "vote":
		if a.state.Phase != PhaseVote {
			a.log.Warn("handleDistributedAction", "vote ignoré hors phase VOTE")
			return
		}
		a.state.Votes[voterID] = targetID
		a.pushEvent(map[string]interface{}{
			"type":   "voted",
			"voter":  voterID,
			"target": targetID,
		})
		if a.checkAllVotesCompleted() {
			a.applyVoteResult()
			a.transitionToNight()
		}
	case "attribution":

		role := a.pickRole()
		a.pending.data["cmd"] = "applyattribution"
		a.pending.data["role"] = string(role)
		a.pending.data["id"] = a.myID
		a.myRole = role
		a.applyAttribution(a.myID, role)
		if a.checkEveryoneHasRole() {
			a.transitionFromStart()
		}

	case "applyattribution":
		a.applyAttribution(data["id"], Role(data["role"]))
		if a.checkEveryoneHasRole() {
			a.transitionFromStart()
		}

	case "restart":
		a.state = NewGameState(a.myID)
		a.myRole = RoleUnknown
		a.spectating = false
		a.needsRejoin = true
		a.pushEvent(map[string]interface{}{
			"type": "gameRestart",
		})
		a.sendInit()
		a.log.Info("handleDistributedAction", "partie redémarrée, retour en LOBBY")

	default:
		a.log.Warn("handleDistributedAction", "action inconnue: "+data["cmd"])
	}
}

// applyDepart - marque un joueur comme mort suite à son départ du réseau
func (a *App) applyDepart(playerID string) {
	if _, ok := a.state.Players[playerID]; !ok {
		a.log.Warn("applyDepart", "joueur inconnu: "+playerID)
		return
	}

	p := a.state.Players[playerID]
	p.Alive = false
	a.state.Players[playerID] = p
	delete(a.state.Votes, playerID)

	a.pushEvent(map[string]interface{}{
		"type":     "playerLeft",
		"playerId": playerID,
	})
	a.log.Info("applyDepart", "joueur parti: "+playerID)

	if a.checkEndOfGame() && a.state.Phase != PhaseEnd {
		a.sendGameEnd()
		return
	}

	// On vérifie en plus si le départ débloque la phase courante
	switch a.state.Phase {
	case PhaseNight:
		if a.checkAllVotesCompleted() {
			target, valid := a.computeVoteResults()
			if valid {
				a.state.KillWolf = target
			}
			a.transitionToWitch()
		}
	case PhaseVote:
		if a.checkAllVotesCompleted() {
			a.applyVoteResult()
			a.transitionToNight()
		}
	case PhaseWitch:
		witchAlive := false
		for _, p := range a.state.Players {
			if p.Role == RoleWitch && p.Alive {
				witchAlive = true
				break
			}
		}
		if !witchAlive {
			a.transitionToVote()
		}
	}
}

// handleNewSiteInit - restaure l'état du jeu reçu du parrain et passe en mode spectateur
func (a *App) handleNewSiteInit(stateJSON string) {
	var gs GameState
	if err := json.Unmarshal([]byte(stateJSON), &gs); err != nil {
		a.log.Error("handleNewSiteInit", "unmarshal state: "+err.Error())
		return
	}
	gs.MyID = a.myID
	a.state = gs
	a.myRole = RoleUnknown
	a.spectating = true
	a.applyJoin(a.myID)
	a.sendInit()
	a.log.Info("handleNewSiteInit", "état reçu du parrain, mode spectateur activé")
}

// applyJoin - ajoute un joueur à l'état local, notifie le navigateur et les autres joueurs via une SC "join"
func (a *App) applyJoin(playerID string) {
	if _, ok := a.state.Players[playerID]; ok {
		return
	}
	a.state.Players[playerID] = Player{ID: playerID, Role: RoleUnknown, Alive: true}
	a.pushEvent(map[string]interface{}{
		"type":     "playerJoined",
		"playerId": playerID,
	})
	a.log.Info("applyJoin", "joueur rejoint: "+playerID)
}

// applyStart - distribue les rôles, initialise les votes et passe en phase NIGHT
func (a *App) applyStart(data map[string]string) {
	if a.state.Phase != PhaseLobby {
		a.log.Warn("applyStart", "start ignoré hors phase LOBBY")
		return
	}

	if data["voter"] == a.myID {
		// Choisir son rôle et le mettre dans le message à envoyer
		role := a.pickRole()
		a.applyAttribution(a.myID, role)
		a.myRole = role
		a.pending.data["role"] = string(role)
		a.pending.data["id"] = a.myID

	} else {
		// Appliquer le rôle reçu et requestCS
		a.applyAttribution(data["id"], Role(data["role"]))
		a.requestCS(map[string]string{
			"cmd": "attribution",
		})
	}
}

// Used only at Vote Phase
// applyVoteResult - élimine le joueur le plus voté
func (a *App) applyVoteResult() {
	target, valid := a.computeVoteResults()

	if valid {
		a.killPlayer(target)
	}

	if a.checkEndOfGame() {
		if valid {
			a.pushEvent(map[string]interface{}{
				"type":      "voteEliminated",
				"playerId":  target,
				"nextPhase": "END",
			})
		}
		a.sendGameEnd()
		return
	}

	if valid {
		a.pushEvent(map[string]interface{}{
			"type":      "voteEliminated",
			"playerId":  target,
			"nextPhase": "NIGHT",
		})
	}

	a.log.Info("applyVoteResult", "vote résolu")
}

// killPlayer - met à jour l'état pour marquer un joueur comme éliminé
func (a *App) killPlayer(targetID string) {
	if p, ok := a.state.Players[targetID]; ok {
		p.Alive = false
		a.state.Players[targetID] = p
		a.log.Info("killPlayer", "joueur éliminé: "+targetID)
	}

}

// computeVoteResults - calcule le joueur le plus voté à partir de a.state.Votes
// Retourne le playerID du plus voté et un booléen indiquant si le résultat est valide (non nul)
func (a *App) computeVoteResults() (string, bool) {
	scores := map[string]int{}
	for _, target := range a.state.Votes {
		if target == "" {
			continue
		}
		scores[target]++
	}

	maxValue := 0
	maxTarget := ""
	for target, value := range scores {
		if value > maxValue {
			maxValue = value
			maxTarget = target
		}
	}

	if maxTarget == NullPlayerID || maxTarget == "" {
		return "", false
	}
	return maxTarget, true
}

// checkEndOfGame - vérifie si la partie est terminée (tous les loups sont morts ou les loups ont l'avantage numérique)
func (a *App) checkEndOfGame() bool {
	allWolvesDead := true
	for _, p := range a.state.Players {
		if p.Alive && p.Role == RoleWolf {
			allWolvesDead = false
			break
		}
	}
	if allWolvesDead {
		return true
	}

	nbWolves, nbVillagers := 0, 0
	for _, p := range a.state.Players {
		if !p.Alive {
			continue
		}
		if p.Role == RoleWolf {
			nbWolves++
		} else {
			nbVillagers++
		}
	}
	return nbWolves >= nbVillagers
}

// checkAllVotesCompleted - vérifie si tous les joueurs qui doivent voter ont voté (aucune valeur vide dans a.state.Votes)
func (a *App) checkAllVotesCompleted() bool {
	if len(a.state.Votes) == 0 {
		return false
	}
	for _, vote := range a.state.Votes {
		if vote == "" {
			return false
		}
	}
	return true
}

// createStartingVoteMap - initialise a.state.Votes avec les joueurs qui doivent voter pour la phase donnée, avec des votes vides
func (a *App) createStartingVoteMap(phase Phase) {
	var votersRole Role
	switch phase {
	case PhaseNight:
		votersRole = RoleWolf
	case PhaseVote:
		votersRole = RoleAny
	case PhaseWitch:
		votersRole = RoleWitch
	}

	a.state.Votes = make(map[string]string)

	for playerID, player := range a.state.Players {
		if !player.Alive {
			continue
		}
		if player.Role == votersRole || votersRole == RoleAny {
			a.state.Votes[playerID] = ""
		}
	}
}
