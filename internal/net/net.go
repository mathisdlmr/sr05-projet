package net

import (
	"fmt"
	"io"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

type Net struct {
	myID                   int
	io                     *transport.IO
	log                    *logger.Logger
	nextSiteId             int
	myTeePid               int
	electionStartPending   bool
	electionStartPendingId int // Id du site a proposer a l'election
}

func New(myID int, io *transport.IO, log *logger.Logger, nextSiteId int) *Net {
	return &Net{
		myID:                 myID,
		io:                   io,
		log:                  log,
		nextSiteId:           nextSiteId,
		electionStartPending: false,
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

		c.handleMessage(*msg)
		// TODO : traiter le message
	}
}

func (c *Net) handleMessage(msg transport.Message) {
	switch msg.Type {
	case transport.TypeApplication:
		return
	case transport.TypeControl:
		if msg.Sender == c.myID { // forward on the ring
			msg.Type = transport.TypeNet
			c.io.Send(msg.String())
		}
	case transport.TypeNet:
		switch msg.Action {
		case transport.ActionConnectToYourNext:
		case ActionAddMeToNet :       
		case ActionConnectToYourNext :
		                       // Message classique
		case ActionElection         : // 		Modify to Type Control and send to control
		case ActionElectionTerminee : // 		Modify to TypeNet and send to Net ?
		// Message NetOnly
		// 		Action sur le message

	}
}

func (c *Net) create_tee() {
	// Cree tee vers
	// - lui même /tmp/in_ctl$i
	// - le suivant /tmp/in_net$next
	// Récupère son PID de tee
}

func (c *Net) kill_tee() {
	// Tue le tee a c.myTeePid

}

func (c *Net) addSite(siteID int) {
	// Kill the tee
	// save old next
	// Set new next
	// Create the tee with new targets
	// Send the connection message to the new site (with info of the old next)
}
