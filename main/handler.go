package main

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Data type
const (
	CHUNK     byte = 0
	TREE      byte = 1
	DIRECTORY byte = 2
)

var debug_handler bool = false

// TODO : Il va falloir une fonction ici pour vérifier les signatures lorsqu'on les implémentera et l'appeler dans chaque handle

func HandleError(message []byte) {
	len := getLength(message)
	fmt.Printf("Error :%s\n", message[7:7+len])
}

func HandleHello(client *http.Client, conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, name string) {
	if nb_byte < 7+4+1 { // Sender's name is not empty
		if debug_handler {
			fmt.Println("[HandleHello] The sender has no name")
		}
		return
	}

	len := getLength(message)
	name_sender := string(message[7+4 : 7+len])
	index := FindCachedPeerByName(name_sender)
	if index == -1 { // I don't know the peer
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

		data := message[:7+len]
		signature := message[7+len:]

		if VerifySignature(key, data, signature) {
			Add_cached_peer(BuildPeer(client, message, addr_sender))
		} else {
			// TODO : Sinon Send Error ?
			if debug_handler {
				fmt.Println("[HandleHello] Invalid signature with fetched key")
			}
			return
		}

	} else { // I know the peer
		data := message[:7+len]
		signature := message[7+len:]
		// I have the peer's verification key
		if VerifySignature(cache_peers.list[index].PublicKey[:], data, signature) {
			// Update his address and the timestamp
			AddAddrToPeer(&cache_peers.list[index], addr_sender)
			cache_peers.list[index].LastMessageTime = time.Now()
		} else {
			// TODO : Sinon Send Error ?
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
	_, err := sendPublicKeyReply(conn, addr_sender, getID(message))
	if err != nil {
		if debug_handler {
			fmt.Println("[HandlePublicKey] Error while sending public key :", err)
		}
	}
}

func HandleRoot(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	_, err := sendRootReply(conn, addr_sender, getID(message))
	if err != nil {
		if debug_handler {
			fmt.Println("[HandleRoot] Error while sending root :", err)
		}
	}
}

func HandleErrorReply(message []byte) {
	len := getLength(message)
	fmt.Printf("Error :%s\n", message[7:7+len])
}

func unblock(message_id int32) {
	wg, b := GetSyncMap(message_id)
	if b {
		wg.Done()
	}
}

func HandleHelloReply(client *http.Client, message []byte, nb_byte int, addr_sender net.Addr) {
	defer unblock(getID(message))
	len := getLength(message)
	name_sender := string(message[7+4 : 7+len])
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

		data := message[:7+len]
		signature := message[7+len:]

		//If signature is verified add the peer to the cache
		if VerifySignature(key, data, signature) {
			Add_cached_peer(BuildPeer(client, message, addr_sender))
		} else {
			// TODO : Sinon Send Error ?
			if debug_handler {
				fmt.Println("[HandleHelloReply] Invalid signature with fetched key")
			}
			return
		}

	} else { // I know the peer
		data := message[:7+len]
		signature := message[7+len:]
		//I have the peer's verification key
		if VerifySignature(cache_peers.list[index_peer].PublicKey[:], data, signature) {
			// Update his address and the timestamp
			AddAddrToPeer(&cache_peers.list[index_peer], addr_sender)
			cache_peers.list[index_peer].LastMessageTime = time.Now()
		} else {
			// TODO : Sinon Send Error ?
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
