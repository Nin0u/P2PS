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

func PeerClearer() {
	sleep_time, _ := time.ParseDuration("30s") // TODO : A adapter peut-être
	for {
		time.Sleep(sleep_time)
		current_time := time.Now()
		cache_peers.mutex.Lock()
		for i := 0; i < len(cache_peers.list); i++ {
			if current_time.Sub(cache_peers.list[i].LastMessageTime) > timeout_cache {
				removeCachedPeer(i)
			}
		}
		cache_peers.mutex.Unlock()
	}
}

func ConnKeeper(client *http.Client, conn net.PacketConn, addr []net.Addr) {
	sleep_time, _ := time.ParseDuration("30s")
	for {
		time.Sleep(sleep_time)
		for i := 0; i < len(addr); i++ {
			code, err := sendHello(conn, addr[i], username)
			if err != nil {
				fmt.Println("[ConnKeeper] Error while sending hello to ", addr[i], ":", err.Error())
				if code != -1 {
					addr = append(addr[:i], addr[i+1:]...)

					if len(addr) == 0 {
						fmt.Println("ERROR : No more addresses for the server. Closing ConnKeeper.")
					}
				}
			}
		}
	}
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
			index_peer, err := CheckHandShake(addr_sender)
			if err != nil {
				if debug {
					fmt.Println(err)
				}
				continue
			}

			// TODO : CheckHandShake and UpdatePeerLastMessage in the same Lock !

			UpdatePeerLastMessageTime(index_peer)

			switch t {
			case NoOp:

			case Error:
				HandleError(message, "[Error]")

			case PublicKey:
				HandlePublicKey(conn, message, nb_byte, addr_sender, index_peer)

			case Root:
				HandleRoot(conn, message, nb_byte, addr_sender, index_peer)

			case GetDatum:
				HandleGetDatum(conn, message, nb_byte, addr_sender)

			case NatTraversal:
				message_bis := make([]byte, nb_byte)
				copy(message_bis, message)
				go HandleNatTraversal(conn, message_bis, nb_byte, addr_sender)

			case ErrorReply:
				HandleError(message, "[ErrorReply]")

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
