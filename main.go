package main

import (
	"fmt"
	"flag"
	"github.com/escicu/Peerster/types"
	"net"
	"sync"
	"strings"
//	"https://github.com/dedis/protobuf"
	"github.com/escicu/Peerster/comm"
)

type Gossiper struct{
	name string
	gaddr *net.UDPAddr
	gconnect *net.UDPConn
	uiconnect *net.UDPConn
	simplemode bool
	peerstring string
	peers []*net.UDPAddr
}
func udpaddrequal(a1,a2 *net.UDPAddr) bool{
	return (a1.Port==a2.Port && a1.IP.Equal(a2.IP))
}

func NewGossiper(uiport *string,gossipaddr *string,name *string,peers *string,simple bool,wg *sync.WaitGroup) (*Gossiper,error){
	var gossiper Gossiper

	udpaddrui,err := net.ResolveUDPAddr("udp", ":"+*uiport)
	if err != nil {
		return nil,err
	}
	gossiper.gaddr,err = net.ResolveUDPAddr("udp", *gossipaddr)
	if err != nil {
		return nil,err
	}

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
			peerstrslice=append(peerstrslice,udpaddradd.String())
		}
	}

	gossiper.peerstring=strings.Join(peerstrslice,",")

	gossiper.simplemode=simple
	gossiper.name=*name
	gossiper.gconnect, err = net.ListenUDP("udp", gossiper.gaddr)
	if err != nil {
		return nil,err
	}
	gossiper.uiconnect, err = net.ListenUDP("udp", udpaddrui)
	if err != nil {
		return nil,err
	}

	wg.Add(2)
	go gossiper.threadGossip(wg)
	go gossiper.threadUI(wg)

	return &gossiper,nil
}

func (g *Gossiper) threadUI(wg *sync.WaitGroup){
	defer wg.Done()

	for {
		n,mess,addr,err:=comm.ReadMessage(g.uiconnect)
		if err != nil {
			fmt.Printf("error %d %+v",n,addr)
		}

		smess:=types.SimpleMessage{g.name,g.gaddr.String(),string(mess)}
		sendpack:=types.GossipPacket{Simple:&smess}

		for _,v:= range g.peers{
			  comm.SendPacket(&sendpack,g.gconnect,v)
	  }
		fmt.Printf("CLIENT MESSAGE %s\n",string(mess))
	}
}


func (g *Gossiper) threadGossip(wg *sync.WaitGroup){
	defer wg.Done()

	for {
		pack,addr,err:=comm.ReadPacket(g.gconnect)
		if err != nil {
			fmt.Printf("error threadgossip readpacket %+v, %+v",addr,err)
		}

		ind := -1
		for i,u:=range g.peers{
			if(udpaddrequal(addr,u)){
				ind=i
				break
			}
		}

		if(ind<0){
			ind=len(g.peers)
			g.peers=append(g.peers,addr)
			g.peerstring=g.peerstring+","+addr.String()
		}

		fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n",pack.Simple.Originalname,pack.Simple.RelayPeerAddr,pack.Simple.Contents)

		simplemess:=pack.Simple
		simplemess.RelayPeerAddr=g.gaddr.String()

		comm.BroadcastPacket(pack,g.gconnect,g.peers[:ind])
		comm.BroadcastPacket(pack,g.gconnect,g.peers[ind+1:])


		fmt.Printf("PEERS %s\n",g.peerstring)
	}
}

func main() {

	var uiport=flag.String("UIPort","8080","port for the UI client (default \"8080\")")
	var gossipaddr=flag.String("gossipAddr","127.0.0.1:5000","ip:port for the gossiper (default \"127.0.0.1:5000\")")
	var name=flag.String("name","defaultname","name of the gossiper")
	var peers=flag.String("peers","127.0.0.1:5000","comma separated list of peers of the form ip:port")
	var simple=flag.Bool("simple",false,"run gossiper in simple broadcast mode")
	flag.Parse()

	var wg sync.WaitGroup

	_,err:=NewGossiper(uiport,gossipaddr,name,peers,*simple ,&wg)
	if err != nil {
		return
	}

	wg.Wait()
}
