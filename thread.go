package main

import (
	"fmt"
	"sync"
	"math/rand"
	"net/http"
)


func (g *Gossiper) threadUIsimple(wg *sync.WaitGroup){
	defer wg.Done()

	for {
		mess,addr,err:=ReadMessageClient(g.uiconnect)
		if err != nil {
			fmt.Printf("error %+v",addr)
		}

		smess:=SimpleMessage{g.name,g.gaddr,mess.Text}
		sendpack:=GossipPacket{Simple:&smess}

		g.peersLock.Lock()
		BroadcastPacket(&sendpack,g.gconnect,g.peers)
		g.peersLock.Unlock()
		fmt.Printf("CLIENT MESSAGE %s\n",mess.Text)
	}
}


func (g *Gossiper) threadUIweb(){
	http.Handle("/", http.FileServer(http.Dir("./html")))
	http.HandleFunc("/messages", g.getRumorMessageHandler)
	http.HandleFunc("/node", g.getRumorMessageHandler)
	http.HandleFunc("/id", g.getRumorMessageHandler)
	for {
		http.ListenAndServe(":8080", nil)
		// error handling, etc..
	}
}


func (g *Gossiper) threadUIcomplet(wg *sync.WaitGroup){
		defer wg.Done()

		go g.threadUIweb()

		for {
			mess,addr,err:=ReadMessageClient(g.uiconnect)
			if err != nil {
				fmt.Printf("error %+v",addr)
			}

			rmess:=&RumorMessage{g.name,g.seq,mess.Text}

			fmt.Printf("CLIENT MESSAGE %s\n",mess.Text)

			g.addRumorClient(rmess)
			g.seq++

			g.sendRumorMonger(rmess,-1,nil)
		}

}

func (g *Gossiper) threadGossip(wg *sync.WaitGroup){
	defer wg.Done()

	for {
		pack,addr,err:=ReadPacket(g.gconnect)
		fmt.Printf("read\n")
		if err != nil {
			fmt.Printf("error threadgossip readpacket %+v, %+v",addr,err)
		}

		g.peersLock.Lock()
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
			if(ind==0){
				g.peerstring=g.peerstring+addr.String()
			} else {
				g.peerstring=g.peerstring+","+addr.String()
			}
			g.ackwaitLock.Lock()
			g.ackwaiting[addr.String()]=make(map[string]AckRumorSlice)
			g.ackwaitLock.Unlock()
		}

		g.peersLock.Unlock()

		switch {
		case pack.Simple!=nil:
			fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n",pack.Simple.Originalname,pack.Simple.RelayPeerAddr,pack.Simple.Contents)
			fmt.Printf("PEERS %s\n",g.peerstring)
			simplemess:=pack.Simple
			simplemess.RelayPeerAddr=g.gaddr
			g.peersLock.Lock()
			BroadcastPacket(pack,g.gconnect,g.peers[:ind])
			BroadcastPacket(pack,g.gconnect,g.peers[ind+1:])
			g.peersLock.Unlock()
		case pack.Rumor!=nil:
			fmt.Printf("RUMOR origin %s from %s ID %d contents %s\n",pack.Rumor.Origin,addr.String(),pack.Rumor.ID,pack.Rumor.Text)
			fmt.Printf("PEERS %s\n",g.peerstring)


			if( g.addRumorPeer(pack.Rumor) ){
				//StatusPacket ack
				g.sendStatPacket(addr)

				//MONGERING next peer
				g.sendRumorMonger(pack.Rumor,ind,nil)
			}



		case pack.Status!=nil:
			fmt.Printf("STATUS from %s",addr.String())
			vectorclockreceive:=make(map[string]uint32)
			acked:=false
			rumori:=0
			g.ackwaitLock.Lock()
			maporigack:=g.ackwaiting[addr.String()]
				for _,ps:=range pack.Status.Want{
					fmt.Printf(" peer %s nextID %d",ps.Identifier,ps.NextID)
					vectorclockreceive[ps.Identifier]=ps.NextID
					ackslice,exist:=maporigack[ps.Identifier]
					if(exist){
							for len(ackslice)>0 && ackslice[0].ID<ps.NextID {
								ackslice[0].Timer.Stop()
								rumori=g.originrumor[ps.Identifier][ackslice[0].ID]
								ackslice=ackslice[1:]
								acked=true
							}
					}
					maporigack[ps.Identifier]=ackslice
				}
				g.ackwaitLock.Unlock()
				fmt.Printf("\n")
			fmt.Printf("PEERS %s\n",g.peerstring)


			continuation:=true
			sendstat:=false
			g.originnextLock.Lock()
			for k,v:=range vectorclockreceive{
				if (v<g.originnext[k]){
					g.sendRumorMonger(g.rumorlist[g.originrumor[k][v]],0,addr)
					continuation = false
					break
				}
			}
			if(continuation){
				for k,v:=range vectorclockreceive{
					if (v>g.originnext[k]){
						sendstat=true
						continuation = false
						break
					}
				}
			}
			g.originnextLock.Unlock()
			if(sendstat){//because of lock (grrrrrrrrrrrrrrrrr)
				g.sendStatPacket(addr)
			}

			if(continuation){
				fmt.Printf("IN SYNC WITH %s\n",addr.String())

				if(acked){
					flip:=rand.Intn(2)
					if(flip==1){
						indcoin,addrcoin:=g.randomPeer(ind)
						fmt.Printf("FLIPPED COIN sending rumor to %s\n",addrcoin.String())
						g.sendRumorMonger(g.rumorlist[rumori],indcoin,addrcoin)
					}
				}
			}



		}


	}
}
