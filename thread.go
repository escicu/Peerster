package main

import (
	"fmt"
	"sync"
	"math/rand"
	"net/http"
	"io/ioutil"
	"crypto/sha256"
	"bytes"
	"strings"
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
	http.HandleFunc("/message", g.RumorMessageHandler)
	http.HandleFunc("/node", g.NodeHandler)
	http.HandleFunc("/id", g.IdHandler)
	http.HandleFunc("/private", g.PrivateMessageHandler)
	http.HandleFunc("/origin", g.OriginListHandler)
	http.HandleFunc("/download", g.DownloadFileHandler)
	http.HandleFunc("/share", g.ShareFileHandler)
	for {
		http.ListenAndServe(":8080", nil)
		//fmt.Printf("http listenandserve stopped: %+v\n",err)
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

			if(*mess.Destination==""){//public message or file sharing

				if(mess.File==nil){
					rmess:=&RumorMessage{g.name,g.seq,mess.Text}

					fmt.Printf("CLIENT MESSAGE %s\n",mess.Text)

					g.addRumorClient(rmess)
					g.seq++

					g.sendRumorMonger(rmess,-1,nil)
				}else{
					g.FileSharePrepare(*mess.File)
				}

			}else{//private message or file download
				if(mess.Request==nil){
					pmess:=&PrivateMessage{g.name,0,mess.Text,*mess.Destination,Def_hoplimit}

					fmt.Printf("CLIENT MESSAGE %s dest %s\n",mess.Text, *mess.Destination)

					g.sendDirect(&GossipPacket{Private:pmess},*mess.Destination)
				}else{
					fmt.Printf("DOWNLOADING metafile of %s from %s\n",*mess.File, *mess.Destination)
					g.FileDownloadMetaStart(*mess.File, *mess.Destination, *mess.Request)
				}
			}

		}

}

func (g *Gossiper) threadGossip(wg *sync.WaitGroup){
	defer wg.Done()

	for {
		pack,addr,err:=ReadPacket(g.gconnect)
		if err != nil {
			fmt.Printf("error threadgossip readpacket %+v, %+v",addr,err)
		}

		ind:=g.addPeer(addr)

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

			go func ()  {
					if( g.addRumorPeer(pack.Rumor) ){
						//StatusPacket ack
						go g.sendStatPacket(addr)

						//MONGERING next peer
						g.sendRumorMonger(pack.Rumor,ind,nil)
						//no optimize
						//go g.sendRumorMonger(pack.Rumor,-1,nil)
					}

					g.nexthopLock.Lock()
					tuple,exist:=g.nexthop[pack.Rumor.Origin]
					if(!exist || tuple.maxid<pack.Rumor.ID){
						g.nexthop[pack.Rumor.Origin]=&Nexthoptuple{pack.Rumor.ID,addr}
						if(pack.Rumor.Text!=""){
							fmt.Printf("DSDV %s %s\n",pack.Rumor.Origin,addr.String())
						}
					}
					g.nexthopLock.Unlock()
				}()
		case pack.Status!=nil:
			fmt.Printf("STATUS from %s",addr.String())
			vectorclockreceive:=make(map[string]uint32)
			acked:=false
			var rumordforwardorigin string
			var rumorforwardid uint32
			g.ackwaitLock.Lock()
			maporigack:=g.ackwaiting[addr.String()]
				for _,ps:=range pack.Status.Want{
					fmt.Printf(" peer %s nextID %d",ps.Identifier,ps.NextID)
					vectorclockreceive[ps.Identifier]=ps.NextID
					ackslice,exist:=maporigack[ps.Identifier]
					if(exist){
							for len(ackslice)>0 && ackslice[0].ID<ps.NextID {
								ackslice[0].Timer.Stop()
								rumordforwardorigin=ps.Identifier
								rumorforwardid=ackslice[0].ID
								ackslice=ackslice[1:]
								acked=true
							}
					}else{
						g.addCheckOrigin(ps.Identifier)
					}
					maporigack[ps.Identifier]=ackslice
				}
				g.ackwaitLock.Unlock()
				fmt.Printf("\n")
			fmt.Printf("PEERS %s\n",g.peerstring)


			continuation:=true
			sendstat:=false
			var rm *RumorMessage
			sendrm:=false

			g.originstatLock.RLock()
			for k,v:=range vectorclockreceive{
				stat:=g.originstat[k]
				if (v<stat.next){
					lt:=stat.rumorinlist[v]
					switch lt.list {
					case Enum_rumorlist:
						g.rumorlistLock.Lock()//because locking
						rm=g.rumorlist[lt.index]
						g.rumorlistLock.Unlock()
					case Enum_routerumlist:
						g.routerumlistLock.Lock()//because locking
						rm=g.routerumlist[lt.index]
						g.routerumlistLock.Unlock()

					}
					sendrm=true
					continuation = false
					break
				}
			}
			if(continuation){
				for k,v:=range vectorclockreceive{
					if (v>g.originstat[k].next){
						sendstat=true
						continuation = false
						break
					}
				}
			}
			g.originstatLock.RUnlock()
			switch {//because of lock (grrrrrrrrrrrrrrrrr)
			case sendrm:
					g.sendRumorMonger(rm,-1,addr)
			case sendstat:
					g.sendStatPacket(addr)
			}


			if(continuation){
				fmt.Printf("IN SYNC WITH %s\n",addr.String())

				if(acked){
					flip:=rand.Intn(2)
					if(flip==1){
						indcoin,addrcoin:=g.randomPeer(ind)
						fmt.Printf("FLIPPED COIN sending rumor to %s\n",addrcoin.String())
						g.originstatLock.RLock()//because locking
						rumortuple:=g.originstat[rumordforwardorigin].rumorinlist[rumorforwardid]
						g.originstatLock.RUnlock()
						var rms *RumorMessage
						switch rumortuple.list {
						case Enum_rumorlist:
							g.rumorlistLock.Lock()//because locking
							rms=g.rumorlist[rumortuple.index]
							g.rumorlistLock.Unlock()
						case Enum_routerumlist:
							g.routerumlistLock.Lock()//because locking
							rms=g.routerumlist[rumortuple.index]
							g.routerumlistLock.Unlock()
						}
						g.sendRumorMonger(rms,indcoin,addrcoin)
					}
				}
			}


		case pack.Private!=nil:
			if(pack.Private.Destination==g.name){
				priv:=pack.Private
				fmt.Printf("PRIVATE origin %s hop-limit %d contents %s\n",priv.Origin,priv.HopLimit,priv.Text)
				g.privlistLock.Lock()
				l,exist:=g.privlist[priv.Origin]
				if(exist){
					g.privlist[priv.Origin]=append(l, priv)
				}else{
					g.privlist[priv.Origin]=[]*PrivateMessage{priv}
				}
				g.privlistLock.Unlock()

			}else{
				priv:=pack.Private
				if(priv.HopLimit>0){
					priv.HopLimit--
					g.sendDirect(pack, priv.Destination)
				}
			}

		case pack.DataRequest!=nil:
			go func ()  {
					if(pack.DataRequest.Destination==g.name){
						g.hashreadyLock.Lock()
						filename,exist:=g.hashready[string(pack.DataRequest.HashValue)]
						g.hashreadyLock.Unlock()
						if(exist){
							val,err:=ioutil.ReadFile(filename)
							if(err==nil){
									h:=sha256.Sum256(val)
									g.sendDirect(&GossipPacket{DataReply:&DataReply{Origin:g.name,Destination:pack.DataRequest.Origin,HopLimit:Def_hoplimit,HashValue:h[:],Data:val}},pack.DataRequest.Origin)
								}
							}else{
								g.sendDirect(&GossipPacket{DataReply:&DataReply{Origin:g.name,Destination:pack.DataRequest.Origin,HopLimit:Def_hoplimit,HashValue:pack.DataRequest.HashValue}},pack.DataRequest.Origin)
							}
					}else{
						d:=pack.DataRequest
						if(d.HopLimit>0){
							d.HopLimit--
							g.sendDirect(pack, d.Destination)
						}
					}
				}()
		case pack.DataReply!=nil:
			go func ()  {
						if(pack.DataReply.Destination==g.name){
							//fmt.Printf("DOWNLOADING metafile of flyingDrone.gif from anotherPeer")
							h:=pack.DataReply.HashValue
							g.hashrequestedLock.Lock()
							tuple,exist:=g.hashrequested[string(h)]
							if(exist){
								val:=pack.DataReply.Data
								hverif:=sha256.Sum256(val)
								if(bytes.Equal(h, hverif[:])){
									err := ioutil.WriteFile(tuple.Name, val, 0644)
									if err == nil {
										if(strings.HasSuffix(tuple.Name,".meta")){
											g.Filemetaprocess(tuple.file,val,pack.DataReply.Origin)
										}
										tuple.Ticker.Stop()
										if(tuple.file!=nil){
											tuple.file.lock.Lock()
											tuple.file.pendingreply--
											if(tuple.file.pendingreply<1){
												go g.ReconstructFile(tuple.file.name)
											}
											tuple.file.lock.Unlock()
										}
										delete(g.hashrequested, string(h))
									}
								}
							}
							g.hashrequestedLock.Unlock()
						}else{
							d:=pack.DataReply
							if(d.HopLimit>0){
								d.HopLimit--
								g.sendDirect(pack, d.Destination)
							}
						}
				}()

		}

	}
}
