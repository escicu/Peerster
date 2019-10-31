package main

import (
	"fmt"
	"flag"
	"sync"
)

func main() {

	var uiport=flag.String("UIPort","8080","port for the UI client (default \"8080\")")
	var gossipaddr=flag.String("gossipAddr","127.0.0.1:5000","ip:port for the gossiper (default \"127.0.0.1:5000\")")
	var name=flag.String("name","defaultname","name of the gossiper")
	var peers=flag.String("peers","127.0.0.1:5000","comma separated list of peers of the form ip:port")
	var simple=flag.Bool("simple",false,"run gossiper in simple broadcast mode")
	var antientropy=flag.Uint("antiEntropy",10,
		"Use the given timeout in seconds for anti-entropy (relevant only for Part 2). If the flag is absent, the default anti-entropy duration is 10 seconds.")
	var rtimer=flag.Int("rtimer",0,"Timeout in seconds to send route rumors. 0 (default) means disable sending route rumors.")
	flag.Parse()

	var wg sync.WaitGroup

	_,err:=NewGossiper(uiport,gossipaddr,name,peers,*simple ,*antientropy , *rtimer,&wg)
	if err != nil {
		fmt.Printf ("err %+v",err)
		return
	}

	wg.Wait()
}
