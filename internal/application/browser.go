package application

import "encoding/json"

type browserAction struct {
	Action string `json:"action"`
	Target string `json:"target,omitempty"`
	Msg    string `json:"msg,omitempty"`
}

// handleBrowserConnect - appelé à chaque nouvelle connexion WebSocket depuis le navigateur
// Ajoute le joueur à l'état local s'il n'existe pas déjà, envoie l'init et notifie les autres joueurs via une SC "join"
func (a *App) handleBrowserConnect() {
	a.log.Info("handleBrowserConnect", "navigateur connecté")
	if _, ok := a.state.Players[a.myID]; !ok {
		a.state.Players[a.myID] = Player{ID: a.myID, Role: RoleUnknown, Alive: true}
	}
	a.sendInit()
	a.requestCS(map[string]string{
		"cmd":    "join",
		"player": a.myID,
	})
}

// handleFromBrowser - appelé à chaque message JSON reçu du navigateur
func (a *App) handleFromBrowser(raw string) {
	a.log.Debug("browser->app", raw)

	var action browserAction
	if err := json.Unmarshal([]byte(raw), &action); err != nil {
		a.log.Warn("handleFromBrowser", "parse: "+err.Error())
		return
	}
	if action.Action == "" {
		a.log.Warn("handleFromBrowser", "action vide, ignoré")
		return
	}
	if action.Action == "init" {
		a.sendInit()
		return
	}
	data := map[string]string{
		"cmd":   action.Action,
		"voter": a.myID,
	}
	if action.Target != "" {
		data["target"] = action.Target
	}
	a.requestCS(data)
	a.log.Info("handleFromBrowser", "SC demandée pour: "+action.Action)
}

// getVisibleVotes - retourne les votes visibles pour ce joueur (sans les vides)
func (a *App) getVisibleVotes() map[string]string {
	result := map[string]string{}
	switch a.state.Phase {
	case PhaseNight:
		if a.myRole == RoleWolf {
			for k, v := range a.state.Votes {
				if v != "" {
					result[k] = v
				}
			}
		}
	case PhaseVote:
		for k, v := range a.state.Votes {
			if v != "" {
				result[k] = v
			}
		}
	}
	return result
}

// Construit l'état du jeu avec seulement les parties de l'état du jeu visible au joueur
func (a *App) buildFilteredPlayers() map[string]interface{} {
	result := make(map[string]interface{})
	for id, p := range a.state.Players {
		visibleRole := "?"
		switch {
		case id == a.myID:
			visibleRole = string(p.Role)
		case a.myRole == RoleWolf && p.Role == RoleWolf:
			visibleRole = string(p.Role)
		case a.state.Phase == PhaseEnd:
			visibleRole = string(p.Role)
		}
		result[id] = map[string]interface{}{
			"id":    id,
			"role":  visibleRole,
			"alive": p.Alive,
		}
	}
	return result
}

// sendInit - envoie l'état du jeu au navigateur
func (a *App) sendInit() {
	p, ok := a.state.Players[a.myID]
	if !ok {
		p = Player{ID: a.myID, Alive: true}
	}
	evt := map[string]interface{}{
		"type":    "init",
		"phase":   string(a.state.Phase),
		"myId":    a.myID,
		"myRole":  string(a.myRole),
		"myAlive": p.Alive,
		"players": a.buildFilteredPlayers(),
		"votes":   a.getVisibleVotes(),
	}
	if a.state.Phase == PhaseWitch && a.myRole == RoleWitch && a.state.KillWolf != "" {
		evt["killWolf"] = a.state.KillWolf
	}
	a.pushEvent(evt)
}

// pushEvent - envoie un event (une map convertie en JSON) au navigateur
func (a *App) pushEvent(evt map[string]interface{}) {
	out, err := json.Marshal(evt)
	if err != nil {
		a.log.Error("pushEvent", "marshal: "+err.Error())
		return
	}
	if err := a.srv.Send(string(out)); err != nil {
		a.log.Warn("pushEvent", "send: "+err.Error())
	}
}

// sendGameEnd - calcule le vainqueur, met à jour l'état et notifie le navigateur
func (a *App) sendGameEnd() {
	allWolvesDead := true
	for _, p := range a.state.Players {
		if p.Alive && p.Role == RoleWolf {
			allWolvesDead = false
			break
		}
	}

	winner := "VILLAGERS"
	if !allWolvesDead {
		winner = "WOLVES"
	}

	a.state.Phase = PhaseEnd
	a.state.Winner = winner

	allPlayers := make(map[string]interface{})
	for id, p := range a.state.Players {
		allPlayers[id] = map[string]interface{}{
			"id":    id,
			"role":  string(p.Role),
			"alive": p.Alive,
		}
	}

	a.pushEvent(map[string]interface{}{
		"type":    "gameEnd",
		"winner":  winner,
		"players": allPlayers,
	})
	a.log.Info("sendGameEnd", "fin de partie, vainqueur: "+winner)
}

