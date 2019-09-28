package comm

import (
	"fmt"
//	"https://github.com/escicu/Peerster/types"
	"net"
	"strings"
)

func sendMessage(message []byte,addr string){

	con,err:=net.Dial("udp",addr)
  if err != nil {
    // handle error
  }
  defer con.Close()

  const udpmaxsize =1<<16

  n, err := con.Write(message)
  if err != nil {
    fmt.Printf("error %d, %s\n",n,err)
  }
}

func broadcastMessage(message []byte,addrList string){

  peerslice:=strings.Split(addrList,",")

  for _,v:= peerslice{
    sendMessage(message,v)
  }

}
