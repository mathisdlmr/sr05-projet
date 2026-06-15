package net

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sr05-projet/pkg/transport"
)

func (c *Net) TryLeavingIfPossible() bool {
	c.tryingToLeave = true
	if c.electionGoingOn != -1 {
		return false
	}

	c.aboutToLeave = true

	pendingStrings := toTabOfStrings(c.electionStartPending)
	c.sendMessage(transport.Message{
		Type:   transport.TypeNet,
		Action: transport.ActionDepart,
		Data: map[string]string{
			"site":             strconv.Itoa(c.myID),
			"nextSite":         strconv.Itoa(c.nextSiteId),
			"pendingElections": strings.Join(pendingStrings, ","),
		},
	})
	return true
}

func (c *Net) handleDepart(msg transport.Message) {
	if msg.Sender == c.myID {
		os.Exit(3)
		return
	}

	if c.aboutToLeave && msg.Data["nextSite"] == strconv.Itoa(c.myID) {
		// En cas de départs simultanés
		// Si je suis entrain de partir et que mon précédent veut partir,
		// Je signal qu'il ne faut pas se rattacher a moi mais a mon suivant.
		// Je modifie donc le message de celui qui veut partir
		// Les autres conflits sont négligeables, car la suppression se fait "d'un coup"
		// Le site est donc rattaché au suivant immediatement et saura donc si il faut "l'enjamber" aussi
		msg.Data["nextSite"] = strconv.Itoa(c.nextSiteId)
	}

	if msg.Data["pendingElections"] != "" && !c.tryingToLeave {
		pendingElectionsStrings := strings.Split(msg.Data["pendingElection"], ",")
		for _, electionString := range pendingElectionsStrings {
			election, _ := strconv.Atoi(electionString)
			c.electionStartPending = append(c.electionStartPending, election)
		}
		delete(msg.Data, "pendingElections")
	}

	c.io.Send(msg.String()) // Forward on ring

	msg.Type = transport.TypeControl // Send to control
	c.sendToControl(msg)

	if msg.Sender == c.nextSiteId { // "enjamber le site suivant qui souhaite être supprimé"
		newSiteToConnectTo, _ := strconv.Atoi(msg.Data["nextSite"])
		c.removeSite(newSiteToConnectTo)
		c.log.Info("handleDepart", fmt.Sprintf("Site %d supprimé", msg.Sender))
	}
}

func toTabOfStrings(tab []int) []string {
	tabOfStrings := []string{}
	for _, element := range tab {
		tabOfStrings = append(tabOfStrings, strconv.Itoa(element))
	}
	return tabOfStrings
}
