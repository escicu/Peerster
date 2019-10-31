package main

import (
	"fmt"
	"net"
	"net/http"
	"encoding/json"
	"encoding/hex"
	"io/ioutil"
)

//handle the web communication of normal messages
func (g *Gossiper) RumorMessageHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
		case "GET":
			g.rumorlistLock.Lock()
			msgList := g.rumorlist
			msgListJson, err := json.Marshal(msgList)
			g.rumorlistLock.Unlock()

			if err != nil {
				fmt.Printf("error in webhandler json.marshal message: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(msgListJson)
			}
		case "POST":
			mes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Printf("error in webhandler new message: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
				mess:=string(mes)

			rmess:=&RumorMessage{g.name,g.seq,mess}

			fmt.Printf("CLIENT MESSAGE %s\n",mess)

			g.addRumorClient(rmess)
			g.seq++

			g.sendRumorMonger(rmess,-1,nil)

				w.WriteHeader(http.StatusOK)
			}
	}
}

//handle the web communication related to connected peers
func (g *Gossiper) NodeHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
		case "GET":
			g.peersLock.Lock()
			list := g.peers
			listJson, err := json.Marshal(list)
			g.peersLock.Unlock()

			if err != nil {
				fmt.Printf("error in webhandler json.marshal message: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(listJson)
			}
		case "POST":
			mes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Printf("error in webhandler new node: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
				udpaddradd,err := net.ResolveUDPAddr("udp", string(mes))
				if err == nil {
					g.addPeer(udpaddradd)
				}
				w.WriteHeader(http.StatusOK)
			}
	}
}

//handle the web communication related to the node name
func (g *Gossiper) IdHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
		case "GET":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(g.name))
	}
}


type backfrontprivget struct{
	dest string
}

//handle the web communication related to private messages
func (g *Gossiper) PrivateMessageHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
	case "GET":
		mes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error in webhandler get private message: %+v",err)
			w.WriteHeader(http.StatusServiceUnavailable)
		}else{
			req:=backfrontprivget{}
			err:=json.Unmarshal(mes, req)
			if err != nil {
				fmt.Printf("error in webhandler json.unmarshal message: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
					g.privlistLock.Lock()
					list := g.privlist[req.dest]
					listJson, err := json.Marshal(list)
					g.privlistLock.Unlock()
					if err != nil {
						fmt.Printf("error in webhandler json.marshal message: %+v",err)
						w.WriteHeader(http.StatusServiceUnavailable)
					}else{
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						w.Write(listJson)
					}
			}
		}
		case "POST":
			mes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Printf("error in webhandler new message: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
				p:=PrivateMessage{}
				err:=json.Unmarshal(mes, p)
				if err != nil {
					fmt.Printf("error in webhandler json.marshal message: %+v",err)
					w.WriteHeader(http.StatusServiceUnavailable)
				}else{
					pmess:=&PrivateMessage{g.name,0,p.Text,p.Destination,49}

					g.sendDirect(&GossipPacket{Private:pmess},p.Destination)

					w.WriteHeader(http.StatusOK)
				}
			}
	}
}

//handle the web communication related to the Origins/nodes names in network
func (g *Gossiper) OriginListHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
		case "GET":
			g.nexthopLock.Lock()
			msgList:=[]string{}
			for k,_:=range g.nexthop{
				msgList=append(msgList,k)
			}
			msgListJson, err := json.Marshal(msgList)
			g.nexthopLock.Unlock()

			if err != nil {
				fmt.Printf("error in webhandler json.marshal message: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(msgListJson)
			}
	}
}


type backfrontfiledown struct{
	name string
	dest string
	hash string
}

//handle the web communication related to downloading files
func (g *Gossiper) DownloadFileHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
		case "POST":
			mes, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Printf("error in webhandler new message: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
				p:=backfrontfiledown{}
				err:=json.Unmarshal(mes, p)
				if err != nil {
					fmt.Printf("error in webhandler json.marshal message: %+v",err)
					w.WriteHeader(http.StatusServiceUnavailable)
				}else{
					h,err:=hex.DecodeString(p.hash)
					if(err==nil){
						g.FileDownloadMetaStart(p.name, p.dest, h)
					}
					w.WriteHeader(http.StatusOK)
				}
			}
	}
}

//handle the web communication related to sharing files
func (g *Gossiper) ShareFileHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
		case "POST":
			name, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Printf("error in webhandler new message: %+v",err)
				w.WriteHeader(http.StatusServiceUnavailable)
			}else{
					g.FileSharePrepare(string(name))
					w.WriteHeader(http.StatusOK)
				}

	}
}
