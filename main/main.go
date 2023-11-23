package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

var debug bool = true
var username string = ""

func Recv(conn net.PacketConn) {
	message := make([]byte, 65535+7) //TODO: + une signature

	for {
		nb_byte, addr_sender, err := conn.ReadFrom(message)

		if err != nil {
			log.Fatal("rcv error :", err)
			continue
		}

		t := GetType(message)
		switch t {
		case NoOp:

		case Error:
			HandleError(message)

		case Hello:
			HandleHello(conn, message, nb_byte, addr_sender, username)

		case PublicKey:
			HandlePublicKey(conn, message, nb_byte, addr_sender)

		case Root:
			HandleRoot(conn, message, nb_byte, addr_sender)

		case GetDatum:
			HandleGetDatum(conn, message, nb_byte, addr_sender)
		case NatTraversalRequest:
			//TODO: Plus Tard

		case NatTraversal:
			//TODO: Plus tard

		case ErrorReply:
			HandleErrorReply(message)

		case HelloReply:
			HandleHelloReply(message, nb_byte, addr_sender)

		case Datum:
			HandleDatum(message, nb_byte, addr_sender)

		case NoDatum:
			HandleNoDatum(message, nb_byte, addr_sender)

		default:
			fmt.Printf("Unknown/Unexpected message type %d\n", t)
		}
	}
}

func main() {
	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	cli(client, conn)
	// go recv(conn)

}
