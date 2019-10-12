package main

import (
	"fmt"
	"net"
	"time"
	"sort"
)

func (g *Gossiper) sendRumorMonger(rumor *RumorMessage,indexNot int,addrudp *net.UDPAddr){
	pei :=indexNot

	addr:=addrudp

	if(addr==nil){
		pei,addr=g.randomPeer(pei)
	}

	if(addr!=nil){
		fmt.Printf("MONGERING with %s\n",addr.String())
		SendPacket(&GossipPacket{Rumor:rumor},g.gconnect,addr)
		timer:=time.AfterFunc(g.acktimeout, func() {
	        g.sendRumorMonger(rumor,pei,nil)
	    })
		g.ackwaitLock.Lock()
		maporigack:=g.ackwaiting[addr.String()]
		maporigack[rumor.Origin]=append(maporigack[rumor.Origin],&AckRumor{rumor.ID,timer})
		sort.Sort(maporigack[rumor.Origin])
		g.ackwaitLock.Unlock()
	}
}

func (g *Gossiper) sendStatPacket(addrudp *net.UDPAddr){

	addr:=addrudp

	if(addrudp==nil){
		_,addr=g.randomPeer(-1)
	}

	if(addr!=nil){
		ackpeerstat:=[]PeerStatus{}
		g.originnextLock.Lock()
		for k,v:=range g.originnext{
			ps:= PeerStatus{k,v}
			ackpeerstat=append(ackpeerstat,ps)
		}
		g.originnextLock.Unlock()
		statpack:=StatusPacket{Want:ackpeerstat}
		SendPacket(&GossipPacket{Status:&statpack},g.gconnect,addr)
	}
}
