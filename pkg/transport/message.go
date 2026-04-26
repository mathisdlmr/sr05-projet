// Package transport - gère la communication inter-processus partagé par les 3 binaires : application, control et server
// message.go - gère la construction et lectures des messages envoyés/reçus
//
// Format des messages sur stdin/stdout :
//  <separateur_de_champ><séparateur_de_clé><clé><séparateur_de_clé><valeur><separateur_de_champ><séparateur_de_clé><clé><séparateur_de_clé><valeur>
//
// Exemples : 
//  ,=snd=arthur,=hlg=38
//  /=snd=arthur-pid~8286/=hlg=(38,4)

package transport

import "strings"

const (
	fieldSep  = "/"
	keyValSep = "="
)

// Field - met un couple (clé, valeur) au format souhaité
func Field(key, val string) string {
	return fieldSep + keyValSep + key + keyValSep + val
}

// Build - construit un message complet à partir d'une liste ordonnée de paires (clé, valeur)
// Les couples sont envoyés dans le même ordre que tels qu'ils sont reçus en paramètre
// Exemple : msg := transport.Build("type", "vote", "player", "J1", "target", "J3")
func Build(keyvals ...string) string {
	if len(keyvals)%2 != 0 {
		panic("transport.Build: nombre impair d'arguments, il manque une clé ou une valeur")
	}
	out := ""
	for i := 0; i < len(keyvals); i += 2 {
		out += Field(keyvals[i], keyvals[i+1])
	}
	return out
}

// Get - retourne la valeur associée à key dans msg, ou "" si absente
func Get(msg, key string) string {
	if len(msg) < 4 {
		return ""
	}
	sep := msg[0:1]
	pairs := strings.Split(msg[1:], sep)
	for _, pair := range pairs {
		if len(pair) < 3 {
			continue
		}
		equ := pair[0:1]
		parts := strings.SplitN(pair[1:], equ, 2)
		if len(parts) == 2 && parts[0] == key {
			return parts[1]
		}
	}
	return ""
}

// Has - retourne true si le champ key est présent dans msg
func Has(msg, key string) bool {
	return Get(msg, key) != ""
}