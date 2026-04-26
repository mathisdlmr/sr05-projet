// Package application - Contient la logique du jeu, et le filtrage entre le client (server) et le système réparti (control)
// state.go - contient les structures de données représentant l'état du jeu côté application

package application

type Phase string

const (
	PhaseLobby Phase = "LOBBY"
	PhaseNight Phase = "NIGHT"
	PhaseWitch Phase = "WITCH"
	PhaseVote  Phase = "VOTE"
	PhaseEnd   Phase = "END"
)

type Role string

const (
	RoleWolf     Role = "WOLF"
	RoleVillager Role = "VILLAGER"
	RoleWitch    Role = "WITCH"
	RoleUnknown  Role = "?"
)

type Player struct {
	ID    string `json:"id"`
	Role  Role   `json:"role"`
	Alive bool   `json:"alive"`
}

type GameState struct {
	Phase     Phase             `json:"phase"`
	Players   map[string]Player `json:"players"`
	Votes     map[string]string `json:"votes"`     // votant : cible
	KillWolf  string            `json:"killWolf"`
	KillWitch string            `json:"killWitch"`
	Winner    string            `json:"winner"`    // "" | "WOLVES" | "VILLAGERS"
	MyID      string            `json:"myId"`      // identifiant du joueur local
}

func NewGameState(myID string) GameState {
	return GameState{
		Phase:   PhaseLobby,
		Players: make(map[string]Player),
		Votes:   make(map[string]string),
		MyID:    myID,
	}
}