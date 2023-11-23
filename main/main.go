package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"
)

var debug bool = true
var username string = ""

// func recv(conn net.PacketConn) {
// 	message := make([]byte, 65535+7) //TODO: + une signature

// 	for {
// 		nb_byte, addr_sender, err := conn.ReadFrom(message)

// 		if err != nil {
// 			log.Fatal("rcv error :", err)
// 			continue
// 		}

// 		t := getType(message)
// 		switch t {
// 		case NoOp:
// 			break
// 		case Error:
// 			handleError(message)
// 			break
// 		case Hello:
// 			handleHello(conn, message, nb_byte, addr_sender, username)
// 			break
// 		case PublicKey:
// 			handlePublicKey(conn, message, nb_byte, addr_sender)
// 			break
// 		case Root:
// 			handleRoot(conn, message, nb_byte, addr_sender)
// 			break
// 		case GetDatum:
// 			handleGetDatum(conn, message, nb_byte, addr_sender)
// 			break
// 		case NatTraversalRequest:
// 			//TODO: Plus Tard
// 			break
// 		case NatTraversal:
// 			//TODO: Plus tard
// 			break

// 		case ErrorReply:
// 			handleErrorReply(message)
// 			break
// 		case HelloReply:
// 			handleHelloReply(message, nb_byte, addr_sender)
// 			break
// 		case Datum:
// 			handleDatum(message, nb_byte, addr_sender)
// 			break
// 		case NoDatum:
// 			handleNoDatum(message, nb_byte, addr_sender)
// 			break
// 		default:
// 			fmt.Printf("Unknown/Unexpected message type %d\n", t)
// 			break
// 		}
// 	}
// }

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
	// appelle au cli
}
