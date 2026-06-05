// Package transport - gère la communication inter-processus partagé par les 3 binaires : application, control et server
// message.go - gère la construction et lectures des messages envoyés/reçus
//
// Les messages sont structurés autour de champs fixes (type, timestamp, sender)
// et de données supplémentaires (data) formatées à plat (flat key-value).
// Format des messages sur stdin/stdout :
//  <separateur_de_champ><séparateur_de_clé><clé><séparateur_de_clé><valeur>...
//
// Exemple :
//  /=timestamp=17/=sender=1/=type=vote/=player=J1
//
// On transforme ça en donnée structurée :
// Message{
// Type: "Control",
// Timestamp: 17,
// Sender: 1,
// Data: map[string]string{
//   "type": "vote",
//   "player": "J1",
// }
//}

package transport

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	fieldSep  = "/"
	keyValSep = "="

	// Séparateurs alternatifs utilisés quand une valeur (typiquement un JSON
	// dans Data) contient les séparateurs par défaut. Le parser les détecte
	// dynamiquement (cf. ParseMessage qui lit sep et equ depuis le wire).
	altFieldSep  = "\x01"
	altKeyValSep = "\x02"
)

// Couleurs pour l'algorithme d'instantané (lestage Lai-Yang, algo 11).
const (
	ColorWhite = "white"
	ColorRed   = "red"
)

// Types de messages

type MessageType string

const (
	TypeControl     MessageType = "control"
	TypeApplication MessageType = "application"
)

type Action string

// Types d'actions
const (

	// Envoyé par app au contrôleur, puis par le contrôleur aux autres sites
	ActionRequestCS Action = "requestCS"

	// Entre les contrôles
	ActionReleaseCS    Action = "releaseCS"
	ActionAcknowlegeCS Action = "acknowlegeCS"

	// Depuis et vers l'application
	ActionEndCS   Action = "endCS"
	ActionBeginCS Action = "beginCS"

	// Snapshot
	ActionStartSnapshot    Action = "startSnapshot"
	ActionSnapshotState    Action = "snapshotState"
	ActionState            Action = "state"
	ActionPrepost          Action = "prepost"
	ActionSnapshotComplete Action = "snapshotComplete"
	ActionRestoreSnapshot  Action = "restoreSnapshot"

	// ActionSnapshotRejected : envoyé par le Control à son App quand un nouveau
	// snapshot est demandé alors qu'un autre est déjà en cours.
	ActionSnapshotRejected Action = "snapshotRejected"

	// ActionWakeup : envoyé par l'initiateur après sa bascule pour réveiller
	// les autres sites quand il n'y a pas assez de trafic applicatif
	// (cf. exo 127/128 du poly). Pas lesté en couleur, pas compté dans le
	// bilan : son handler déclenche la bascule directement, sinon le bilan
	// ne s'équilibre plus entre snapshots successifs.
	ActionWakeup Action = "wakeup"
)

// Message représente un type de message avec horodatage, expéditeur et des données structurées.
type Message struct {
	Type        MessageType // type de message : control (d'un site à l'autre) ou application (de l'application locale au contrôle)
	Action      Action      // champs dédié pour communiquer l'action : enterCS, endCS, startSauvegarde
	Timestamp   *int
	VectorClock map[int]int // horloge vectorielle : map d'ID du site vers sa valeur
	Color       string      // lestage Lai-Yang : "" si non-applicatif, ColorWhite ou ColorRed sinon
	Sender      int
	Data        map[string]string
}

// ParseMessage construit un objet Message à partir de la chaîne de caractères formatée.
func ParseMessage(msg string) (*Message, error) {
	if len(msg) < 4 {
		return nil, fmt.Errorf("message too short")
	}

	// Parse fields
	sep := msg[0:1]
	pairs := strings.Split(msg[1:], sep)

	var timestamp *int
	sender := 0
	msgType := MessageType("")
	var msgAction Action
	var msgVectorClock map[int]int
	color := ""
	data := make(map[string]string)

	for _, pair := range pairs {
		if len(pair) < 3 {
			continue
		}
		equ := pair[0:1]
		parts := strings.SplitN(pair[1:], equ, 2)
		if len(parts) == 2 {
			switch parts[0] {
			case "type":
				msgType = MessageType(parts[1])
			case "action":
				msgAction = Action(parts[1])
			case "timestamp":
				if parsed, err := strconv.Atoi(parts[1]); err == nil {
					timestamp = &parsed
				}
			case "vectorClock":
				// Parse vectorClock comme map: clé1,valeur1;clé2,valeur2;...
				if parts[1] != "" {
					msgVectorClock = make(map[int]int)
					pairs := strings.Split(parts[1], ";")
					for _, pair := range pairs {
						kv := strings.SplitN(pair, ":", 2)
						if len(kv) == 2 {
							if k, err := strconv.Atoi(strings.TrimSpace(kv[0])); err == nil {
								if v, err := strconv.Atoi(strings.TrimSpace(kv[1])); err == nil {
									msgVectorClock[k] = v
								}
							}
						}
					}
				}
			case "sender":
				sender, _ = strconv.Atoi(parts[1])
			case "color":
				color = parts[1]
			default:
				data[parts[0]] = parts[1]
			}
		}
	}

	return &Message{
		Type:        msgType,
		Action:      msgAction,
		Timestamp:   timestamp,
		VectorClock: msgVectorClock,
		Color:       color,
		Sender:      sender,
		Data:        data,
	}, nil
}

// String sérialise le Message. Les séparateurs par défaut sont "/" et "=",
// mais on bascule sur \x01 / \x02 si une valeur dans Data contient déjà
// l'un de ces caractères (typiquement quand on transporte du JSON).
// ParseMessage lit les séparateurs depuis les deux premiers octets du wire,
// donc le switch est transparent pour le destinataire.
func (m Message) String() string {
	field, kv := m.pickSeparators()
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s%stype%s%s", field, kv, kv, m.Type))
	builder.WriteString(fmt.Sprintf("%s%saction%s%s", field, kv, kv, m.Action))
	if m.Timestamp != nil {
		builder.WriteString(fmt.Sprintf("%s%stimestamp%s%d", field, kv, kv, *m.Timestamp))
	}
	// Sérialiser vectorClock map comme clé:valeur;clé:valeur
	if len(m.VectorClock) > 0 {
		parts := make([]string, 0, len(m.VectorClock))
		for k, v := range m.VectorClock {
			parts = append(parts, strconv.Itoa(k)+":"+strconv.Itoa(v))
		}
		builder.WriteString(fmt.Sprintf("%s%svectorClock%s%s", field, kv, kv, strings.Join(parts, ";")))
	}
	if m.Color != "" {
		builder.WriteString(fmt.Sprintf("%s%scolor%s%s", field, kv, kv, m.Color))
	}
	builder.WriteString(fmt.Sprintf("%s%ssender%s%d", field, kv, kv, m.Sender))

	for k, v := range m.Data {
		builder.WriteString(fmt.Sprintf("%s%s%s%s%s", field, kv, k, kv, v))
	}

	return builder.String()
}

// pickSeparators choisit les séparateurs à utiliser pour la sérialisation
// en fonction du contenu : si une valeur de Data contient "/" ou "=",
// on bascule sur les séparateurs alternatifs pour ne pas casser le parsing.
func (m Message) pickSeparators() (field, kv string) {
	for _, v := range m.Data {
		if strings.ContainsAny(v, fieldSep+keyValSep) {
			return altFieldSep, altKeyValSep
		}
	}
	return fieldSep, keyValSep
}
