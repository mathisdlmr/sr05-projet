
// Package transport : gère la communication inter-processus partagé par les 3 binaires : application, control et server
// io : gère la lecture sur stdin et écriture sur stdout
//
package transport

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type IO struct {
	scanner *bufio.Scanner
	out     io.Writer
}

func NewIO() *IO {
	return &IO{
		scanner: bufio.NewScanner(os.Stdin),
		out:     os.Stdout,
	}
}

// ReadLine - lit la prochaine ligne non vide sur stdin
// Retourne ("", io.EOF) quand stdin est fermé
func (t *IO) ReadLine() (string, error) {
	for t.scanner.Scan() {
		line := strings.TrimSpace(t.scanner.Text())
		if line != "" {
			return line, nil
		}
	}
	if err := t.scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}

// Send - écrit un message sur stdout
func (t *IO) Send(msg string) error {
	_, err := fmt.Fprintln(t.out, msg)
	return err
}

// MustSend - comme Send() mais panique en cas d'erreur
// A utiliser uniquement quand une erreur d'écriture est fatale
func (t *IO) MustSend(msg string) {
	if err := t.Send(msg); err != nil {
		panic("transport.IO: écriture stdout impossible: " + err.Error())
	}
}