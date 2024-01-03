package main

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"net/netip"

	"github.com/fatih/color"
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
func checkHello(client *http.Client, conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, error_label string) {
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
	cache_peers.mutex.Lock()
	index_peer := FindCachedPeerByName(name_sender)

	var key []byte
	// I don't know the peer
	if index_peer == -1 {
		cache_peers.mutex.Unlock()
		if debug_handler {
			color.Magenta("%s Didn't find peer in cache\n", error_label)
		}

		// Ask the server the peer's public key
		k, err := GetKey(client, name_sender)
		if err != nil {
			if debug_handler {
				color.Magenta("%s Error while fetching key : %s\n", error_label, err.Error())
			}
			return
		}
		key = k
	} else {
		key = make([]byte, 64)
		copy(key, cache_peers.list[index_peer].PublicKey[:])
		cache_peers.mutex.Unlock()
	}

	if VerifySignature(key, data, signature) {
		p := BuildPeer(client, message, addr_sender, key)
		modified := AddCachedPeer(p)

		if modified {
			cache_peers.mutex.Lock()
			index := FindCachedPeerByName(p.Name)
			if index == -1 {
				color.Red("[CheckHello] Peer venished before RTO computation")
				cache_peers.mutex.Unlock()
				return
			}

			p = cache_peers.list[index]
			cache_peers.mutex.Unlock()

			rto := ComputeRTO(conn, addr_sender)
			cache_peers.mutex.Lock()
			p.AddRTO(addr_sender, rto)
			PrintCachedPeers()
			cache_peers.mutex.Unlock()
		}
	} else {
		if debug_handler {
			color.Magenta("%s Invalid signature\n", error_label)
		}
		return
	}
}

func HandleHello(client *http.Client, conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleHello] Triggered")
	}

	// Check if we expected a Hello from the sender through a NATTraversal
	nat_sync_map.Unblock(addr_sender)

	// Checking signature and all
	checkHello(client, conn, message, nb_byte, addr_sender, "[HandleHello]")

	_, err := sendHelloReply(conn, addr_sender, getID(message))
	if err != nil {
		if debug_handler {
			color.Red("[HandleHello] Error in sending HelloReply : %s\n", err.Error())
		}
	}
}

func HandleHelloReply(client *http.Client, conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleHelloReply] Triggered")
	}
	defer sync_map.Unblock(getID(message))

	checkHello(client, conn, message, nb_byte, addr_sender, "[HandleHelloReply]")
}

func HandlePublicKey(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandlePublicKey] Triggered")
	}

	// Length Checking
	l := getLength(message)
	if l != 64 {
		color.Cyan("[HandlePublicKey] Invalid Length : Expected 64, got %d\n", l)
		return
	}
	data := message[:7+l]
	signature := message[7+l : nb_byte]

	key := make([]byte, 64)
	cache_peers.mutex.Lock()
	index_peer, _ := FindCachedPeerByAddr(addr_sender)
	if index_peer == -1 {
		color.Magenta("[HandlePublicKey] Unknown Peer\n")
		cache_peers.mutex.Unlock()
		return
	}
	copy(key, cache_peers.list[index_peer].PublicKey[:])
	cache_peers.mutex.Unlock()

	if VerifySignature(key, data, signature) {
		_, err := sendPublicKeyReply(conn, addr_sender, getID(message))
		if err != nil {
			if debug_handler {
				color.Red("[HandlePublicKey] Error while sending public key : %s\n", err.Error())
			}
		}
	} else {
		if debug_handler {
			color.Magenta("[HandlePublicKey] Invalid signature with known peer\n")
		}
		return
	}
}

func HandleRoot(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleRoot] Triggered")
	}

	// Length Checking
	l := getLength(message)
	if l != 32 {
		color.Cyan("[HandleRoot] Invalid Length : Expected 32, got %d\n", l)
		return
	}
	data := message[:7+l]
	signature := message[7+l : nb_byte]

	key := make([]byte, 64)
	cache_peers.mutex.Lock()
	index_peer, _ := FindCachedPeerByAddr(addr_sender)
	if index_peer == -1 {
		color.Magenta("[HandleRoot] Unknown Peer\n")
		cache_peers.mutex.Unlock()
		return
	}
	copy(key, cache_peers.list[index_peer].PublicKey[:])
	cache_peers.mutex.Unlock()

	if VerifySignature(key, data, signature) {
		_, err := sendRootReply(conn, addr_sender, getID(message))
		if err != nil {
			if debug_handler {
				color.Red("[HandleRoot] Error while sending root : %s\n", err.Error())
			}
		}
	} else {
		if debug_handler {
			color.Magenta("[HandleRoot] Invalid signature with known peer")
		}
		return
	}
}

func HandleError(message []byte, error_label string) {
	if debug_handler {
		fmt.Println(error_label, "Triggered")
	}
	len := getLength(message)
	color.Cyan("%s : %s\n", error_label, string(message[7:7+len]))
}

func HandleDatum(message []byte, nb_byte int, addr_sender net.Addr, conn net.PacketConn) {
	defer sync_map.Unblock(getID(message))

	if debug_handler {
		fmt.Println("[HandleDatum] Datum Received id :", getID(message))
	}

	// Length Checking
	l := getLength(message)
	if int(l)+7 > nb_byte {
		color.Cyan("[HandleDatum] Invalid Length : Expected at least %d, got %d\n", nb_byte, l+7)
		return
	}

	hash := make([]byte, 32)
	value := make([]byte, getLength(message)-32)

	//! Copy is important, cause bug if we didn't
	copy(hash, message[7:7+32])
	copy(value, message[7+32:7+getLength(message)])
	check := sha256.Sum256(value)
	if check != [32]byte(hash) {
		if debug_handler {
			color.Magenta("[HandleDatum] Invalid checksum : Given Hash = %x, Expected Hash = %x\n", hash, check)
		}
		return
	}

	//Store in cache
	AddDatumCache([32]byte(hash), value)
}

func HandleNoDatum(message []byte, nb_byte int, addr_sender net.Addr) {
	defer sync_map.Unblock(getID(message))

	// Length Checking
	l := getLength(message)
	if l != 32 {
		color.Cyan("[HandleNoDatum] Invalid Length : Expected 32, got %d\n", l)
		return
	}

	hash := message[7 : 7+l]
	if debug_handler {
		color.Magenta("[handleNoDatum] NoDatum for the hash : %x\n", hash)
	}

	AddDatumCache([32]byte(hash), nil)
}

func HandleGetDatum(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleGetDatum] GetDatum Received id :", getID(message))
	}

	l := getLength(message)
	if l != 32 {
		color.Magenta("[HandleGetDatum] Invalid Length : Expected 32, got %d\n", l)
		return
	}

	hash := message[7 : 7+32]

	map_export.Mutex.Lock()
	node, ok := map_export.Content[[32]byte(hash)]
	map_export.Mutex.Unlock()
	if !ok {
		_, err := sendNoDatum(conn, addr_sender, [32]byte(hash), getID(message))
		if err != nil {
			if debug_handler {
				color.Red("[HandleGetDatum] Error while sending datum : %s\n", err.Error())
			}
		}
		return
	}

	sendDatum(conn, addr_sender, [32]byte(hash), getID(message), node)
}

func HandleNatTraversal(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	l := getLength(message)
	if int(l)+7 > nb_byte {
		color.Magenta("[HandleGetDatum] Invalid Length : Expected %d, got %d\n", nb_byte, l)
		return
	}

	//Parse Addr
	addr_byte := message[7 : 7+l]
	if debug_handler {
		fmt.Println("[HandleNatTraversal] addr_byte =", addr_byte)
	}

	ip, ok := netip.AddrFromSlice(addr_byte[:len(addr_byte)-2])
	if !ok {
		color.Red("[HandleNatTarversal] Error addr from slice handleNat : %x\n", addr_byte)
		return
	}
	port := uint16(addr_byte[len(addr_byte)-2])<<8 + uint16(addr_byte[len(addr_byte)-1])
	addr_string := netip.AddrPortFrom(ip, port).String()
	addr_dest, err := net.ResolveUDPAddr("udp", addr_string)
	if err != nil {
		color.Red("[HandleNatTarversal] Error build addr %s : %s\n", addr_string, err.Error())
		return
	}

	sendHello(conn, addr_dest, false)
}

func HandleRootReply(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	defer sync_map.Unblock(getID(message))

	// Length Checking
	l := getLength(message)
	if l != 32 {
		color.Cyan("[HandleRootReply] Invalid Length : Expected 32, got %d\n", l)
		return
	}

	// Signature Checking
	cache_peers.mutex.Lock()
	index_peer, _ := FindCachedPeerByAddr(addr_sender)

	var key []byte
	// I don't know the peer
	if index_peer == -1 {
		cache_peers.mutex.Unlock()
		if debug_handler {
			color.Magenta("[HandleRootReply] Didn't find peer in cache\n")
		}
		return
	} else {
		key = make([]byte, 64)
		copy(key, cache_peers.list[index_peer].PublicKey[:])
		cache_peers.mutex.Unlock()
	}

	data := message[:7+l]
	signature := message[7+l : nb_byte]
	if VerifySignature(key, data, signature) {
		// There's nothing to do here
		return
	} else {
		if debug_handler {
			color.Magenta("[HandleRootReply] Invalid signature\n")
		}
		return
	}
}
