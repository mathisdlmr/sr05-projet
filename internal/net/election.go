package net

import (
	"fmt"
	"strconv"

	"github.com/sr05-projet/pkg/transport"
)

func (c *Net) handleElectionMessage(msg transport.Message) {
	currentCandidate, _ := strconv.Atoi(msg.Data["candidat"])
	siteToAddInMessageId, _ := strconv.Atoi(msg.Data["idToAdd"])

	if c.electionGoingOn != -1 && siteToAddInMessageId != c.electionGoingOn {
		// conflict between elections
		if siteToAddInMessageId < c.electionGoingOn {
			c.log.Debug("handleElectionMessage", "Conflict entre election, on ecrase l'election en cours")
			c.electionGoingOn = siteToAddInMessageId
		} else {
			c.log.Debug("handleElectionMessage", fmt.Sprintf("Conflict entre election, stockage de l'election du site %d", siteToAddInMessageId))
			c.electionStartPending = append(c.electionStartPending, siteToAddInMessageId) // Store election for later
			return
		}
	}

	if currentCandidate == c.myID { // My site is elected
		if c.tryingToLeave { // I won but i want to leave : i restart the election process, without me
			msg.Data["candidat"] = strconv.Itoa(6000)
			c.sendMessage(msg)
		}
		c.log.Info("handleElectionMessage", "Election won, adding site after me")
		siteToInsert, _ := strconv.Atoi(msg.Data["idToAdd"])
		nextSiteIdString := strconv.Itoa(c.nextSiteId)
		c.insertSite(siteToInsert)
		c.sendMessage(transport.Message{
			Type:   transport.TypeNet,
			Action: transport.ActionConnectToYourNext,
			Data:   map[string]string{"nextSite": nextSiteIdString},
		})
		c.sendMessage(transport.Message{
			Type:   transport.TypeNet,
			Action: transport.ActionElectionTerminee,
		})
		c.electionGoingOn = -1
		c.checkElectionQueue()
		return
	}

	if currentCandidate > c.myID && !c.tryingToLeave { // Je deviens le nouveau candidat
		msg.Data["candidat"] = strconv.Itoa(c.myID)
		c.sendMessage(msg)
	}

	c.sendMessage(msg) // Envoi du message au prochain site
}

func (c *Net) handleElectionEnd() {

	if c.tryingToLeave {
		success := c.TryLeavingIfPossible()
		if success {
			return
		}
	}
	c.electionGoingOn = -1
	c.checkElectionQueue()
}

func (c *Net) checkElectionQueue() {
	if len(c.electionStartPending) != 0 {
		c.startElection(c.electionStartPending[0])
		c.electionStartPending = c.electionStartPending[1:] // retirer l'election pending de la queue
	}
}

func (c *Net) startElection(idToAdd int) {
	if c.tryingToLeave {
		c.log.Error("startElection", "starting election but i'm trying to leave")
	}

	if c.electionGoingOn != -1 { // Election already going on, storing in pending
		c.electionStartPending = append(c.electionStartPending, idToAdd)
		return
	}

	c.sendMessage(transport.Message{
		Type:   transport.TypeNet,
		Action: transport.ActionElection,
		Data:   map[string]string{"candidat": strconv.Itoa(c.myID), "idToAdd": strconv.Itoa(idToAdd)},
	})
}
