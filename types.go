package main

import (
	"net"
	"sync"
	"time"
)

type Message struct{
  Text string
}

type SimpleMessage struct {
  Originalname string
  RelayPeerAddr string
  Contents string
}

type RumorMessage struct{
  Origin string
  ID uint32
  Text string
}

type PeerStatus struct {
  Identifier string
  NextID uint32
}

type StatusPacket struct{
  Want []PeerStatus
}

type GossipPacket struct{
  Simple *SimpleMessage
  Rumor *RumorMessage
  Status *StatusPacket
}

type AckRumor struct{
  ID uint32
  Timer *time.Timer
}

type AckRumorSlice []*AckRumor


/*
lock order:
			originrumorlock
				then	originnextLock or rumorlistLock

			peersLock
				then ackwaitLock

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
	originrumor map[string](map[uint32]int)
		originrumorLock sync.Mutex
	originnext map[string]uint32
		originnextLock sync.Mutex
	//ackwaiting[addresse][origin]
	ackwaiting map[string](map[string]AckRumorSlice)
		ackwaitLock sync.Mutex
}
