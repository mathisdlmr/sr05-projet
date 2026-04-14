package main

import (
    "fmt"
    "log"
    "os"
)

// Couleurs ANSI
// https://en.wikipedia.org/wiki/ANSI_escape_code
// https://stackoverflow.com/questions/4842424/list-of-ansi-color-escape-sequences
var rouge = "\033[1;31m"
var orange = "\033[1;33m"
var jaune = "\033[1;41m"
var cyan = "\033[1;36m"
var raz = "\033[0;00m"

// Variables globales
var stderr = log.New(os.Stderr, "", 0)
var pid = os.Getpid()
var p_nom *string

// Log level debug
func display_d(where string, what string) {
    stderr.Printf(" DEBUG - [%.6s %d] %-8.8s : %s\n", *p_nom, pid, where, what)
}

// Log level warning
func display_w(where string, what string) {
    stderr.Printf("%s WARNING - [%.6s %d] %-8.8s : %s\n%s", orange, *p_nom, pid, where, what, raz)
}

// Log level error
func display_e(where string, what string) {
    stderr.Printf("%s ERROR - [%.6s %d] %-8.8s : %s\n%s", rouge, *p_nom, pid, where, what, raz)
}

// Envoi message (stdout)
func msg_send(msg string) {
    display_d("msg_send", "émission : "+msg)
    fmt.Println(msg)
}