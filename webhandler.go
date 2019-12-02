package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sort"
	"strings"
)

//handle the web communication of normal messages
func (g *Gossiper) RumorMessageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		g.rumorlistLock.Lock()
		msgList := g.rumorlist
		msgListJson, err := json.Marshal(msgList)
		g.rumorlistLock.Unlock()

		if err != nil {
			fmt.Printf("error in webhandler json.marshal message: %+v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(msgListJson)
		}
	case "POST":
		mes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error in webhandler new message: %+v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			mess := string(mes)

			rmess := &RumorMessage{g.name, g.seq, mess}

			s := fmt.Sprintf("CLIENT MESSAGE %s\n", mess)
			g.Printf(1, s)

			g.addRumorClient(rmess)
			g.seq++

			g.sendRumorMonger(rmess, -1, nil)

			w.WriteHeader(http.StatusOK)
		}
	}
}

//handle the web communication related to connected peers
func (g *Gossiper) NodeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		g.peersLock.Lock()
		list := g.peers
		listJson, err := json.Marshal(list)
		g.peersLock.Unlock()

		if err != nil {
			fmt.Printf("error in webhandler json.marshal message: %+v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(listJson)
		}
	case "POST":
		mes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error in webhandler new node: %+v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			udpaddradd, err := net.ResolveUDPAddr("udp", string(mes))
			if err == nil {
				g.addPeer(udpaddradd)
			}
			w.WriteHeader(http.StatusOK)
		}
	}
}

//handle the web communication related to the node name
func (g *Gossiper) IdHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(g.name))
	}
}

type backfrontprivpost struct {
	Dest string
	Text string
}

//handle the web communication related to private messages
func (g *Gossiper) PrivateMessageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, exist := r.URL.Query()["dest"]
		if !exist || len(d) < 1 {
			fmt.Printf("error in webhandler url query: dest not exist\n")
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			g.privlistLock.Lock()
			list := g.privlist[d[0]]
			listJson, err := json.Marshal(list)
			g.privlistLock.Unlock()
			if err != nil {
				fmt.Printf("error in webhandler json.marshal message: %+v\n", err)
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(listJson)
			}
		}
	case "POST":
		mes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error in webhandler new message: %+v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			p := backfrontprivpost{}
			err := json.Unmarshal(mes, &p)
			if err != nil {
				fmt.Printf("error in webhandler json.marshal message: %+v", err)
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				pmess := &PrivateMessage{g.name, 0, p.Text, p.Dest, g.hoplimit}

				g.sendDirect(&GossipPacket{Private: pmess}, p.Dest)

				w.WriteHeader(http.StatusOK)
			}
		}
	}
}

//handle the web communication related to the Origins/nodes names in network
func (g *Gossiper) OriginListHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		g.nexthopLock.Lock()
		msgList := []string{}
		for k, _ := range g.nexthop {
			msgList = append(msgList, k)
		}
		g.nexthopLock.Unlock()
		sort.Strings(msgList)
		msgListJson, err := json.Marshal(msgList)

		if err != nil {
			fmt.Printf("error in webhandler json.marshal message: %+v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(msgListJson)
		}
	}
}

type backfrontfiledown struct {
	Name string
	Dest string
	Hash string
}

//handle the web communication related to downloading files
func (g *Gossiper) DownloadFileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		mes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error in webhandler new message: %+v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			p := backfrontfiledown{}
			err := json.Unmarshal(mes, &p)
			if err != nil {
				fmt.Printf("error in webhandler json.marshal message: %+v", err)
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				h, err := hex.DecodeString(p.Hash)
				if err == nil {
					s := fmt.Sprintf("DOWNLOADING metafile of %s from %s\n", p.Name, p.Dest)
					g.Printf(3, s)
					g.FileDownloadMetaStart(p.Name, p.Dest, h)
				}
				w.WriteHeader(http.StatusOK)
			}
		}
	}
}

//handle the web communication related to sharing files
func (g *Gossiper) ShareFileHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		name, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error in webhandler new message: %+v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			g.FileSharePrepare(string(name))
			w.WriteHeader(http.StatusOK)
		}

	}
}

type backfrontfilesearched struct {
	Name string
	Hash string
}

func (g *Gossiper) SearchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ls := g.lastcompletedsearch
		var list []backfrontfilesearched
		if ls != nil {
			ls.lock.Lock()
			for _, rfs := range ls.matchedfiles {
				var temp *SearchResult
				for _, x := range rfs.resultlist {
					temp = x
					break
				}
				list = append(list, backfrontfilesearched{rfs.selectedname, hex.EncodeToString(temp.MetafileHash)})
			}
			ls.lock.Unlock()
		}
		listJson, err := json.Marshal(list)
		if err != nil {
			fmt.Printf("error in webhandler json.marshal message: %+v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(listJson)
		}

	case "POST":
		mes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error in webhandler new message: %+v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			if g.pendingsearch != nil && g.pendingsearch.ticker != nil {
				g.pendingsearch.ticker.Stop()
			}

			g.pendingsearch = g.newSearch(&SearchRequest{g.name, 0, strings.Split(string(mes), ",")})

			w.WriteHeader(http.StatusOK)

		}
	}
}

func (g *Gossiper) DownloadSearchedHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		mess, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error in webhandler new message: %+v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			p := backfrontfilesearched{}
			err := json.Unmarshal(mess, &p)
			if err != nil {
				fmt.Printf("error in webhandler json.marshal message: %+v", err)
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				h, err := hex.DecodeString(p.Hash)
				if err == nil {
					g.FileDownloadMetaStart(p.Name, "", h)
				}
				w.WriteHeader(http.StatusOK)
			}
		}
	}
}

func (g *Gossiper) RawLogHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(g.rawlog))
	}
}
