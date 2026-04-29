// Package control - file d'attente répartie (algorithme d'exclusion mutuelle).
// queue.go - implémentation de l'Algorithme 28 du poly "5-POLY-file-attente-2018.pdf",
// adapté à la topologie en anneau du projet.
//
// Principe : chaque site maintient un tableau Tab[N] de couples (type, date) où
// type ∈ {requete, liberation, accuse} et date est une estampille de Lamport.
// Un site n'entre en SC que lorsque sa propre demande est la plus ancienne dans Tab,
// le départage des égalités se faisant par ID de site.
//
// Format des messages échangés via broadcast en anneau :
//   /=type=req/=h=<lamport>                          (demande de SC)
//   /=type=rel/=h=<lamport>                          (libération de SC)
//   /=type=ack/=to=<destinataire>/=h=<lamport>       (accusé ciblé)
// Le préfixe BROADCAST: et le champ from=<émetteur> sont ajoutés par broadcast().

package control

import (
	"strconv"

	"github.com/sr05-projet/pkg/transport"
)

type entryKind string

const (
	entryRequest entryKind = "requete"
	entryRelease entryKind = "liberation"
	entryAck     entryKind = "accuse"
)

type tabEntry struct {
	Kind entryKind
	Date int
}

// pendingSC représente une action de jeu en attente d'entrée en section critique.
// Le fn est exécuté une fois la SC obtenue ; il doit re-vérifier sa précondition
// car la phase peut avoir changé pendant l'attente (un autre site est passé avant).
type pendingSC struct {
	name string
	fn   func()
}

// siteIndex convertit un identifiant "Jk" en index 0-based dans tab.
// Retourne -1 si l'ID est mal formé.
func siteIndex(id string) int {
	if len(id) < 2 || id[0] != 'J' {
		return -1
	}
	n, err := strconv.Atoi(id[1:])
	if err != nil || n < 1 {
		return -1
	}
	return n - 1
}

// requestSC est appelée par la logique de jeu pour demander la SC.
// Si une demande est déjà en cours, l'appel est ignoré (les transitions
// de phase sont mutuellement exclusives au sein d'un site).
func (c *Control) requestSC(action pendingSC) {
	if c.pending != nil {
		c.log.Debug("requestSC", "ignorée (demande déjà en cours: "+c.pending.name+")")
		return
	}
	c.pending = &action
	c.lamport++
	c.tab[c.myIndex] = tabEntry{Kind: entryRequest, Date: c.lamport}
	c.log.Debug("requestSC", action.name+" h="+strconv.Itoa(c.lamport))
	c.broadcast(transport.Build("type", "req", "h", strconv.Itoa(c.lamport)))
	// Cas N=1 : pas d'autre site, on entre immédiatement.
	c.tryEnterSC()
}

// tryEnterSC évalue la condition d'entrée du poly (ligne 19/26/35 de l'Algo 28).
// Le site entre en SC si sa propre case Tab[myIndex] contient une requête dont
// le couple (date, index) est strictement plus ancien que toutes les autres cases.
func (c *Control) tryEnterSC() {
	if c.pending == nil || c.tab[c.myIndex].Kind != entryRequest {
		return
	}
	myDate := c.tab[c.myIndex].Date
	for k, e := range c.tab {
		if k == c.myIndex {
			continue
		}
		// Comparaison lexicographique sur (date, index).
		// Si une autre case est strictement plus ancienne, on attend.
		if e.Date < myDate || (e.Date == myDate && k < c.myIndex) {
			return
		}
	}
	action := c.pending
	c.pending = nil
	c.log.Success("enterSC", action.name)
	action.fn()
	c.releaseSC()
}

// releaseSC est appelée à la fin de l'action en SC.
func (c *Control) releaseSC() {
	c.lamport++
	c.tab[c.myIndex] = tabEntry{Kind: entryRelease, Date: c.lamport}
	c.log.Debug("releaseSC", "h="+strconv.Itoa(c.lamport))
	c.broadcast(transport.Build("type", "rel", "h", strconv.Itoa(c.lamport)))
}

// handleReq traite une demande de SC reçue d'un autre site.
func (c *Control) handleReq(from string, h int) {
	idx := siteIndex(from)
	if idx < 0 || idx >= len(c.tab) {
		c.log.Warn("handleReq", "from invalide: "+from)
		return
	}
	c.lamport = maxInt(c.lamport, h) + 1
	c.tab[idx] = tabEntry{Kind: entryRequest, Date: h}
	// On accuse réception, ciblé sur l'émetteur.
	c.broadcast(transport.Build("type", "ack", "to", from, "h", strconv.Itoa(c.lamport)))
	c.tryEnterSC()
}

// handleRel traite une libération de SC reçue d'un autre site.
func (c *Control) handleRel(from string, h int) {
	idx := siteIndex(from)
	if idx < 0 || idx >= len(c.tab) {
		c.log.Warn("handleRel", "from invalide: "+from)
		return
	}
	c.lamport = maxInt(c.lamport, h) + 1
	c.tab[idx] = tabEntry{Kind: entryRelease, Date: h}
	c.tryEnterSC()
}

// handleAck traite un accusé de réception ciblé sur ce site.
// On n'écrase pas une requête plus récente par un accusé (cf. ligne 32 du poly).
func (c *Control) handleAck(from string, h int) {
	idx := siteIndex(from)
	if idx < 0 || idx >= len(c.tab) {
		c.log.Warn("handleAck", "from invalide: "+from)
		return
	}
	c.lamport = maxInt(c.lamport, h) + 1
	if c.tab[idx].Kind != entryRequest {
		c.tab[idx] = tabEntry{Kind: entryAck, Date: h}
	}
	c.tryEnterSC()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
