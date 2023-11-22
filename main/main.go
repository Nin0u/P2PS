package main

import (
	"fmt"
	"log"
	"net"
)

func recv(conn net.PacketConn) {
	message := make([]byte, 65535+7) //TODO: + une signature

	for {
		nb_byte, addr_sender, err := conn.ReadFrom(message)

		if err != nil {
			log.Fatal("rcv error :", err)
			continue
		}

		t := getType(message)
		switch t {
		case NoOp:
			break
		case Error:
			handleError(message, nb_byte, addr_sender)
			break
		case Hello:
			handleHello(message, nb_byte, addr_sender)
			break
		case PublicKey:
			handlePublicKey(message, nb_byte, addr_sender)
			break
		case Root:
			handleRoot(message, nb_byte, addr_sender)
			break
		case GetDatum:
			handleGetDatum(message, nb_byte, addr_sender)
			break
		case NatTraversalRequest:
			//TODO: Plus Tard
			break
		case NatTraversal:
			//TODO: Plus tard
			break

		case ErrorReply:
			handleErrorReply(message, nb_byte, addr_sender)
			break
		case HelloReply:
			handleHelloReply(message, nb_byte, addr_sender)
			break
		case Datum:
			handleDatum(message, nb_byte, addr_sender)
			break
		case NoDatum:
			handleNoDatum(message, nb_byte, addr_sender)
			break
		default:
			fmt.Printf("Unknown/Unexpected message type %d\n", t)
			break
		}
	}
}

func main() {
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	// go recv(conn)
	// appelle au cli
}
