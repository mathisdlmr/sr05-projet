package main

import (
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
		display_w("findval", "message trop court : "+msg)
		return ""
	}

	// 1. On récupère le séparateur de paires ainsi que chaque paire
	sep := msg[0:1];
	tab_allkeyvals := strings.Split(msg[1:], sep);   

	// 2. On parcourt les paires et récupère chaque séparateur
	for _, keyval := range tab_allkeyvals {
        if len(keyval) < 3 {
            display_w("findval", "paire invalide : "+keyval)
            continue
        }
		
		equ := keyval[0:1]

		// 3. On récupère la clé de la paire, et vérifie si c'est la bonne
		tabkeyval := strings.Split(keyval[1:], equ)
		if tabkeyval[0] == key {
			return tabkeyval[1] // 3.a. Oui, on retourne la valeur
		}
	}
	return "" // 3.b. Non, on retourne une chaîne vide
}