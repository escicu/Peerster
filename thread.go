package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

func (g *Gossiper) threadUIsimple(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		mess, addr, err := ReadMessageClient(g.uiconnect)
		if err != nil {
			fmt.Printf("error %+v", addr)
		}

		smess := SimpleMessage{g.name, g.gaddr, mess.Text}
		sendpack := GossipPacket{Simple: &smess}

		g.peersLock.Lock()
		BroadcastPacket(&sendpack, g.gconnect, g.peers)
		g.peersLock.Unlock()
		s := fmt.Sprintf("CLIENT MESSAGE %s\n", mess.Text)
		g.Printf(1, s)
	}
}

func (g *Gossiper) threadUIweb() {
	http.Handle("/", http.FileServer(http.Dir("./html")))
	http.HandleFunc("/message", g.RumorMessageHandler)
	http.HandleFunc("/node", g.NodeHandler)
	http.HandleFunc("/id", g.IdHandler)
	http.HandleFunc("/private", g.PrivateMessageHandler)
	http.HandleFunc("/origin", g.OriginListHandler)
	http.HandleFunc("/download", g.DownloadFileHandler)
	http.HandleFunc("/share", g.ShareFileHandler)
	http.HandleFunc("/search", g.SearchHandler)
	http.HandleFunc("/downloadsearched", g.DownloadSearchedHandler)
	http.HandleFunc("/rawlog", g.RawLogHandler)
	for {
		http.ListenAndServe(g.uiconnect.LocalAddr().String(), nil)
	}
}

func (g *Gossiper) threadUIcomplet(wg *sync.WaitGroup) {
	defer wg.Done()

	go g.threadUIweb()

	var s string

	for {
		mess, addr, err := ReadMessageClient(g.uiconnect)
		if err != nil {
			fmt.Printf("error %+v", addr)
		}

		if mess.Destination == nil || *mess.Destination == "" { //public message or file sharing

			switch {
			case mess.File != nil:
				g.FileSharePrepare(*mess.File)
			case mess.Keywords != nil:
				bu := *mess.Budget

				if g.pendingsearch != nil {
					g.pendingsearch.ticker.Stop()
				}

				g.pendingsearch = g.newSearch(&SearchRequest{g.name, bu, strings.Split(*mess.Keywords, ",")})
			case mess.Request != nil:
				g.FileDownloadMetaStart(*mess.File, "", *mess.Request)
			default:
				rmess := &RumorMessage{g.name, g.seq, mess.Text}

				s = fmt.Sprintf("CLIENT MESSAGE %s\n", mess.Text)
				g.Printf(1, s)

				g.addRumorClient(rmess)
				g.seq++

				g.sendRumorMonger(rmess, -1, nil)
			}

		} else { //private message or file download
			if mess.Request == nil {
				pmess := &PrivateMessage{g.name, 0, mess.Text, *mess.Destination, g.hoplimit}

				s = fmt.Sprintf("CLIENT MESSAGE %s dest %s\n", mess.Text, *mess.Destination)
				g.Printf(1, s)

				g.sendDirect(&GossipPacket{Private: pmess}, *mess.Destination)
			} else {
				g.FileDownloadMetaStart(*mess.File, *mess.Destination, *mess.Request)
			}
		}

	}

}

func (g *Gossiper) threadGossip(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		pack, addr, err := ReadPacket(g.gconnect)
		if err != nil {
			fmt.Printf("error threadgossip readpacket %+v, %+v", addr, err)
		}

		ind := g.addPeer(addr)

		switch {
		case pack.Simple != nil:
			go g.processSimplePacket(pack, ind)
		case pack.Rumor != nil:
			go g.processRumorPacket(pack.Rumor, ind, addr)
		case pack.Status != nil:
			go g.processStatusPacket(pack.Status, ind, addr)
		case pack.Private != nil:
			go g.processPrivatePacket(pack)
		case pack.DataRequest != nil:
			go g.processDataRequestPacket(pack)
		case pack.DataReply != nil:
			go g.processDataReplyPacket(pack)
		case pack.SearchRequest != nil:
			go g.processSearchRequestPacket(pack.SearchRequest, ind)
		case pack.SearchReply != nil:
			go g.processSearchReplyPacket(pack)
		case pack.TLCMessage != nil:
			go g.processTLCPacket(pack.TLCMessage, ind, addr)
		case pack.Ack != nil:
			go g.processTLCAckPacket(pack)
		}

	}
}
