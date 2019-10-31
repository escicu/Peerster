package main

import (
	"net"
	"sync"
	"time"
	"crypto/sha256"
)

const(
	Def_hoplimit=49
)

type AckRumor struct{
  ID uint32
  Timer *time.Timer
}

type AckRumorSlice []*AckRumor

type Rumorlistindextuple struct{
  list int
  index int
}

type OriginState struct{
  rumorinlist map[uint32]Rumorlistindextuple
  next uint32
	stateLock sync.Mutex
}

type Nexthoptuple struct{
  maxid uint32
  addr *net.UDPAddr
}

const(
	Enum_rumorlist=iota
	Enum_routerumlist=iota
)

/*
lock order:
			peersLock
				then ackwaitLock
						then originrumorlock
							then	originnextLock or rumorlistLock
*/
type Gossiper struct{
	name string
	gaddr string
	gconnect *net.UDPConn
	uiconnect *net.UDPConn
	peerstring string
	peers []*net.UDPAddr
		peersLock sync.Mutex
	seq uint32
	antientroptimeout time.Duration
	acktimeout time.Duration
	rumorlist []*RumorMessage
		rumorlistLock sync.Mutex
	routerumlist []*RumorMessage
		routerumlistLock sync.Mutex
	originstat map[string]*OriginState
		originstatLock sync.RWMutex
	//ackwaiting[addresse][origin]
	ackwaiting map[string](map[string]AckRumorSlice)
		ackwaitLock sync.Mutex
	rttimer time.Duration
	nexthop map[string]*Nexthoptuple
		nexthopLock sync.Mutex
	//privlist[origin]
	privlist map[string][]*PrivateMessage
		privlistLock sync.Mutex
	listfile []*Fileintern
		listfileLock sync.Mutex
	//hashready[string(HashValue)]
	hashready map[string]string
		hashreadyLock sync.Mutex
	//hashrequested[string(HashValue)]
	hashrequested map[string]hashrequesttuple
		hashrequestedLock sync.Mutex
	reqtimeout time.Duration
}

type hashrequesttuple struct{
	Name string
  Ticker *time.Ticker
	file *fileindown
}

type Fileintern struct{
	name string
	size int64
	metafile []byte
	metahash [sha256.Size]byte
}

type fileindown struct{
	name string
	pendingreply int
	lock sync.Mutex
}
