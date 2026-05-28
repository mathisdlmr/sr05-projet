package net

import (
	"fmt"
	"io"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

type Net struct {
	myID    int
	io      *transport.IO
	log     *logger.Logger
	// TODO
}

func New(myID int, io *transport.IO, log *logger.Logger) *Net {
	return &Net{
		myID: myID,
		io:   io,
		log:  log,
		// TODO
	}
}

func (c *Net) Run() {
	c.log.Info("Run", fmt.Sprintf("démarrage net"))
	for {
		line, err := c.io.ReadLine()
		if err == io.EOF {
			c.log.Info("Run", "stdin fermé, arrêt")
			return
		}
		if err != nil {
			c.log.Error("Run", "lecture stdin: "+err.Error())
			return
		}

		// parse message
		msg, err := transport.ParseMessage(line)
		if err != nil {
			c.log.Error("Run", "parse message: "+err.Error())
			continue
		}
		c.log.Info("Run", fmt.Sprintf("message reçu: %v", msg))

		// TODO : traiter le message
	}
}