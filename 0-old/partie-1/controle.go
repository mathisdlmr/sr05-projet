package main

import (
    "fmt"
    "strconv"
    "strings"
)

// Séparateurs
var fieldsep = "/"
var keyvalsep = "="

// msg_format - Met en forme un message pour l'émission à partir des séparateurs définis
func msg_format(key string, val string) string {
    return fieldsep + keyvalsep + key + keyvalsep + val
}

// findval - Retourne la valeur associée à la clé key dans un message msg
func findval(msg string, key string) string {
	// 0. Vérification sur l'input : au moins 4 caractères : séparateur de paires + clé + séparateur de clé-valeur + valeur
	if len(msg) < 4 {
		return ""
	}

	// 1. On récupère le séparateur de paires ainsi que chaque paire
	sep := msg[0:1];
	tab_allkeyvals := strings.Split(msg[1:], sep);   

	// 2. On parcourt les paires et récupère chaque séparateur
	for _, keyval := range tab_allkeyvals {
		equ := keyval[0:1]

		// 3. On récupère la clé de la paire, et vérifie si c'est la bonne
		tabkeyval := strings.Split(keyval[1:], equ)
		if tabkeyval[0] == key {
			return tabkeyval[1] // 3.a. Oui, on retourne la valeur
		}
	}
	return "" // 3.b. Non, on retourne une chaîne vide
}

// recaler - Retourne la valeur de l'horloge locale recalée à partir de l'horloge reçue
func recaler(x, y int) int {
    if x < y {
        return y + 1
    }
    return x + 1
}

func main() {

    var rcvmsg string
    var h int = 0

    for {
		// 0. On reçoit un message
        fmt.Scanln(&rcvmsg)

        // 1. On regarde si le message contient une horloge, et on recale l'horloge locale en conséquence
        s_hrcv := findval(rcvmsg, "hlg")
        if s_hrcv != "" {
            hrcv, _ := strconv.Atoi(s_hrcv)
            h = recaler(h, hrcv)
        } else {
            h = h + 1
        }

        // 2. On regarde si le message contient un message de contrôle ou d'application
		// Hypothèse : les messages de l'application ne contiennent pas de champ "msg"
		// - Si c'est un message de contrôle (avec un champ "msg"), on transmet le message brut à notre application locale
		// - Si c'est un message d'application (sans champ "msg"), on encapsule le message avec l'horloge locale pour envoyer à un centre de contrôle distant
        sndmsg := findval(rcvmsg, "msg")
        if sndmsg == "" {
            fmt.Println(msg_format("msg", rcvmsg) + msg_format("hlg", strconv.Itoa(h)))
        } else {
            fmt.Println(sndmsg)
        }
    }
}