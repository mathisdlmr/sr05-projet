// Package logger : Fournit un système de log sur stderr partagé par les 3 binaires : application, control et server
package logger

import (
	"fmt"
	"log"
	"os"
)

// Couleurs ANSI
// https://en.wikipedia.org/wiki/ANSI_escape_code
// https://stackoverflow.com/questions/4842424/list-of-ansi-color-escape-sequences
const (
	rouge  = "\033[1;31m"
	orange = "\033[1;33m"
	cyan   = "\033[1;36m"
	vert   = "\033[1;32m"
	raz    = "\033[0;00m"
)

// On associe à notre Logger Go un nom du processus et son PID 
// (comme dans la partie 3 sur Moodle, question de lisibilité)
type Logger struct {
	name string
	pid  int
	l    *log.Logger
}

func New(name string) *Logger {
	return &Logger{
		name: name,
		pid:  os.Getpid(),
		l:    log.New(os.Stderr, "", 0),
	}
}

// Log level DEBUG (gris)
func (lg *Logger) Debug(where, what string) {
	lg.l.Printf(" DEBUG   - [%.8s %d] %-10.10s : %s\n", lg.name, lg.pid, where, what)
}

// Log level INFO (cyan)
func (lg *Logger) Info(where, what string) {
	lg.l.Printf("%s INFO    - [%.8s %d] %-10.10s : %s%s\n", cyan, lg.name, lg.pid, where, what, raz)
}

// Log level WARNING (orange)
func (lg *Logger) Warn(where, what string) {
	lg.l.Printf("%s WARNING - [%.8s %d] %-10.10s : %s%s\n", orange, lg.name, lg.pid, where, what, raz)
}

// Log level ERROR (rouge)
func (lg *Logger) Error(where, what string) {
	lg.l.Printf("%s ERROR   - [%.8s %d] %-10.10s : %s%s\n", rouge, lg.name, lg.pid, where, what, raz)
}

// Log level SUCCESS (vert)
func (lg *Logger) Success(where, what string) {
	lg.l.Printf("%s SUCCESS - [%.8s %d] %-10.10s : %s%s\n", vert, lg.name, lg.pid, where, what, raz)
}

// Fatal - log une erreur et quitte le processus
func (lg *Logger) Fatal(where, what string) {
	lg.l.Printf("%s FATAL   - [%.8s %d] %-10.10s : %s%s\n", rouge, lg.name, lg.pid, where, what, raz)
	fmt.Fprintln(os.Stderr, "Arrêt du processus.")
	os.Exit(1)
}