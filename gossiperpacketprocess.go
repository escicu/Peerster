package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"sort"
)

func (g *Gossiper) processSimplePacket(pack *GossipPacket, indexNot int) {
	s := fmt.Sprintf("SIMPLE MESSAGE origin %s from %s contents %s\n", pack.Simple.Originalname, pack.Simple.RelayPeerAddr, pack.Simple.Contents)
	s += fmt.Sprintf("PEERS %s\n", g.peerstring)
	g.Printf(1, s)
	simplemess := pack.Simple
	simplemess.RelayPeerAddr = g.gaddr
	g.peersLock.Lock()
	BroadcastPacket(pack, g.gconnect, g.peers[:indexNot])
	BroadcastPacket(pack, g.gconnect, g.peers[indexNot+1:])
	g.peersLock.Unlock()
}

func (g *Gossiper) processRumorPacket(rum *RumorMessage, indexNot int, addrfrom *net.UDPAddr) {
	s := fmt.Sprintf("RUMOR origin %s from %s ID %d contents %s\n", rum.Origin, addrfrom.String(), rum.ID, rum.Text)
	s += fmt.Sprintf("PEERS %s\n", g.peerstring)
	g.Printf(1, s)

	if g.addRumorPeer(rum) {
		//StatusPacket ack
		go g.sendStatPacket(addrfrom)

		//MONGERING next peer
		g.sendRumorMonger(rum, indexNot, nil)
	}

	g.nexthopLock.Lock()
	tuple, exist := g.nexthop[rum.Origin]
	if !exist || tuple.maxid < rum.ID {
		g.nexthop[rum.Origin] = &Nexthoptuple{rum.ID, addrfrom}
		if rum.Text != "" {
			s = fmt.Sprintf("DSDV %s %s\n", rum.Origin, addrfrom.String())
			g.Printf(2, s)
		}
	}
	g.nexthopLock.Unlock()

}

func (g *Gossiper) processStatusPacket(status *StatusPacket, indexNot int, addrfrom *net.UDPAddr) {
	s := fmt.Sprintf("STATUS from %s", addrfrom.String())
	vectorclockreceive := make(map[string]uint32)
	acked := false
	var rumordforwardorigin string
	var rumorforwardid uint32
	g.ackwaitLock.Lock()
	maporigack := g.ackwaiting[addrfrom.String()]
	for _, ps := range status.Want {
		s += fmt.Sprintf(" peer %s nextID %d", ps.Identifier, ps.NextID)
		vectorclockreceive[ps.Identifier] = ps.NextID
		ackslice, exist := maporigack[ps.Identifier]
		if exist {
			for len(ackslice) > 0 && ackslice[0].ID < ps.NextID {
				ackslice[0].Timer.Stop()
				rumordforwardorigin = ps.Identifier
				rumorforwardid = ackslice[0].ID
				ackslice = ackslice[1:]
				acked = true
			}
		} else {
			g.addCheckOrigin(ps.Identifier)
		}
		maporigack[ps.Identifier] = ackslice
	}
	g.ackwaitLock.Unlock()
	s += fmt.Sprintf("\nPEERS %s\n", g.peerstring)
	g.Printf(1, s)

	continuation := true
	sendstat := false
	var rm *RumorMessage
	var tlcm *TLCMessage
	sendrm := false
	sendtlc := false

	g.originstatLock.RLock()
	for k, v := range vectorclockreceive {
		stat := g.originstat[k]
		if v < stat.next {
			lt := stat.gossipedinlist[v]
			switch lt.list {
			case Enum_rumorlist:
				g.rumorlistLock.Lock() //because locking
				rm = g.rumorlist[lt.index]
				g.rumorlistLock.Unlock()
				sendrm = true
			case Enum_routerumlist:
				g.routerumlistLock.Lock() //because locking
				rm = g.routerumlist[lt.index]
				g.routerumlistLock.Unlock()
				sendrm = true
			case Enum_tlclist:
				g.tlclistLock.Lock() //because locking
				tlcm = g.tlclist[lt.index]
				g.tlclistLock.Unlock()
				sendtlc = true

			}
			continuation = false
			break
		}
	}
	if continuation {
		for k, v := range vectorclockreceive {
			if v > g.originstat[k].next {
				sendstat = true
				continuation = false
				break
			}
		}
	}
	g.originstatLock.RUnlock()
	switch { //because of lock (grrrrrrrrrrrrrrrrr)
	case sendrm:
		g.sendRumorMonger(rm, -1, addrfrom)
	case sendtlc:
		g.sendTLCMonger(tlcm, -1, addrfrom)
	case sendstat:
		g.sendStatPacket(addrfrom)
	}

	if continuation {
		s = fmt.Sprintf("IN SYNC WITH %s\n", addrfrom.String())
		g.Printf(2, s)

		if acked {
			flip := rand.Intn(2)
			if flip == 1 {
				indcoin, addrcoin := g.randomPeer(indexNot)
				s = fmt.Sprintf("FLIPPED COIN sending rumor to %s\n", addrcoin.String())
				g.Printf(2, s)
				g.originstatLock.RLock() //because locking
				rumortuple := g.originstat[rumordforwardorigin].gossipedinlist[rumorforwardid]
				g.originstatLock.RUnlock()
				var rms *RumorMessage
				var tlcms *TLCMessage
				switch rumortuple.list {
				case Enum_rumorlist:
					g.rumorlistLock.Lock() //because locking
					rms = g.rumorlist[rumortuple.index]
					g.rumorlistLock.Unlock()
				case Enum_routerumlist:
					g.routerumlistLock.Lock() //because locking
					rms = g.routerumlist[rumortuple.index]
					g.routerumlistLock.Unlock()
				case Enum_tlclist:
					g.tlclistLock.Lock() //because locking
					tlcms = g.tlclist[rumortuple.index]
					g.tlclistLock.Unlock()
				}
				if rms != nil {
					g.sendRumorMonger(rms, indcoin, addrcoin)
				}
				if tlcms != nil {
					g.sendTLCMonger(tlcms, indcoin, addrcoin)
				}
			}
		}
	}
}

func (g *Gossiper) processPrivatePacket(pack *GossipPacket) {
	priv := pack.Private
	if pack.Private.Destination == g.name {
		s := fmt.Sprintf("PRIVATE origin %s hop-limit %d contents %s\n", priv.Origin, priv.HopLimit, priv.Text)
		g.Printf(1, s)
		g.privlistLock.Lock()
		l, exist := g.privlist[priv.Origin]
		if exist {
			g.privlist[priv.Origin] = append(l, priv)
		} else {
			g.privlist[priv.Origin] = []*PrivateMessage{priv}
		}
		g.privlistLock.Unlock()

	} else {
		if priv.HopLimit > 0 {
			priv.HopLimit--
			g.sendDirect(pack, priv.Destination)
		}
	}
}

func (g *Gossiper) processDataRequestPacket(pack *GossipPacket) {

	d := pack.DataRequest
	if d.Destination == g.name {
		g.hashreadyLock.RLock()
		filename, exist := g.hashready[string(d.HashValue)]
		g.hashreadyLock.RUnlock()
		if exist {
			val, err := ioutil.ReadFile(filename)
			if err == nil {
				h := sha256.Sum256(val)
				g.sendDirect(&GossipPacket{DataReply: &DataReply{Origin: g.name, Destination: d.Origin, HopLimit: g.hoplimit, HashValue: h[:], Data: val}}, d.Origin)
			}
		} else {
			g.sendDirect(&GossipPacket{DataReply: &DataReply{Origin: g.name, Destination: d.Origin, HopLimit: g.hoplimit, HashValue: d.HashValue}}, d.Origin)
		}
	} else {
		if d.HopLimit > 0 {
			d.HopLimit--
			g.sendDirect(pack, d.Destination)
		}
	}
}

func (g *Gossiper) processDataReplyPacket(pack *GossipPacket) {

	d := pack.DataReply
	if d.Destination == g.name {
		h := d.HashValue
		g.hashrequestedLock.Lock()
		tuple, exist := g.hashrequested[string(h)]
		if exist {
			val := d.Data
			hverif := sha256.Sum256(val)
			if bytes.Equal(h, hverif[:]) {
				err := ioutil.WriteFile(tuple.Name, val, 0644)
				if err == nil {
					if tuple.externChunkNum == 0 {
						g.Filemetaprocess(tuple.file, val, h, d.Origin, tuple.searched)
					}
					tuple.Ticker.Stop()
					go g.FileChunkUpdateAndReconstruct(tuple.file, tuple.externChunkNum)
					delete(g.hashrequested, string(h))
				}
			} else if len(val) < 1 {
				tuple.Ticker.Stop()
				delete(g.hashrequested, string(h))
			}
		}
		g.hashrequestedLock.Unlock()
	} else {
		if d.HopLimit > 0 {
			d.HopLimit--
			g.sendDirect(pack, d.Destination)
		}
	}

}

func (g *Gossiper) processSearchRequestPacket(r *SearchRequest, indexNot int) {
	fmt.Printf("processSearchRequestPacket %v , i %v\n", r, indexNot)
	//duplicate
	if g.searchRequestDuplicate(r) {
		fmt.Printf("dup\n")
		return
	}

	sr := g.Search(r.Keywords)

	if len(sr) > 0 {
		go g.sendDirect(&GossipPacket{SearchReply: &SearchReply{Origin: g.name, Destination: r.Origin, HopLimit: g.hoplimit, Results: sr}}, r.Origin)
	}

	r.Budget--
	if r.Budget > 0 {
		go g.sendRepartitSearchRequest(r, indexNot)
	}

}

func (g *Gossiper) processSearchReplyPacket(pack *GossipPacket) {
	s := pack.SearchReply
	if s.Destination == g.name {

		if g.pendingsearch == nil {
			return
		}

		g.pendingsearch.lock.Lock()

		for _, v := range s.Results {
			stru, exist := g.pendingsearch.returnedfiles[string(v.MetafileHash)]
			if exist {

				for _, cv := range v.ChunkMap {
					i := sort.Search(len(stru.chunklist), func(i int) bool { return stru.chunklist[i] >= cv })
					if !(i < len(stru.chunklist)) || stru.chunklist[i] != cv {
						stru.chunklist = append(stru.chunklist, cv)
						sort.Slice(stru.chunklist, func(i, j int) bool { return stru.chunklist[i] < stru.chunklist[j] })
					}
				}

				stru.resultlist[s.Origin] = v

				//check match request
				if !(len(stru.chunklist) < int(stru.chunkcount)) {
					add := true
					for _, val := range g.pendingsearch.matchedfiles {
						var sl *SearchResult
						for _, x := range val.resultlist {
							sl = x
							break
						}
						if bytes.Equal(sl.MetafileHash, v.MetafileHash) {
							add = false
							break
						}
					}

					if add {
						g.pendingsearch.matchedfiles = append(g.pendingsearch.matchedfiles, stru)
					}
				}

			} else {

				rf := &returnedfilesstruct{selectedname: v.FileName, chunklist: make([]uint64, 0), chunkcount: v.ChunkCount, resultlist: make(map[string]*SearchResult, 0)}

				rf.chunklist = append(rf.chunklist, v.ChunkMap...)
				sort.Slice(rf.chunklist, func(i, j int) bool { return rf.chunklist[i] < rf.chunklist[j] })

				rf.resultlist[s.Origin] = v

				g.pendingsearch.returnedfiles[string(v.MetafileHash)] = rf

				//check match request
				if !(len(v.ChunkMap) < int(v.ChunkCount)) {
					add := true
					for _, val := range g.pendingsearch.matchedfiles {
						var sl *SearchResult
						for _, x := range val.resultlist {
							sl = x
							break
						}
						if bytes.Equal(sl.MetafileHash, v.MetafileHash) {
							add = false
							break
						}
					}

					if add {
						g.pendingsearch.matchedfiles = append(g.pendingsearch.matchedfiles, rf)
					}
				}

				g.pendingsearch.returnedfiles[string(v.MetafileHash)] = rf

			}

			s := fmt.Sprintf("FOUND match %s at %s metafile=%s chunks=%v\n", v.FileName, s.Origin, hex.EncodeToString(v.MetafileHash), v.ChunkMap)
			g.Printf(3, s)

			if !(len(g.pendingsearch.matchedfiles) < Def_max_result) {
				g.endPendingSearch()
				break
			}

		}

		g.pendingsearch.lock.Unlock()

	} else {
		if s.HopLimit > 0 {
			s.HopLimit--
			g.sendDirect(pack, s.Destination)
		}
	}

}

func (g *Gossiper) processTLCPacket(tlcm *TLCMessage, indexNot int, addrfrom *net.UDPAddr) {
	if tlcm.Confirmed < 0 {
		s := fmt.Sprintf("UNCONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %s\n",
			tlcm.Origin, tlcm.ID, tlcm.TxBlock.Transaction.Name, tlcm.TxBlock.Transaction.Size, hex.EncodeToString(tlcm.TxBlock.Transaction.MetafileHash))
		g.Printf(3, s)

		pmess := &TLCAck{g.name, tlcm.ID, "", tlcm.Origin, g.hoplimit}

		s = fmt.Sprintf("SENDING ACK origin %s ID %d\n", tlcm.Origin, tlcm.ID)
		g.Printf(3, s)

		go g.sendDirect(&GossipPacket{Ack: pmess}, tlcm.Origin)
	} else {
		s := fmt.Sprintf("CONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %s\n",
			tlcm.Origin, tlcm.Confirmed, tlcm.TxBlock.Transaction.Name, tlcm.TxBlock.Transaction.Size, hex.EncodeToString(tlcm.TxBlock.Transaction.MetafileHash))
		g.rawlog+=s
		g.Printf(3, s)
	}

	if g.addTLCPeer(tlcm) {
		//StatusPacket ack
		go g.sendStatPacket(addrfrom)

		g.sendTLCMonger(tlcm, indexNot, nil)
	}
}

func (g *Gossiper) processTLCAckPacket(pack *GossipPacket) {
	ta := pack.Ack
	if ta.Destination == g.name {
		g.tlcackwaitLock.Lock()
		var acks *AckTLC
		var aind int
		for i, a := range g.tlcackwaiting {
			if a.ID == ta.ID {
				acks = a
				aind = i
				break
			}
		}
		g.tlcackwaitLock.Unlock()

		if acks != nil {
			acks.lock.Lock()
			acks.acked[ta.Origin] = true
			sw := ""
			var aid uint32
			if len(acks.acked) > g.N/2 {
				sw += "WITNESSES "
				for k, _ := range acks.acked {
					sw += fmt.Sprintf("%s,", k)
				}
				aid = acks.ID
				acks.Ticker.Stop()
			}
			acks.lock.Unlock()
			if sw != "" {
				g.originstatLock.RLock()
				tlcm := g.tlclist[g.originstat[g.name].gossipedinlist[aid].index]
				g.originstatLock.RUnlock()

				mess := &TLCMessage{g.name, g.seq, int(tlcm.ID), tlcm.TxBlock, tlcm.VectorClock, tlcm.Fitness}

				g.addTLCClient(mess)
				g.seq++

				s := fmt.Sprintf("RE-BROADCAST ID %d ", aid)+sw+"\n"
				g.rawlog+=s
				g.Printf(3, s)
				go g.sendTLCMonger(mess, -1, nil)

				g.tlcackwaitLock.Lock()
				var newtlcackwait AckTLCSlice
				if aind > 0 {
					newtlcackwait = append(newtlcackwait, g.tlcackwaiting[:aind]...)
				}
				if aind < len(g.tlcackwaiting)-1 {
					newtlcackwait = append(newtlcackwait, g.tlcackwaiting[aind+1:]...)
				}
				g.tlcackwaiting = newtlcackwait
				g.tlcackwaitLock.Lock()
			}
		}

	} else {
		if ta.HopLimit > 0 {
			ta.HopLimit--
			g.sendDirect(pack, ta.Destination)
		}
	}
}
