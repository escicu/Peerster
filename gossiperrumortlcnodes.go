package main

import (
	"fmt"
	"math/rand"
	"net"
)

func (g *Gossiper) randomPeer(indexNot int) (int, *net.UDPAddr) {
	pei := indexNot
	var addr *net.UDPAddr
	addr = nil
	limit := 0

	g.peersLock.Lock()
	l := len(g.peers)
	if l > 0 {
		for pei == indexNot && limit < 5 {
			pei = rand.Intn(l)
			limit++
		}
		addr = g.peers[pei]
	}
	g.peersLock.Unlock()

	return pei, addr
}

//lock originstatLock rumorlistLock
func (g *Gossiper) addRumorPeer(rumor *RumorMessage) bool {

	g.addCheckOrigin(rumor.Origin)

	g.originstatLock.RLock()
	stat, vu := g.originstat[rumor.Origin]
	if !vu {
		fmt.Printf("Error saving data\n")
	}
	g.originstatLock.RUnlock()

	stat.stateLock.Lock()
	_, got := stat.gossipedinlist[rumor.ID]
	if !got {
		if rumor.Text == "" {
			g.routerumlistLock.Lock()
			stat.gossipedinlist[rumor.ID] = Gossipedlistindextuple{Enum_routerumlist, len(g.routerumlist)}
			g.routerumlist = append(g.routerumlist, rumor)
			g.routerumlistLock.Unlock()
		} else {
			g.rumorlistLock.Lock()
			stat.gossipedinlist[rumor.ID] = Gossipedlistindextuple{Enum_rumorlist, len(g.rumorlist)}
			g.rumorlist = append(g.rumorlist, rumor)
			g.rumorlistLock.Unlock()
		}
	} else {
		stat.stateLock.Unlock()
		return false
	}

	_, got = stat.gossipedinlist[rumor.ID]
	if !got {
		fmt.Printf("Error saving data\n")
	}

	if rumor.ID == stat.next {
		nextid := rumor.ID + 1
		_, got = stat.gossipedinlist[nextid]
		for got {
			nextid++
			_, got = stat.gossipedinlist[nextid]
		}
		stat.next = nextid
	}

	stat.stateLock.Unlock()

	return true

}

//lock rumorlistLock, originrumorLock , originnextLock
func (g *Gossiper) addRumorClient(rumor *RumorMessage) {

	g.originstatLock.RLock()
	stat := g.originstat[rumor.Origin]
	g.originstatLock.RUnlock()

	stat.stateLock.Lock()

	if rumor.Text == "" {
		g.routerumlistLock.Lock()
		stat.gossipedinlist[rumor.ID] = Gossipedlistindextuple{Enum_routerumlist, len(g.routerumlist)}
		g.routerumlist = append(g.routerumlist, rumor)
		g.routerumlistLock.Unlock()
	} else {
		g.rumorlistLock.Lock()
		stat.gossipedinlist[rumor.ID] = Gossipedlistindextuple{Enum_rumorlist, len(g.rumorlist)}
		g.rumorlist = append(g.rumorlist, rumor)
		g.rumorlistLock.Unlock()
	}

	_, got := stat.gossipedinlist[rumor.ID]
	if !got {
		fmt.Printf("Error saving data\n")
	}

	stat.next = g.seq + 1

	stat.stateLock.Unlock()

}

//lock peersLock, ackwaitLock
func (g *Gossiper) addPeer(addr *net.UDPAddr) int {

	g.peersLock.Lock()
	ind := -1
	for i, u := range g.peers {
		if udpaddrequal(addr, u) {
			ind = i
			break
		}
	}

	if ind < 0 {
		ind = len(g.peers)
		g.peers = append(g.peers, addr)
		if ind == 0 {
			g.peerstring = g.peerstring + addr.String()
		} else {
			g.peerstring = g.peerstring + "," + addr.String()
		}
		g.ackwaitLock.Lock()
		g.ackwaiting[addr.String()] = make(map[string]AckGossipingSlice)
		g.ackwaitLock.Unlock()
	}

	g.peersLock.Unlock()

	return ind
}

//lock originrumorLock originnextLock
func (g *Gossiper) addCheckOrigin(origin string) {

	g.originstatLock.Lock()
	_, exist := g.originstat[origin]
	if !exist {
		var ostat OriginState
		ostat.gossipedinlist = make(map[uint32]Gossipedlistindextuple)
		ostat.next = 1
		g.originstat[origin] = &ostat
	}
	g.originstatLock.Unlock()

}

//lock originstatLock tlclistLock
func (g *Gossiper) addTLCPeer(tlcm *TLCMessage) bool {

	g.addCheckOrigin(tlcm.Origin)

	g.originstatLock.RLock()
	stat, vu := g.originstat[tlcm.Origin]
	if !vu {
		fmt.Printf("Error saving data\n")
	}
	g.originstatLock.RUnlock()

	stat.stateLock.Lock()
	_, got := stat.gossipedinlist[tlcm.ID]
	if !got {
		g.tlclistLock.Lock()
		stat.gossipedinlist[tlcm.ID] = Gossipedlistindextuple{Enum_tlclist, len(g.tlclist)}
		g.tlclist = append(g.tlclist, tlcm)
		g.tlclistLock.Unlock()
	} else {
		stat.stateLock.Unlock()
		return false
	}

	_, got = stat.gossipedinlist[tlcm.ID]
	if !got {
		fmt.Printf("Error saving data\n")
	}

	if tlcm.ID == stat.next {
		nextid := tlcm.ID + 1
		_, got = stat.gossipedinlist[nextid]
		for got {
			nextid++
			_, got = stat.gossipedinlist[nextid]
		}
		stat.next = nextid
	}

	stat.stateLock.Unlock()

	return true

}

//lock tlclistLock, originstatLock , stateLock
func (g *Gossiper) addTLCClient(tlc *TLCMessage) {

	g.originstatLock.RLock()
	stat := g.originstat[tlc.Origin]
	g.originstatLock.RUnlock()

	stat.stateLock.Lock()

	g.tlclistLock.Lock()
	stat.gossipedinlist[tlc.ID] = Gossipedlistindextuple{Enum_tlclist, len(g.tlclist)}
	g.tlclist = append(g.tlclist, tlc)
	g.tlclistLock.Unlock()

	_, got := stat.gossipedinlist[tlc.ID]
	if !got {
		fmt.Printf("Error saving data\n")
	}

	stat.next = g.seq + 1

	stat.stateLock.Unlock()

}
