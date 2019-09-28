package main

import (
	"fmt"
	"flag"
//	"https://github.com/escicu/Peerster/types"4
	"net"
//	"https://github.com/dedis/protobuf"
)

func main() {

	var uiport=flag.String("UIPort","8080","port for the UI client (default \"8080\")")
	var msg=flag.String("msg","","message to be sent")
	flag.Parse()

	con,err:=net.Dial("udp","127.0.0.1:"+*uiport)
	if err != nil {
	// handle error
	}

	const udpmaxsize =1<<16

	buf := make([]byte, 20)

	for i := 0; i < 20; i++ {
		buf[i]=65
	}

	n, err := con.Write(buf)
	if err != nil {
	fmt.Printf("error %d, %s\n",n,err)
	}

	fmt.Printf("uiport : %s\n",*uiport)
	fmt.Printf("msg : %s\n",*msg)
}
