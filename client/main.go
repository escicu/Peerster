package main

import (
	"fmt"
	"flag"
	"net"
  "github.com/dedis/protobuf"
)

type Message struct{
  Text string
}

func SendMessageClient(message *Message,con net.Conn){

  const udpmaxsize =1<<16

	b,err :=protobuf.Encode(message)
	if err != nil {
		fmt.Printf("error comm client encode %+v\n",err)
	}
  n, err := con.Write(b)
  if err != nil {
    fmt.Printf("error comm client %d, %+v\n",n,err)
  }
}

func main() {

	var uiport=flag.String("UIPort","8080","port for the UI client (default \"8080\")")
	var msg=flag.String("msg","","message to be sent")
	flag.Parse()

	con,err:=net.Dial("udp","127.0.0.1:"+*uiport)
	if err != nil {
	// handle error
	}

	mess:=&Message{Text:*msg}

	SendMessageClient(mess,con)
}
