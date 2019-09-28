package types

type SimpleMessage struct {
  Originalname String
  RelayPeerAddr string
  Contents string
}

type GossipPacket struct{
  Simple *SimpleMessage
}
