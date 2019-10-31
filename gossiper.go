package main

import (
	"net"
	"sync"
	"strings"
	"math/rand"
	"time"
)


func NewGossiper(uiport *string,gossipaddr *string,name *string,peers *string,simple bool, aetimeout uint,rtimer int,wg *sync.WaitGroup) (*Gossiper,error){
	var gossiper Gossiper

	udpaddrui,err := net.ResolveUDPAddr("udp", ":"+*uiport)
	if err != nil {
		return nil,err
	}

	//extract port of gossip addresse, needed behind nat
	portind:=strings.IndexAny(*gossipaddr,":")

	gaddrudp,err := net.ResolveUDPAddr("udp", (*gossipaddr)[portind:])
	if err != nil {
		return nil,err
	}

	gossiper.ackwaiting=make(map[string](map[string]AckRumorSlice))

	peerslice:=strings.Split(*peers,",")
	var peerstrslice []string
peeradding:
	for _,v:=range peerslice{
		if(v!=""){
			udpaddradd,err := net.ResolveUDPAddr("udp", v)
			if err != nil {
				return nil,err
			}
			for _,u:=range gossiper.peers{
				if(udpaddrequal(udpaddradd,u)){
					continue peeradding
				}
			}
			gossiper.peers=append(gossiper.peers,udpaddradd)
			gossiper.ackwaiting[udpaddradd.String()]=make(map[string]AckRumorSlice)
			peerstrslice=append(peerstrslice,udpaddradd.String())
		}
	}

	gossiper.peerstring=strings.Join(peerstrslice,",")

	gossiper.name=*name
	gossiper.gconnect, err = net.ListenUDP("udp", gaddrudp)
	if err != nil {
		return nil,err
	}
	gossiper.uiconnect, err = net.ListenUDP("udp", udpaddrui)
	if err != nil {
		return nil,err
	}

	gossiper.seq=1
	gossiper.gaddr=*gossipaddr
	gossiper.antientroptimeout=time.Duration(aetimeout)*time.Second
	gossiper.acktimeout=time.Duration(10)*time.Second
	gossiper.reqtimeout=time.Duration(5)*time.Second
	gossiper.originstat=make(map[string]*OriginState)
	gossiper.rttimer=time.Duration(rtimer)*time.Second
	gossiper.nexthop=make(map[string]*Nexthoptuple)
	gossiper.privlist=make(map[string][]*PrivateMessage)

	var ostat OriginState
	ostat.rumorinlist=make(map[uint32]Rumorlistindextuple)
	ostat.next=gossiper.seq
	gossiper.originstat[gossiper.name]=&ostat

	gossiper.listfile=make([]*Fileintern,5)
	gossiper.hashready=make(map[string]string)
	gossiper.hashrequested=make(map[string]hashrequesttuple)


	rand.Seed(time.Now().UnixNano())
	wg.Add(2)
	go gossiper.threadGossip(wg)
	if simple {
		go gossiper.threadUIsimple(wg)
	} else {
			go gossiper.threadUIcomplet(wg)
			if(aetimeout>0){
				tickerantientropy:=time.NewTicker(gossiper.antientroptimeout)
				go func(){
					for _ = range tickerantientropy.C {
						gossiper.sendStatPacket(nil)
					}
				}()
			}
			if(rtimer>0){
				gossiper.sendRumorMonger(&RumorMessage{gossiper.name,gossiper.seq,""},-1,nil)
				gossiper.seq++
				tickerrttimer:=time.NewTicker(gossiper.rttimer)
				go func(){
					for _ = range tickerrttimer.C {
						gossiper.sendRumorMonger(&RumorMessage{gossiper.name,gossiper.seq,""},-1,nil)
						gossiper.seq++
					}
				}()
			}

	}

	return &gossiper,nil
}
