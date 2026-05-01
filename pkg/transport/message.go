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
)

// Types de messages
const (
	CriticalSection = "critical_section"
	ControlMessage  = "control_message"
	DataMessage     = "data_message"
)

// Message représente un type de message avec horodatage, expéditeur et des données structurées.
type Message struct {
	Type      string
	Timestamp *int
	Sender    int
	Data      map[string]string
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
	msgType := ""
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
				msgType = parts[1]
			case "timestamp":
				if parsed, err := strconv.Atoi(parts[1]); err == nil {
					timestamp = &parsed
				}
			case "sender":
				sender, _ = strconv.Atoi(parts[1])
			default:
				data[parts[0]] = parts[1]
			}
		}
	}

	return &Message{
		Type:      msgType,
		Timestamp: timestamp,
		Sender:    sender,
		Data:      data,
	}, nil
}

// String sérialise le Message en utilisant format spécifié.
func (m *Message) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s%stype%s%s", fieldSep, keyValSep, keyValSep, m.Type))
	if m.Timestamp != nil {
		builder.WriteString(fmt.Sprintf("%s%stimestamp%s%d", fieldSep, keyValSep, keyValSep, *m.Timestamp))
	}
	builder.WriteString(fmt.Sprintf("%s%ssender%s%d", fieldSep, keyValSep, keyValSep, m.Sender))

	for k, v := range m.Data {
		builder.WriteString(fmt.Sprintf("%s%s%s%s%s", fieldSep, keyValSep, k, keyValSep, v))
	}

	return builder.String()
}
