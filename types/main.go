package types

type SimpleMessage struct {
  Originalname string
  RelayPeerAddr string
  Contents string
}

type GossipPacket struct{
  Simple *SimpleMessage
}
