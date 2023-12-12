package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

var debug bool = false
var username string = ""

// Keys are Id and value are sync.WaitGroup
type SyncMap struct {
	content map[int32]*sync.WaitGroup
	mutex   sync.Mutex
}

var sync_map SyncMap = SyncMap{content: make(map[int32]*sync.WaitGroup)}

func SetSyncMap(id int32, wg *sync.WaitGroup) {
	sync_map.mutex.Lock()
	sync_map.content[id] = wg
	sync_map.mutex.Unlock()
}

func GetSyncMap(id int32) (*sync.WaitGroup, bool) {
	sync_map.mutex.Lock()
	wg, ok := sync_map.content[id]
	sync_map.mutex.Unlock()
	return wg, ok
}

func DeleteSyncMap(id int32) {
	sync_map.mutex.Lock()
	delete(sync_map.content, id)
	sync_map.mutex.Unlock()
}

func Recv(client *http.Client, conn net.PacketConn) {
	message := make([]byte, 65535+7) //TODO: + une signature

	for {
		nb_byte, addr_sender, err := conn.ReadFrom(message)
		if err != nil {
			fmt.Print("rcv error :")
			fmt.Println(err)
			continue
		}

		if debug {
			fmt.Printf("[Recv] Received message : %x\n", message[:nb_byte])
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

			case NatTraversal:
				HandleNatTraversal(conn, message, nb_byte, addr_sender)

			case ErrorReply:
				HandleErrorReply(message)

			case PublicKeyReply:
				HandlePublicKeyReply(message, nb_byte, addr_sender)

			case RootReply:
				HandleRootReply(message, nb_byte, addr_sender)

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
	export("./Test")

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--debug" {
			debug = true
			debug_peer = true
			debug_rest = true
			debug_handler = true
			debug_message = true
			debug_signature = true
		}
	}
	GenKeys()

	transport := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(conn.LocalAddr().String())

	go Recv(client, conn)
	cli(client, conn)
}
