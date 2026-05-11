package application

func (a *App) transitionToWitch() {
	a.state.Phase = PhaseWitch
	a.state.Votes = make(map[string]string)

	evt := map[string]interface{}{
		"type":  "phaseChange",
		"phase": "WITCH",
	}
	if a.myRole == RoleWitch && a.state.KillWolf != "" {
		evt["killWolf"] = a.state.KillWolf
	}
	a.pushEvent(evt)

	// Si la sorcière est morte, on passe directement au vote du village
	witchAlive := false
	for _, p := range a.state.Players {
		if p.Role == RoleWitch && p.Alive {
			witchAlive = true
			break
		}
	}
	if !witchAlive {
		a.log.Info("transitionToWitch", "sorcière morte, auto-passage à VOTE")
		a.transitionToVote()
	}
}

// transitionToVote - applique les kills de la nuit et passe en phase VOTE
func (a *App) transitionToVote() {
	killed := []string{}
	if a.state.KillWolf != "" {
		a.killPlayer(a.state.KillWolf)
		killed = append(killed, a.state.KillWolf)
	}
	if a.state.KillWitch != "" {
		a.killPlayer(a.state.KillWitch)
		killed = append(killed, a.state.KillWitch)
	}
	a.state.KillWolf = ""
	a.state.KillWitch = ""

	if a.checkEndOfGame() {
		a.pushEvent(map[string]interface{}{
			"type":      "nightKills",
			"killed":    killed,
			"nextPhase": "END",
		})
		a.sendGameEnd()
		return
	}

	a.createStartingVoteMap(PhaseVote)
	a.state.Phase = PhaseVote

	a.pushEvent(map[string]interface{}{
		"type":      "nightKills",
		"killed":    killed,
		"nextPhase": "VOTE",
	})
	a.log.Info("transitionToVote", "passage en phase VOTE")
}

func (a *App) transitionFromStart() {
	a.transitionToNight()
	a.pushEvent(map[string]interface{}{
		"type":    "gameStart",
		"myRole":  string(a.myRole),
		"players": a.buildFilteredPlayers(),
	})

	a.log.Info("transitionFromStart", "partie démarrée, rôle local: "+string(a.myRole))

}

func (a *App) transitionToNight() {
	a.createStartingVoteMap(PhaseNight)
	a.state.Phase = PhaseNight
	a.log.Info("transitionToNight", "passage en phase NIGHT")
}
