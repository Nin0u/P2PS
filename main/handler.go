package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"
)

// Data type
const (
	CHUNK     byte = 0
	TREE      byte = 1
	DIRECTORY byte = 2
)

var debug_handler bool = false

func HandleError(message []byte) {
	if debug_handler {
		fmt.Println("[HandleError] Triggered")
	}
	len := getLength(message)
	fmt.Printf("Error :%s\n", message[7:7+len])
}

func HandleHello(client *http.Client, conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, name string) {
	if debug_handler {
		fmt.Println("[HandleHello] Triggered")
	}
	if nb_byte < 7+4+1 { // Sender's name is not empty
		if debug_handler {
			fmt.Println("[HandleHello] The sender has no name")
		}
		return
	}

	len := getLength(message)
	name_sender := string(message[7+4 : 7+len])
	data := message[:7+len]
	signature := message[7+len : nb_byte]
	index := FindCachedPeerByName(name_sender)

	// I don't know the peer
	if index == -1 {
		if debug_handler {
			fmt.Println("[HandleHello] Didn't find peer in cache")
		}

		// Ima ask the server the peer's public key
		key, err := GetKey(client, name_sender)
		if err != nil {
			if debug_handler {
				fmt.Println("[HandleHello] Error while fetching key :", err)
			}
			return
		}

		if VerifySignature(key, data, signature) {
			Add_cached_peer(BuildPeer(client, message, addr_sender))
		} else {
			if debug_handler {
				fmt.Println("[HandleHello] Invalid signature with fetched key")
			}
			return
		}

	} else { // I know the peer
		// I have the peer's verification key
		if VerifySignature(cache_peers.list[index].PublicKey[:], data, signature) {
			// Update his address and the timestamp
			AddAddrToPeer(&cache_peers.list[index], addr_sender)
			cache_peers.list[index].LastMessageTime = time.Now()
		} else {
			if debug_handler {
				fmt.Println("[HandleHello] Invalid signature with known peer")
			}
			return
		}
	}

	// sends HelloReply
	_, err := sendHelloReply(conn, addr_sender, name, getID(message))
	if err != nil {
		if debug_handler {
			fmt.Println("[HandleHello] Error in sending HelloReply :", err)
		}
	}
}

func HandlePublicKey(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandlePublicKey] Triggered")
	}

	// Make sure I known the peer (must have sent hello before)
	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		if debug_handler {
			fmt.Println("[HandlePublicKey] Peer not in cache. Message will be ignored")
		}
		return
	}

	len := getLength(message)
	data := message[:7+len]
	signature := message[7+len : nb_byte]

	if VerifySignature(cache_peers.list[index].PublicKey[:], data, signature) {
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

func HandleRoot(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleRoot] Triggered")
	}

	// Make sure I known the peer (must have sent hello before)
	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		if debug_handler {
			fmt.Println("[HandleRoot] Peer not in cache. Message will be ignored")
		}
		return
	}

	len := getLength(message)
	data := message[:7+len]
	signature := message[7+len : nb_byte]

	if VerifySignature(cache_peers.list[index].PublicKey[:], data, signature) {
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

func HandleErrorReply(message []byte) {
	if debug_handler {
		fmt.Println("[HandleErrorReply] Triggered")
	}
	len := getLength(message)
	fmt.Printf("ErrorReply :%s\n", message[7:7+len])
}

func unblock(message_id int32) {
	wg, b := GetSyncMap(message_id)
	if b {
		wg.Done()
	}
}

func HandlePublicKeyReply(message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandlePublicKeyReply] Triggered")
	}
	defer unblock(getID(message))

	// Make sure I known the peer (must have sent hello before)
	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		if debug_handler {
			fmt.Println("[HandlePublicKeyReply] Peer not in cache. Message will be ignored")
		}
		return
	}

	len := getLength(message)
	data := message[:7+len]
	key := message[7 : 7+len]
	signature := message[7+len : nb_byte]

	if len != 0 && !bytes.Equal(key, cache_peers.list[index].PublicKey[:]) {
		fmt.Printf("[HandlePublicKeyReply] Key given is different from what is stored in cache : %x != %x\n", key, cache_peers.list[index].PublicKey[:])
		return
	}

	if VerifySignature(cache_peers.list[index].PublicKey[:], data, signature) {
		fmt.Printf("PublicKey of %s is : %x\n", cache_peers.list[index].Name, key)
	} else {
		if debug_handler {
			fmt.Println("[HandlePublicKeyReply] Invalid signature with known peer")
		}
		return
	}
}

func HandleRootReply(message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleRootReply] Triggered")
	}
	defer unblock(getID(message))

	// Make sure I known the peer (must have sent hello before)
	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		if debug_handler {
			fmt.Println("[HandleRootReply] Peer not in cache. Message will be ignored")
		}
		return
	}

	len := getLength(message)
	data := message[:7+len]
	root := message[7 : 7+len]
	signature := message[7+len : nb_byte]

	if VerifySignature(cache_peers.list[index].PublicKey[:], data, signature) {
		fmt.Printf("Root of %s is : %x\n", cache_peers.list[index].Name, root)
	} else {
		if debug_handler {
			fmt.Println("[HandleRootReply] Invalid signature with known peer")
		}
		return
	}
}

func HandleHelloReply(client *http.Client, message []byte, nb_bytes int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleHelloReply] Triggered")
	}
	defer unblock(getID(message))
	len := getLength(message)
	name_sender := string(message[7+4 : 7+len])
	data := message[:7+len]
	signature := message[7+len : nb_bytes]
	index_peer := FindCachedPeerByName(name_sender)

	// I don't know the peer
	if index_peer == -1 {
		if debug_handler {
			fmt.Println("[HandleHelloReply] Didn't find peer. Creating a new one")
		}

		//First, ask the server the peer's public key
		key, err := GetKey(client, name_sender)
		if err != nil {
			if debug_handler {
				fmt.Println("[HandleHelloReply] Error while fetching key :", err)
			}
			return
		}

		//If signature is verified add the peer to the cache
		if VerifySignature(key, data, signature) {
			Add_cached_peer(BuildPeer(client, message, addr_sender))
		} else {
			if debug_handler {
				fmt.Println("[HandleHelloReply] Invalid signature with fetched key")
			}
			return
		}

	} else { // I know the peer
		//I have the peer's verification key
		if VerifySignature(cache_peers.list[index_peer].PublicKey[:], data, signature) {
			// Update his address and the timestamp
			AddAddrToPeer(&cache_peers.list[index_peer], addr_sender)
			cache_peers.list[index_peer].LastMessageTime = time.Now()
		} else {
			if debug_handler {
				fmt.Println("[HandleHelloReply] Invalid signature with known peer")
			}
			return
		}

	}
}

func HandleDatum(message []byte, nb_byte int, addr_sender net.Addr, conn net.PacketConn) {
	defer unblock(getID(message))

	if debug_handler {
		fmt.Println("[HandleDatum] Datum Received !")
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
	defer unblock(getID(message))

	hash := message[7 : 7+32]
	fmt.Printf("NoDatum for the hash : %x\n", hash)

	AddDatumCache([32]byte(hash), nil)
}

func HandleGetDatum(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	if debug_handler {
		fmt.Println("[HandleGetDatum] GetDatum Received !")
	}

	len := getLength(message)
	if len != 32 {
		fmt.Println("[HandleGetDatum] Error on the length !")
		return
	}

	hash := message[7 : 7+32]

	node, ok := map_export[[32]byte(hash)]
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
	addr_byte := message[7:nb_byte]
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
