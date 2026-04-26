// Package control - s'occupe de la communication entre nos sites, comment partager l'état du jeu, choisir les phases, etc.
// state.go - permet de définir les structures de données représentant l'état du jeu côté controle, qui sont différentes de celles utilisées côté application (filtrées)

package control

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
)

type Player struct {
	ID    string `json:"id"`
	Role  Role   `json:"role"`
	Alive bool   `json:"alive"`
}

type GameState struct {
	Phase     Phase             `json:"phase"`
	Players   map[string]Player `json:"players"`
	Votes     map[string]string `json:"votes"`
	KillWolf  string            `json:"killWolf"`
	KillWitch string            `json:"killWitch"`
	Winner    string            `json:"winner"`
	MyID      string            `json:"myId"`
}

// ensemble des IDs ayant confirmé une action
type readySet map[string]bool

func newGameState(myID string) GameState {
	return GameState{
		Phase:   PhaseLobby,
		Players: make(map[string]Player),
		Votes:   make(map[string]string),
		MyID:    myID,
	}
}