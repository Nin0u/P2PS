package main

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"net/netip"
)

// Data type
const (
	CHUNK     byte = 0
	TREE      byte = 1
	DIRECTORY byte = 2
)

var debug_handler bool = false

/*
This function is called while handling Hello and HelloReply
It checks if the name is empty or not
It verifies the signature and adds or update the cache
*/
func checkHello(client *http.Client, message []byte, nb_byte int, addr_sender net.Addr, error_label string) {
	// Sender's name is not empty
	len := getLength(message)
	if len == 0 {
		if debug_handler {
			fmt.Println(error_label, "The sender has no name")
		}
		return
	}

	name_sender := string(message[7+4 : 7+len])
	data := message[:7+len]
	signature := message[7+len : nb_byte]
	index_peer := FindCachedPeerByName(name_sender)

	var key []byte
	// I don't know the peer
	if index_peer == -1 {
		if debug_handler {
			fmt.Println(error_label, "Didn't find peer in cache")
		}

		// Ask the server the peer's public key
		k, err := GetKey(client, name_sender)
		if err != nil {
			if debug_handler {
				fmt.Println(error_label, "Error while fetching key :", err)
			}
			return
		}
		key = k
	} else {
		key = make([]byte, 64)
		cache_peers.mutex.Lock()
		copy(key, cache_peers.list[index_peer].PublicKey[:])
		cache_peers.mutex.Unlock()
	}

	if VerifySignature(key, data, signature) {
		AddCachedPeer(BuildPeer(client, message, addr_sender, key))
	} else {
		if debug_handler {
			fmt.Println(error_label, "Invalid signature")
		}
		return
	}
}

func HandleHello(client *http.Client, conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, name string) {
	if debug_handler {
		fmt.Println("[HandleHello] Triggered")
	}

	// Check if we expected a Hello from the sender through a NATTraversal
	nat_sync_map.Unblock(addr_sender)

	// Checking signature and all
	checkHello(client, message, nb_byte, addr_sender, "[HandleHello]")

	_, err := sendHelloReply(conn, addr_sender, name, getID(message))
	if err != nil {
		if debug_handler {
			fmt.Println("[HandleHello] Error in sending HelloReply :", err)
		}
	}
}

func HandleHelloReply(client *http.Client, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleHelloReply] Triggered")
	}
	defer sync_map.Unblock(getID(message))

	checkHello(client, message, nb_byte, addr_sender, "[HandleHelloReply]")
}

func HandlePublicKey(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, index_peer int) {
	if debug_handler {
		fmt.Println("[HandlePublicKey] Triggered")
	}

	len := getLength(message)
	data := message[:7+len]
	signature := message[7+len : nb_byte]

	key := make([]byte, 64)
	cache_peers.mutex.Lock()
	copy(key, cache_peers.list[index_peer].PublicKey[:])
	cache_peers.mutex.Unlock()

	if VerifySignature(key, data, signature) {
		_, err := sendPublicKeyReply(conn, addr_sender, getID(message))
		if err != nil {
			if debug_handler {
				fmt.Println("[HandlePublicKey] Error while sending public key :", err)
			}
		}
	} else {
		if debug_handler {
			fmt.Println("[HandlePublicKey] Invalid signature with known peer")
		}
		return
	}
}

func HandleRoot(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, index_peer int) {
	if debug_handler {
		fmt.Println("[HandleRoot] Triggered")
	}

	len := getLength(message)
	data := message[:7+len]
	signature := message[7+len : nb_byte]

	key := make([]byte, 64)
	cache_peers.mutex.Lock()
	copy(key, cache_peers.list[index_peer].PublicKey[:])
	cache_peers.mutex.Unlock()

	if VerifySignature(key, data, signature) {
		_, err := sendRootReply(conn, addr_sender, getID(message))
		if err != nil {
			if debug_handler {
				fmt.Println("[HandleRoot] Error while sending root :", err)
			}
		}
	} else {
		if debug_handler {
			fmt.Println("[HandleRoot] Invalid signature with known peer")
		}
		return
	}
}

func HandleError(message []byte, error_label string) {
	if debug_handler {
		fmt.Println(error_label, "Triggered")
	}
	len := getLength(message)
	fmt.Println(error_label, ":", string(message[7:7+len]))
}

func HandleDatum(message []byte, nb_byte int, addr_sender net.Addr, conn net.PacketConn) {
	defer sync_map.Unblock(getID(message))

	if debug_handler {
		fmt.Println("[HandleDatum] Datum Received id :", getID(message))
	}

	hash := make([]byte, 32)
	value := make([]byte, getLength(message)-32)

	//! Copy is important, cause bug if we didn't
	copy(hash, message[7:7+32])
	copy(value, message[7+32:7+getLength(message)])
	check := sha256.Sum256(value)
	if check != [32]byte(hash) {
		if debug_handler {
			fmt.Printf("[HandleDatum] Invalid checksum : Given Hash = %x, Expected Hash = %x\n", hash, check)
			fmt.Println("[HandleDatum]", hash, value)
		}
		return
	}

	//Store in cache
	AddDatumCache([32]byte(hash), value)
}

func HandleNoDatum(message []byte, nb_byte int, addr_sender net.Addr) {
	defer sync_map.Unblock(getID(message))

	hash := message[7 : 7+32]
	fmt.Printf("NoDatum for the hash : %x\n", hash)

	AddDatumCache([32]byte(hash), nil)
}

func HandleGetDatum(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	//if debug_handler {
	fmt.Println("[HandleGetDatum] GetDatum Received id :", getID(message))
	//}

	len := getLength(message)
	if len != 32 {
		fmt.Println("[HandleGetDatum] Error on the length !")
		return
	}

	hash := message[7 : 7+32]

	map_export.Mutex.Lock()
	node, ok := map_export.Content[[32]byte(hash)]
	map_export.Mutex.Unlock()
	if !ok {
		//fmt.Println("No Datum :", hash)
		_, err := sendNoDatum(conn, addr_sender, [32]byte(hash), getID(message))
		if err != nil {
			if debug_handler {
				fmt.Println("[HandleGetDatum] Error while sending datum :", err)
			}
		}
		return
	}

	//fmt.Println("Node :", node)
	sendDatum(conn, addr_sender, [32]byte(hash), getID(message), node)
}

// ? Pour plus tard, Quand on fait un send hello, si il timeout
// ? On lance un NatTraversalRequest et va attendre un hello de l'addr qu'on a mis dedans !
// ? Au moment où on recoit un hello, il faut peut etre débloquer le gars en question !
// ? Au moment où on est débloqué, il faut relancer un hello !
// ! Le sendHello est bloquant, donc à faire dans un autre thread
// * Peut etre que lui doit attendre un hello apres le sendHello ? on verra...
func HandleNatTraversal(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	//Parse Addr
	addr_byte := message[7 : 7+getLength(message)]
	fmt.Println("nb_byte = ", nb_byte)
	if debug_handler {
		fmt.Println("[HandleNatTraversal]", addr_byte)
	}

	ip, ok := netip.AddrFromSlice(addr_byte[:len(addr_byte)-2])
	if !ok {
		fmt.Println("[HandleNatTarversal] Error addr from slice handleNat :", addr_byte)
		return
	}
	port := uint16(addr_byte[len(addr_byte)-2])<<8 + uint16(addr_byte[len(addr_byte)-1])
	addr_string := netip.AddrPortFrom(ip, port).String()
	fmt.Println("addr_string :", addr_string)
	addr_dest, err := net.ResolveUDPAddr("udp", addr_string)
	if err != nil {
		fmt.Println("[HandleNatTarversal] Error build addr :", addr_string, err.Error())
		return
	}

	sendHello(conn, addr_dest, username)
}

func HandleRootReply(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	defer sync_map.Unblock(getID(message))

	hash := message[7 : 7+32]
	map_export.Mutex.Lock()
	hash_c := rootExport.Hash
	map_export.Mutex.Unlock()

	if [32]byte(hash) != hash_c {
		fmt.Println("[HandleRootReply] Error not correct hash")
		return
	}
}
