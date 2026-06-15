package net

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/sr05-projet/pkg/logger"
	"github.com/sr05-projet/pkg/transport"
)

// Pour éviter les conflits, lire dans départ
// Aussi entre départ et arrivées, on s'arrange pour qu'un site qui part ne perde pas d'arrivée en cours
// Car il transmet ses election pending
// Ne se présente pas comme candidat aux elections
// Part des que une election est finie (au cas ou il était candidat dans une election en cours)

type Net struct {
	myID                 int
	io                   *transport.IO
	log                  *logger.Logger
	nextSiteId           int
	myTeePid             int
	electionStartPending []int
	electionGoingOn      int // Id du site proposé a l'election en cours
	tryingToLeave        bool
	aboutToLeave         bool
	ttin                 int
	ttout                int
	static               bool
}

func New(myID int, io *transport.IO, log *logger.Logger, nextSiteId int, ttin int, ttout int, static bool) *Net {
	return &Net{
		myID:            myID,
		io:              io,
		log:             log,
		nextSiteId:      nextSiteId,
		electionGoingOn: -1,
		tryingToLeave:   false,
		aboutToLeave:    false,
		ttin:            ttin,
		ttout:           ttout,
		static:          static,
	}
}

func (c *Net) init() {
	c.log.Info("init", "Starting Net")
	c.kill_pid(c.ttout)
	c.create_tee()
	time.Sleep(2000000)
	if c.static {
		c.log.Debug("init", "Net started in static mode")
		return
	}
	c.sendMessage(transport.Message{
		Type:   transport.TypeNet,
		Action: transport.ActionAddMeToNet,
		Data:   map[string]string{"idToAdd": strconv.Itoa(c.myID)},
	})
	c.log.Debug("init", "Init message sent")
}

func (c *Net) Run() {
	c.init()
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
		c.log.Debug("Run", fmt.Sprintf("message reçu: %v", msg))
		c.handleMessage(*msg)
	}
}

func (c *Net) handleMessage(msg transport.Message) {
	switch msg.Type {
	case transport.TypeApplication:
		return // ça devrait pas arriver
	case transport.TypeControl:
		if msg.Sender == c.myID && !msg.ToControl { // Send control message on the ring
			switch msg.Action {
			case transport.ActionDepart:
				c.TryLeavingIfPossible()
			default:
				msg.Type = transport.TypeNet
				c.sendMessage(msg)
			}
		}
	case transport.TypeNet:
		switch msg.Action {
		case transport.ActionAddMeToNet:
			idToAdd, _ := strconv.Atoi(msg.Data["idToAdd"])
			c.startElection(idToAdd)
		case transport.ActionConnectToYourNext:
			nextSiteId, _ := strconv.Atoi(msg.Data["nextSite"])
			c.nextSiteId = nextSiteId
			c.kill_tee()
			c.create_tee()
			// Message classique
		case transport.ActionElection: // 		Modify to Type Control and send to control
			c.handleElectionMessage(msg)
		case transport.ActionElectionTerminee:
			if msg.Sender != c.myID {
				c.io.Send(msg.String()) // forward election end if not me
			}
			c.handleElectionEnd()
		case transport.ActionDepart: // Remonter au
			c.handleDepart(msg)

		default: // forward vers contrôle et vers net
			if msg.Sender != c.myID {
				c.io.Send(msg.String()) // forward on ring
			}
			c.sendToControl(msg)
		}
	}
}

func (c *Net) create_tee() {
	c.myTeePid = c.tee_redirect(
		filename(c.myID, "net", "out"),
		filename(c.myID, "ctl", "in"),
		filename(c.nextSiteId, "net", "in"),
	)
	time.Sleep(2000000)
	c.log.Debug("create_tee", fmt.Sprintf("Tee created with pid %d", c.myTeePid))
}

func (c *Net) kill_pid(pid int) error {
	cmd := exec.Command("kill", strconv.Itoa(pid))
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	c.log.Debug("kill pid", fmt.Sprintf("Killing pid %d", pid))
	return cmd.Start()
}

func (c *Net) kill_tee() {
	time.Sleep(2000000)
	c.log.Debug("kill_tee", "Killing my tee")
	c.kill_pid(c.myTeePid)
}

func (c *Net) insertSite(siteID int) {
	c.kill_tee()
	oldNext := c.nextSiteId
	c.nextSiteId = siteID
	c.create_tee()
	c.sendMessage(transport.Message{
		Type:   transport.TypeNet,
		Action: transport.ActionConnectToYourNext,
		Data:   map[string]string{"nextSite": strconv.Itoa(oldNext)},
	})

	c.log.Info("insertSite", fmt.Sprintf("Site %d ajouté", siteID))
}

func (c *Net) removeSite(newNextSiteId int) {
	c.kill_tee()
	c.nextSiteId = newNextSiteId
	c.create_tee()
}

func (c *Net) sendToControl(msg transport.Message) {
	msg.Type = transport.TypeControl
	msg.ToControl = true
	c.io.Send(msg.String())
}

func (c *Net) tee_redirect(file_in string, file_out1 string, file_out2 string) int {
	c.log.Debug("tee_redirect", fmt.Sprintf("files redirected are in %s ; out is redirected in %s and %s", file_in, file_out1, file_out2))

	cmd := exec.Command("tee", file_out1)
	cmd.Stderr = os.Stderr

	infile, _ := os.OpenFile(file_in, os.O_RDWR, os.ModeNamedPipe)
	cmd.Stdin = infile

	outfile2, _ := os.OpenFile(file_out2, os.O_RDWR, os.ModeNamedPipe)
	cmd.Stdout = outfile2

	cmd.SysProcAttr = &syscall.SysProcAttr{}
	if err := cmd.Start(); err != nil {
		c.log.Error("tee_redirect", err.Error())
		return 0
	}
	return cmd.Process.Pid
}
func filename(id int, layer string, direction string) string {
	return fmt.Sprintf("/tmp/%s_%s%d", direction, layer, id)
}

func (c *Net) sendMessage(msg transport.Message) {
	msg.Sender = c.myID
	c.io.Send(msg.String())
}
