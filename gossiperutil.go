package main

import (
	"fmt"
	"net"
	"math/rand"
)

func (g *Gossiper) randomPeer(indexNot int)(int , *net.UDPAddr){
	pei :=indexNot
	var addr *net.UDPAddr
	addr=nil
	limit:=0

		g.peersLock.Lock()
		l:=len(g.peers)
		if(l>0){
			for pei==indexNot && limit<5{
				pei = rand.Intn(l)
				limit++
			}
			addr=g.peers[pei]
		}
		g.peersLock.Unlock()

	return pei, addr
}

//lock rumorlistLock, originrumorLock , originnextLock
func (g *Gossiper) addRumorPeer(rumor *RumorMessage) (bool){

	g.originrumorLock.Lock()

	rummapid,vu:=g.originrumor[rumor.Origin]

	if(!vu){
		g.originrumor[rumor.Origin]=make(map[uint32]int)
		g.originnextLock.Lock()
		g.originnext[rumor.Origin]=1
		g.originnextLock.Unlock()
	}
	rummapid,vu=g.originrumor[rumor.Origin]
	if(!vu){
		fmt.Printf("Error saving data\n")
	}
	_,got:=rummapid[rumor.ID]
	if(!got){
		g.rumorlistLock.Lock()
		rummapid[rumor.ID]=len(g.rumorlist)
		g.rumorlist=append(g.rumorlist,rumor)
		g.rumorlistLock.Unlock()
	} else{
		return false
	}
	_,got=rummapid[rumor.ID]
	if(!got){
		fmt.Printf("Error saving data\n")
	}
	g.originnextLock.Lock()
	if(rumor.ID== g.originnext[rumor.Origin]){
		nextid:=rumor.ID+1
		_,got=rummapid[nextid]
		for got{
			nextid++
			_,got=rummapid[nextid]
		}
		g.originnext[rumor.Origin]=nextid
	}
	g.originnextLock.Unlock()

	g.originrumorLock.Unlock()

	return true

}


//lock rumorlistLock, originrumorLock , originnextLock
func (g *Gossiper) addRumorClient(rumor *RumorMessage) (bool){

	g.originrumorLock.Lock()
	rummapid,_:=g.originrumor[rumor.Origin]

	g.rumorlistLock.Lock()
	rummapid[rumor.ID]=len(g.rumorlist)
	g.rumorlist=append(g.rumorlist,rumor)
	g.rumorlistLock.Unlock()

	_,got:=rummapid[rumor.ID]
	if(!got){
		fmt.Printf("Error saving data\n")
	}

	g.originnextLock.Lock()
	g.originnext[g.name]=g.seq+1
	g.originnextLock.Unlock()

	g.originrumorLock.Unlock()

	return true

}
