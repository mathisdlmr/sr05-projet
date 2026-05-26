package net

import (
	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

type Net struct {
	myID    int
	// TODO
}

func New(myID int, io *transport.IO, log *logger.Logger) *Net {
	return &Net{
		myID: myID,
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

		// TODO : traiter le message
	}
}