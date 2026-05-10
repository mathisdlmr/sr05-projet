// Côté application : gère les messages snapshot venant du Control local.
// Référence : docs/snapshot-design.md § "Bridge côté App".

package application

import (
	"encoding/json"

	"github.com/sr05-projet/pkg/transport"
)

// deepCopyGameState retourne une copie complète et indépendante du GameState
// via un aller-retour JSON. Pratique : pas de risque d'oublier une map ou
// un slice, et le coût est négligeable (GameState reste petit).
func deepCopyGameState(s GameState) GameState {
	raw, err := json.Marshal(s)
	if err != nil {
		// Ne devrait pas arriver, GameState est constitué de types JSON-safe.
		return s
	}
	var out GameState
	_ = json.Unmarshal(raw, &out)
	return out
}

// handleSnapshotStateRequest est appelée quand le Control local demande à l'App
// son état pour la prise d'instantané. On deep-copie, on sérialise, on répond
// dans le même format (Action=ActionSnapshotState avec role=response).
func (a *App) handleSnapshotStateRequest() {
	snap := deepCopyGameState(a.state)
	stateJSON, err := json.Marshal(snap)
	if err != nil {
		a.log.Error("handleSnapshotStateRequest", "marshal state: "+err.Error())
		return
	}

	resp := transport.Message{
		Type:   transport.TypeApplication,
		Action: transport.ActionSnapshotState,
		Sender: a.siteID,
		Data: map[string]string{
			"role":  "response",
			"state": string(stateJSON),
		},
	}
	if err := a.io.Send(resp.String()); err != nil {
		a.log.Error("handleSnapshotStateRequest", "envoi réponse: "+err.Error())
		return
	}
	a.log.Info("handleSnapshotStateRequest", "GameState envoyé au Control pour snapshot")
}

// handleSnapshotRestore est appelée quand le Control reçoit l'EG final (via la
// diffusion ActionSnapshotComplete sur l'anneau) et le pousse à l'App pour
// affichage navigateur. On dé-sérialise et on transmet au navigateur tel quel.
func (a *App) handleSnapshotRestore(egJSON string) {
	var eg map[string]interface{}
	if err := json.Unmarshal([]byte(egJSON), &eg); err != nil {
		a.log.Error("handleSnapshotRestore", "unmarshal EG: "+err.Error())
		return
	}
	a.pushEvent(map[string]interface{}{
		"type": "snapshot_received",
		"eg":   eg,
	})
	a.log.Success("handleSnapshotRestore", "sauvegarde répartie reçue, transmise au navigateur")
}
