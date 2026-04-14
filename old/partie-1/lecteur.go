package main

import (
    "fmt"
	"strings"
)

// findval(msg, key) : retourne la valeur associée à la clé key dans un message msg
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

func main() {
    var rcvmsg string

    for {
        fmt.Scanln(&rcvmsg)
		fmt.Println("nom reçu = ", findval(rcvmsg, "msg"));
		fmt.Println("horloge reçue = ", findval(rcvmsg, "hlg"));
    }
}