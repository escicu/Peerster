package main

import (
	"fmt"
	"flag"
//	"https://github.com/escicu/Peerster/types"
	"net"
	"sync"
	"strings"
//	"https://github.com/dedis/protobuf"
	"https://github.com/escicu/Peerster/comm"
)


func threadUI(uiport string, peers *[]string ,wg *sync.WaitGroup){
	defer wg.Done()

	udpaddr,err := net.ResolveUDPAddr("udp", ":"+uiport)
	ln, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		fmt.Printf("error listenudp %s\n",err)
	}
	defer ln.Close()

	const udpmaxsize =1<<16

	buf := make([]byte, udpmaxsize)
	for {
		n, addr, err := ln.ReadFrom(buf)
		if err != nil {
			fmt.Printf("error %d %s %s",n,addr.Network,addr.String)
		}

		fmt.Printf("CLIENT MESSAGE %s\nPEERS ",string(buf))
		for _,v := range *peers{
			fmt.Printf("%s,",v)
			sendMessage(v,buf[:n])
		}
		fmt.Printf("\n")
	}

}

func threadGossip(gossipaddr string,peers *[]string,wg *sync.WaitGroup){
	defer wg.Done()

	udpaddr,err := net.ResolveUDPAddr("udp", gossipaddr)
	ln, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		fmt.Printf("error listenudp %s\n",err)
	}
	defer ln.Close()

	const udpmaxsize =1<<16

	buf := make([]byte, udpmaxsize)
	for {
		n, addr, err := ln.ReadFrom(buf)
		if err != nil {
			fmt.Printf("error %d %s %s",n,addr.Network,addr.String)
		}
		fmt.Printf("SIMPLE MESSAGE origin 000 from aaa contents %s\nPEERS ",string(buf))
		for _,v := range *peers{
			fmt.Printf("%s,",v)
			sendMessage(v,buf[:n])
		}
		fmt.Printf("\n")
	}
}

func main() {

	var uiport=flag.String("UIPort","8080","port for the UI client (default \"8080\")")
	var gossipaddr=flag.String("gossipAddr","127.0.0.1:5000","ip:port for the gossiper (default \"127.0.0.1:5000\")")
	var name=flag.String("name","defaultname","name of the gossiper")
	var peers=flag.String("peers","127.0.0.1:5000","comma separated list of peers of the form ip:port")
	var simple=flag.Bool("simple",false,"run gossiper in simple broadcast mode")
	flag.Parse()

	peerslice:=strings.Split(*peers,",")
	uniqpeers:=*peers

	for _,v:=range peerslice{
		c:=strings.Count(uniqpeers,v)
		if(c>1){
			uniqpeers=strings.Replace(uniqpeers,v,"",c-1)
		}
	}


	var wg sync.WaitGroup

	wg.Add(2)
	go threadUI(*uiport,&peerslice,&wg)
	go threadGossip(*gossipaddr,&peerslice,&wg)

	fmt.Printf("uiport : %s\n",*uiport)
	fmt.Printf("gossipaddr : %s\n",*gossipaddr)
	fmt.Printf("name : %s\n",*name)
	fmt.Printf("peers : %s\n",*peers)
	fmt.Printf("simple : %t\n",*simple)
	fmt.Printf("uniqpeers : %s\n",uniqpeers)

	wg.Wait()
}
