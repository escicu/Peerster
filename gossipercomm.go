package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"
)

//lock ackwaitLock
func (g *Gossiper) sendRumorMonger(rumor *RumorMessage, indexNot int, addrudp *net.UDPAddr) {
	pei := indexNot

	addr := addrudp

	if addr == nil {
		pei, addr = g.randomPeer(pei)
	}

	if addr != nil {
		s := fmt.Sprintf("MONGERING with %s\n", addr.String())
		g.Printf(1, s)
		SendPacket(&GossipPacket{Rumor: rumor}, g.gconnect, addr)
		timer := time.AfterFunc(g.acktimeout, func() {
			g.sendRumorMonger(rumor, pei, nil)
		})
		g.ackwaitLock.Lock()
		maporigack := g.ackwaiting[addr.String()]
		maporigack[rumor.Origin] = append(maporigack[rumor.Origin], &AckGossiping{rumor.ID, timer})
		sort.Sort(maporigack[rumor.Origin])
		g.ackwaitLock.Unlock()
	}
}

//lock originnextLock
func (g *Gossiper) sendStatPacket(addrudp *net.UDPAddr) {

	addr := addrudp

	if addrudp == nil {
		_, addr = g.randomPeer(-1)
	}

	if addr != nil {
		ackpeerstat := []PeerStatus{}
		g.originstatLock.RLock()
		for k, v := range g.originstat {
			ps := PeerStatus{k, v.next}
			ackpeerstat = append(ackpeerstat, ps)
		}
		g.originstatLock.RUnlock()
		statpack := StatusPacket{Want: ackpeerstat}
		SendPacket(&GossipPacket{Status: &statpack}, g.gconnect, addr)
	}
}

//lock nexthopLock
func (g *Gossiper) sendDirect(pack *GossipPacket, dest string) {

	var addr *net.UDPAddr

	g.nexthopLock.Lock()
	t, ex := g.nexthop[dest]
	g.nexthopLock.Unlock()

	if ex {
		addr = t.addr
	}

	if addr != nil {
		SendPacket(pack, g.gconnect, addr)
	}
}

//lock hashrequestedLock
func (g *Gossiper) sendAllchunkRequest(metacontent []byte, chunkn int, dest string, rfi *lockedSearchResult) {

	g.hashrequestedLock.Lock()
	for i := 0; i < chunkn; i++ {
		fname := "_SharedFiles/" + rfi.searchresult.FileName + ".chunk" + strconv.Itoa(i)
		r := &DataRequest{Origin: g.name, Destination: dest, HopLimit: g.hoplimit, HashValue: metacontent[i*sha256.Size : (i+1)*sha256.Size]}
		hrt := hashrequesttuple{fname, time.NewTicker(g.reqtimeout),
			"DOWNLOADING " + rfi.searchresult.FileName + " chunk " + strconv.Itoa(i+1) + " from " + dest, uint64(i + 1), rfi, nil}
		go func() {
			for _ = range hrt.Ticker.C {
				g.Printf(3, hrt.printstr+"\n")
				g.sendDirect(&GossipPacket{DataRequest: r}, dest)
			}
		}()
		g.Printf(3, hrt.printstr+"\n")
		go g.sendDirect(&GossipPacket{DataRequest: r}, dest)
		g.hashrequested[string(r.HashValue)] = hrt
	}
	g.hashrequestedLock.Unlock()

}

//lock peerslock
func (g *Gossiper) sendRepartitSearchRequest(sr *SearchRequest, indexNot int) {
	peercopy := make([]*net.UDPAddr, 0)

	g.peersLock.Lock()
	fmt.Printf("peers %v\n", g.peers)
	if indexNot > -1 && indexNot < len(g.peers) {
		peercopy = append(peercopy, g.peers[:indexNot]...)
		if indexNot < (len(g.peers) - 1) {
			peercopy = append(peercopy, g.peers[indexNot+1:]...)
		}
	} else {
		peercopy = append(peercopy, g.peers...)
	}
	fmt.Printf("peercopy %v\n", peercopy)
	g.peersLock.Unlock()

	bu := sr.Budget
	l := uint64(len(peercopy))
	if l > uint64(0) {
		if l > sr.Budget {
			sr.Budget = 1
			pack := &GossipPacket{SearchRequest: sr}

			BroadcastPacket(pack, g.gconnect, peercopy[:sr.Budget])

		} else {
			b := sr.Budget / l
			rest := sr.Budget % l

			sr.Budget = b
			pack := &GossipPacket{SearchRequest: sr}
			BroadcastPacket(pack, g.gconnect, peercopy[rest:])

			sr.Budget = b + 1
			pack = &GossipPacket{SearchRequest: sr}
			BroadcastPacket(pack, g.gconnect, peercopy[:rest])

		}

	}
	sr.Budget = bu

}

//lock hashrequestedLock
func (g *Gossiper) sendAllchunkRequestFromSearch(file *returnedfilesstruct, metacontent []byte, rfi *lockedSearchResult) {

	askedchunk := make([]uint64, 0)

	g.hashrequestedLock.Lock()
	defer g.hashrequestedLock.Unlock()

	for o, r := range file.resultlist {
		for _, c := range r.ChunkMap {
			ind := sort.Search(len(askedchunk), func(i int) bool { return askedchunk[i] >= c })
			if !(ind < len(askedchunk)) || askedchunk[ind] != c {

				i := int(c) - 1
				fname := "_SharedFiles/" + rfi.searchresult.FileName + ".chunk" + strconv.Itoa(i)
				dr := &DataRequest{Origin: g.name, Destination: o, HopLimit: g.hoplimit, HashValue: metacontent[i*sha256.Size : (i+1)*sha256.Size]}
				hrt := hashrequesttuple{fname, time.NewTicker(g.reqtimeout),
					"DOWNLOADING " + rfi.searchresult.FileName + " chunk " + strconv.FormatUint(c, 10) + " from " + o, c, rfi, file}
				go func() {
					for _ = range hrt.Ticker.C {
						g.Printf(3, hrt.printstr+"\n")
						g.sendDirect(&GossipPacket{DataRequest: dr}, o)
					}
				}()
				g.Printf(3, hrt.printstr+"\n")
				go g.sendDirect(&GossipPacket{DataRequest: dr}, o)
				g.hashrequested[string(dr.HashValue)] = hrt

				askedchunk = append(askedchunk, c)

				if !(len(askedchunk) < int(file.chunkcount)) {
					return
				}

				sort.Slice(askedchunk, func(i, j int) bool { return askedchunk[i] < askedchunk[j] })
			}
		}
	}

}

//lock ackwaitLock
func (g *Gossiper) sendTLCMonger(tlc *TLCMessage, indexNot int, addrudp *net.UDPAddr) {
	pei := indexNot

	addr := addrudp

	if addr == nil {
		pei, addr = g.randomPeer(pei)
	}

	if addr != nil {
		SendPacket(&GossipPacket{TLCMessage: tlc}, g.gconnect, addr)
		timer := time.AfterFunc(g.acktimeout, func() {
			g.sendTLCMonger(tlc, pei, nil)
		})
		g.ackwaitLock.Lock()
		maporigack := g.ackwaiting[addr.String()]
		maporigack[tlc.Origin] = append(maporigack[tlc.Origin], &AckGossiping{tlc.ID, timer})
		sort.Sort(maporigack[tlc.Origin])
		g.ackwaitLock.Unlock()
	}
}

//lock tlcackwaitLock
func (g *Gossiper) sendTLCunconfirmed(tlc *TLCMessage) {

	s := fmt.Sprintf("UNCONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %s\n",
		tlc.Origin, tlc.ID, tlc.TxBlock.Transaction.Name, tlc.TxBlock.Transaction.Size, hex.EncodeToString(tlc.TxBlock.Transaction.MetafileHash))
	g.Printf(3, s)
	g.sendTLCMonger(tlc, -1, nil)
	timer := time.NewTicker(g.stubborntimeout)
	go func() {
		for _ = range timer.C {
			g.sendTLCMonger(tlc, -1, nil)
		}
	}()
	g.tlcackwaitLock.Lock()
	g.tlcackwaiting = append(g.tlcackwaiting, &AckTLC{ID: tlc.ID, Ticker: timer, acked: make(map[string]bool, 0)})
	g.tlcackwaiting[len(g.tlcackwaiting)-1].acked[g.name] = true
	sort.Sort(g.tlcackwaiting)
	g.tlcackwaitLock.Unlock()
}
