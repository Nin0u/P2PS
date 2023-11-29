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

func Recv(client *http.Client, conn net.PacketConn) {
	message := make([]byte, 65535+7) //TODO: + une signature

	for {
		nb_byte, addr_sender, err := conn.ReadFrom(message)
		if err != nil {
			fmt.Print("rcv error :")
			fmt.Println(err)
			continue
		}

		t := GetType(message)

		// Treat Hello separately because it handles handshake between peers
		if t == Hello {
			HandleHello(client, conn, message, nb_byte, addr_sender, username)
		} else if t == HelloReply {
			HandleHelloReply(client, message, nb_byte, addr_sender)
		} else {
			err = CheckHandShake(addr_sender)
			if err != nil {
				if debug {
					fmt.Println(err)
				}
				continue
			}

			switch t {
			case NoOp:

			case Error:
				HandleError(message)

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

			case Datum:
				HandleDatum(message, nb_byte, addr_sender, conn)

			case NoDatum:
				HandleNoDatum(message, nb_byte, addr_sender)

			default:
				fmt.Printf("Unknown/Unexpected message type %d\n", t)
			}
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

	go Recv(client, conn)
	cli(client, conn)
}
