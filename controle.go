package main

import (
    "flag"
    "fmt"
    "strconv"
)

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

    // On récupère le nom du processus à partir des arguments en CLI
    p_nom = flag.String("n", "controle", "nom")
    flag.Parse()

    display_d("main", "démarrage du programme")

    for {
		// 0. On reçoit un message
        fmt.Scanln(&rcvmsg)
        display_d("main", "reçu : "+rcvmsg)

        // 1. On regarde si le message contient une horloge, et on recale l'horloge locale en conséquence
        s_hrcv := findval(rcvmsg, "hlg")
        if s_hrcv != "" {
            hrcv, err := strconv.Atoi(s_hrcv)
            if err != nil {
                display_e("main", "conversion horloge impossible")
            } else {
                display_d("clock", "recalage avec "+s_hrcv)
                h = recaler(h, hrcv)
            }
        } else {
            display_d("clock", "incrément local")
            h = h + 1
        }

        // 2. On regarde si le message contient un message de contrôle ou d'application
		// Hypothèse : les messages de l'application ne contiennent pas de champ "msg"
		// - Si c'est un message de contrôle (avec un champ "msg"), on transmet le message brut à notre application locale
		// - Si c'est un message d'application (sans champ "msg"), on encapsule le message avec l'horloge locale pour envoyer à un centre de contrôle distant
        sndmsg := findval(rcvmsg, "msg")
        if sndmsg == "" {
            display_d("main", "message application -> encapsulation")
            msg := msg_format("msg", rcvmsg) + msg_format("hlg", strconv.Itoa(h))
            msg_send(msg)
        } else {
            display_d("main", "message contrôle -> transmission")
            msg_send(sndmsg)
        }
    }
}