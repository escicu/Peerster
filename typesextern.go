package main


type Message struct {
	Text string
	Destination *string
	File *string
	Request *[]byte
}

type GossipPacket struct{
  Simple *SimpleMessage
  Rumor *RumorMessage
  Status *StatusPacket
	Private *PrivateMessage
	DataRequest *DataRequest
	DataReply *DataReply
}


type SimpleMessage struct {
  Originalname string
  RelayPeerAddr string
  Contents string
}

type PrivateMessage struct {
	Origin string
	ID uint32
	Text string
	Destination string
	HopLimit uint32
}

type RumorMessage struct{
  Origin string
  ID uint32
  Text string
}

type StatusPacket struct{
  Want []PeerStatus
}


type DataRequest struct {
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
}

type DataReply struct {
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
	Data []byte
}

type PeerStatus struct {
  Identifier string
  NextID uint32
}
