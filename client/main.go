package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/dedis/protobuf"
	"net"
	"os"
)

type Message struct {
	Text        string
	Destination *string
	File        *string
	Request     *[]byte
}

func SendMessageClient(message *Message, con net.Conn) {

	const udpmaxsize = 1 << 16

	b, err := protobuf.Encode(message)
	if err != nil {
		fmt.Printf("error comm client encode %+v\n", err)
	}
	n, err := con.Write(b)
	if err != nil {
		fmt.Printf("error comm client %d, %+v\n", n, err)
	}
}

func main() {

	bitflagpresent:= uint8(0)
	var uiport = flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	var dest = flag.String("dest", "", "destination for the private message; ​ can be omitted")
	var msg = flag.String("msg", "", "message to be sent; if the -dest flag is present, this is a private message, otherwise it’s a rumor message")
	var file = flag.String("file", "", "file to be indexed by the gossiper")
	var req = flag.String("request", "", "request a chunk or metafile of this hash")
	flag.Parse()
	flag.Visit(func(arg1 *flag.Flag) {
		switch arg1.Name {
		case "UIPort":
			bitflagpresent = bitflagpresent | 1
		case "dest":
			bitflagpresent = bitflagpresent | 2
		case "msg":
			bitflagpresent = bitflagpresent | 4
		case "file":
			bitflagpresent = bitflagpresent | 8
		case "request":
			bitflagpresent = bitflagpresent | 16

		}
	})

	con, err := net.Dial("udp", "127.0.0.1:"+*uiport)
	if err != nil {
		fmt.Printf("error in connecting to the gossiper node.\n")
		os.Exit(2)
	}

	switch bitflagpresent {
	case 5:
		mess := &Message{Text: *msg, Destination: dest, File: nil, Request: nil}
		SendMessageClient(mess, con)
	case 7:
		mess := &Message{Text: *msg, Destination: dest, File: nil, Request: nil}
		SendMessageClient(mess, con)
	case 9:
		mess := &Message{Text: *msg, Destination: dest, File: file, Request: nil}
		SendMessageClient(mess, con)
	case 27:
		h, err := hex.DecodeString(*req)
		if err != nil {
			fmt.Printf("ERROR (Unable to decode hex hash)​.\n")
			os.Exit(1)
		} else {
			mess := &Message{Text: *msg, Destination: dest, File: file, Request: &h}
			SendMessageClient(mess, con)
		}
	default:
			fmt.Printf("ERROR (Bad argument combination).\n")
			os.Exit(1)
	}

}
