package main

import (
	"net/http"
	"encoding/json"
)



func (g *Gossiper) getRumorMessageHandler(w http.ResponseWriter, r *http.Request){
	switch r.Method {
		case "GET":
			msgList := g.rumorlist
			msgListJson, _ := json.Marshal(msgList)
			// error handling, etc...
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(msgListJson)
	}
}
