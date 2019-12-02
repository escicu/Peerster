package main

import (
	"net"
	"sync"
	"time"
)

const (
	Def_hoplimit       = 10
	Def_max_budget     = 32
	Def_max_result     = 2
	Def_searchringtime = 1
)

type AckGossiping struct {
	ID    uint32
	Timer *time.Timer
}

type AckGossipingSlice []*AckGossiping

type Gossipedlistindextuple struct {
	list  int
	index int
}

type OriginState struct {
	gossipedinlist map[uint32]Gossipedlistindextuple
	next           uint32
	stateLock      sync.Mutex
}

type Nexthoptuple struct {
	maxid uint32
	addr  *net.UDPAddr
}

const (
	Enum_rumorlist    = iota
	Enum_routerumlist = iota
	Enum_tlclist      = iota
)

/*
lock order:
			peersLock
				then ackwaitLock
						then originrumorlock
							then	originnextLock or rumorlistLock
*/
type Gossiper struct {
	print             int
	name              string
	gaddr             string
	gconnect          *net.UDPConn
	uiconnect         *net.UDPConn
	peerstring        string
	peers             []*net.UDPAddr
	peersLock         sync.Mutex
	seq               uint32
	antientroptimeout time.Duration
	acktimeout        time.Duration
	rumorlist         []*RumorMessage
	rumorlistLock     sync.Mutex
	routerumlist      []*RumorMessage
	routerumlistLock  sync.Mutex
	originstat        map[string]*OriginState
	originstatLock    sync.RWMutex
	//ackwaiting[addresse][origin]
	ackwaiting  map[string](map[string]AckGossipingSlice)
	ackwaitLock sync.Mutex
	rttimer     time.Duration
	nexthop     map[string]*Nexthoptuple
	nexthopLock sync.Mutex
	//privlist[origin]
	privlist     map[string][]*PrivateMessage
	privlistLock sync.Mutex
	//hashready[string(HashValue)]
	hashready     map[string]string
	hashreadyLock sync.RWMutex
	//hashrequested[string(HashValue)]
	hashrequested       map[string]hashrequesttuple
	hashrequestedLock   sync.Mutex
	reqtimeout          time.Duration
	files               []*lockedSearchResult
	filesLock           sync.Mutex
	searchduplicate     []*SearchRequest
	searchduplicatelock sync.Mutex
	pendingsearch       *search
	lastcompletedsearch *search
	N                   int
	stubborntimeout     time.Duration
	hoplimit            uint32
	tlclist             []*TLCMessage
	tlclistLock         sync.Mutex
	tlcackwaiting       AckTLCSlice
	tlcackwaitLock      sync.Mutex
	rawlog							string
}

type hashrequesttuple struct {
	Name           string
	Ticker         *time.Ticker
	printstr       string
	externChunkNum uint64
	file           *lockedSearchResult
	searched       *returnedfilesstruct
}

type lockedSearchResult struct {
	lock         sync.Mutex
	searchresult SearchResult
}

type search struct {
	ticker *time.Ticker
	lock   sync.Mutex
	//returnedfiles[string(metahash)]
	returnedfiles map[string]*returnedfilesstruct
	matchedfiles  []*returnedfilesstruct
}

type returnedfilesstruct struct {
	selectedname string
	chunklist    []uint64
	chunkcount   uint64
	//resultlist[nodename]
	resultlist map[string]*SearchResult
}

type AckTLC struct {
	lock   sync.Mutex
	ID     uint32
	Ticker *time.Ticker
	//map[origin]
	acked map[string]bool
}

type AckTLCSlice []*AckTLC
