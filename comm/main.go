package comm

import (
	"fmt"
	"github.com/escicu/Peerster/types"
	"net"
  "github.com/dedis/protobuf"
)

func SendMessage(message []byte,con *net.UDPConn,addr *net.UDPAddr){

  const udpmaxsize =1<<16

  n, err := con.WriteToUDP(message,addr)
  if err != nil {
    fmt.Printf("error comm %d, %+v, addr %+v\n",n,err,*addr)
  }
}

func SendPacket(pack *types.GossipPacket,con *net.UDPConn,addr *net.UDPAddr){
  b,_ :=protobuf.Encode(pack)
  SendMessage(b,con,addr)
}

func BroadcastPacket(pack *types.GossipPacket,con *net.UDPConn,addrList []*net.UDPAddr){
  b,_ :=protobuf.Encode(pack)
  for _,v :=range addrList{
    SendMessage(b,con,v)
  }
}

func ReadMessage(con *net.UDPConn) (int, []byte, *net.UDPAddr, error){

  const udpmaxsize =1<<16

  buf := make([]byte, udpmaxsize)

  n, addr, err := con.ReadFromUDP(buf)
  if err != nil {
    return n,nil,addr,err
  }

  return n, buf[:n],addr,err

}

func ReadPacket(con *net.UDPConn) (*types.GossipPacket, *net.UDPAddr, error){
  pack := types.GossipPacket{}

  _,mess,addr,err:=ReadMessage(con)
  if err != nil {
    return nil,addr,err
  }

  err=protobuf.Decode(mess,&pack)
  return &pack,addr,err
}
