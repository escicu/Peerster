package main

import (
	"flag"
	"fmt"
	"sync"
)

func main() {

	var uiport = flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	var gossipaddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper (default \"127.0.0.1:5000\")")
	var name = flag.String("name", "defaultname", "name of the gossiper")
	var peers = flag.String("peers", "127.0.0.1:5000", "comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	var antientropy = flag.Uint("antiEntropy", 10,
		"Use the given timeout in seconds for anti-entropy (relevant only for Part 2). If the flag is absent, the default anti-entropy duration is 10 seconds.")
	var rtimer = flag.Int("rtimer", 0, "Timeout in seconds to send route rumors. 0 (default) means disable sending route rumors.")
	var n = flag.Int("N", 10, "Number of node in network. (default 10)")
	var stubtim = flag.Int("stubbornTimeout", 5, "Time to resend BlockPublish. (default 5)")
	var hoplimit = flag.Int("hoplimit", Def_hoplimit, "Define hoplimit.")
	var printlevel = flag.Int("printlevel", 1, "Flag to print old homework message. (default 1)")
	var hw3ex2 = flag.Bool("hw3ex2", false, "Flag to use blockchain.")
	var hw3ex3 = flag.Bool("hw3ex3", false, "Flag to use TLC.")
	var hw3ex4 = flag.Bool("hw3ex4", false, "Flag to use ???.")
	flag.Parse()

	if *hw3ex2 {

	}
	if *hw3ex3 {
		fmt.Printf("hw3ex3 not implemented yet")
	}
	if *hw3ex4 {
		fmt.Printf("hw3ex4 not implemented yet")
	}

	var wg sync.WaitGroup

	_, err := NewGossiper(uiport, gossipaddr, name, peers, *simple, *antientropy, *rtimer, *n, *stubtim, *printlevel, *hoplimit, &wg)
	if err != nil {
		fmt.Printf("err %+v", err)
		return
	}

	wg.Wait()
}
