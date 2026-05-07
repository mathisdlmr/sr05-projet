package application

func (a *App) killPlayer(playerID string) {
	if targetcopy, ok := a.state.Players[playerID]; ok {
		targetcopy.Alive = false
		a.state.Players[a.state.KillWolf] = targetcopy
	} else {
		a.log.Error("killPlayer", "Tried to kill player "+playerID+" but not found in players")
	}
}
