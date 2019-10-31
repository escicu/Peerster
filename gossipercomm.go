package main

import (
	"fmt"
	"net"
	"time"
	"sort"
	"crypto/sha256"
	"strconv"
)

//lock ackwaitLock
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

//lock originnextLock
func (g *Gossiper) sendStatPacket(addrudp *net.UDPAddr){

	addr:=addrudp

	if(addrudp==nil){
		_,addr=g.randomPeer(-1)
	}

	if(addr!=nil){
		ackpeerstat:=[]PeerStatus{}
		g.originstatLock.RLock()
		for k,v:=range g.originstat{
			ps:= PeerStatus{k,v.next}
			ackpeerstat=append(ackpeerstat,ps)
		}
		g.originstatLock.RUnlock()
		statpack:=StatusPacket{Want:ackpeerstat}
		SendPacket(&GossipPacket{Status:&statpack},g.gconnect,addr)
	}
}

//lock nexthopLock
func (g *Gossiper) sendDirect(pack *GossipPacket,dest string){

	g.nexthopLock.Lock()
	addr:=g.nexthop[dest].addr
	g.nexthopLock.Unlock()

	if(addr!=nil){
		SendPacket(pack,g.gconnect,addr)
	}
}

//lock hashrequestedLock
func (g *Gossiper) sendAllchunkRequest(basefilename string,metacontent []byte,chunkn int,dest string,rfi *fileindown){

			readytorequest:=make(map[string]*DataRequest)

			for i := 0; i < chunkn; i++ {
				fname:=basefilename+".chunk"+strconv.Itoa(i)
				r:=&DataRequest{Origin:g.name,Destination:dest,	HopLimit:Def_hoplimit,HashValue:metacontent[i*sha256.Size:(i+1)*sha256.Size]}
				readytorequest[fname]=r
			}

			g.hashrequestedLock.Lock()
			for k,v:=range readytorequest{
				hrt:=hashrequesttuple{k,time.NewTicker(g.reqtimeout),rfi}
				go func(){
					for _ = range hrt.Ticker.C {
						g.sendDirect(&GossipPacket{DataRequest:v}, v.Destination)
					}
				}()
				go g.sendDirect(&GossipPacket{DataRequest:v}, v.Destination)
				g.hashrequested[string(v.HashValue)]=hrt
			}
			g.hashrequestedLock.Unlock()

}
