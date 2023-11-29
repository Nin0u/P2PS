package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// Data type
const (
	CHUNK     byte = 0
	TREE      byte = 1
	DIRECTORY byte = 2
)

// TODO : Il va falloir une fonction ici pour vérifier les signatures lorsqu'on les implémentera et l'appeler dans chaque handle

func HandleError(message []byte) {
	len := getLength(message)
	fmt.Printf("Error :%s\n", message[7:7+len])
}

func HandleHello(client *http.Client, conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, name string) {
	if nb_byte < 7+4+1 { // Sender's name is not empty
		fmt.Println("The sender has no name, the message will be ignored.")
		return
	}

	len := getLength(message)
	name_sender := string(message[7+4 : 7+len])
	index := FindCachedPeerByName(name_sender)
	if index == -1 { // I don't know the peer
		if debug {
			fmt.Println("Didn't find peer")
		}

		// Ima ask the server the peer's public key
		key, err := GetKey(client, name_sender)
		if err != nil {
			fmt.Println(err)
			return
		}

		signature := message[7+len:]

		if VerifySignature(key, signature) {
			Add_cached_peer(Build_peer(message, addr_sender))
		} else {
			// TODO : Sinon Send Error ?
			fmt.Println("Unvalide signature")
			return
		}

	} else { // I know the peer
		signature := message[7+len:]
		// I have the peer's verification key
		if VerifySignature(cache_peers.list[index].PublicKey[:], signature) {
			// Update his address and the timestamp
			cache_peers.list[index].Addr = addr_sender
			cache_peers.list[index].LastMessageTime = time.Now()
		} else {
			// TODO : Sinon Send Error ?
			fmt.Println("Unvalide signature")
			return
		}

	}

	// sends HelloReply
	_, err := sendHelloReply(conn, addr_sender, name, getID(message))
	if err != nil {
		log.Fatal(err)
		return
	}
}

func HandlePublicKey(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	_, err := sendPublicKeyReply(conn, addr_sender, getID(message))
	if err != nil {
		if debug {
			log.Fatal(err)
		}

		return
	}
}

func HandleRoot(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	_, err := sendRootReply(conn, addr_sender, getID(message))
	if err != nil {
		log.Fatal(err)
		return
	}
}

func HandleGetDatum(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {

	hash := message[7 : 7+32]
	_, err := sendNoDatum(conn, addr_sender, [32]byte(hash), getID(message))
	if err != nil {
		log.Fatal(err)
		return
	}
}

func HandleErrorReply(message []byte) {
	len := getLength(message)
	fmt.Printf("Error :%s\n", message[7:7+len])
}

func HandleHelloReply(client *http.Client, message []byte, nb_byte int, addr_sender net.Addr) {
	// Find if a request with the same id is in reemit list
	id := getID(message)
	index_reemit := FindReemitById(id)
	if index_reemit == -1 {
		fmt.Printf("No message with id %d in reemit_list\n", id)
		return
	}

	if debug {
		fmt.Println("Reemit message found")
		fmt.Println(reemit_list.list)
	}

	// Check if the message has type Hello
	if reemit_list.list[index_reemit].Type != Hello {
		fmt.Printf("Expected message to have Hello = 2 type in reemit_list, found %d.\n", reemit_list.list[index_reemit].Type)
		return
	}

	if debug {
		fmt.Println("Reemit message has type Hello as expected")
	}
	// We have to check signature before removing the message that was reemited

	len := getLength(message)
	name_sender := string(message[7+4 : 7+len])
	index_peer := FindCachedPeerByName(name_sender)
	if index_peer == -1 {
		// I don't know the peer
		fmt.Println("Didn't find peer. Creating a new one")

		// First, ask the server the peer's public key
		key, err := GetKey(client, name_sender)
		if err != nil {
			fmt.Println(err)
			return
		}

		signature := message[7+len:]

		// Then verify the signature
		if VerifySignature(key, signature) {
			// Add the peer to the cache
			Add_cached_peer(Build_peer(message, addr_sender))
			// Remove the reemited message
			RemoveReemit(index_reemit)
		} else {
			// TODO : Sinon Send Error ?
			fmt.Println("Invalid signature")
			return
		}

	} else { // I know the peer
		signature := message[7+len:]
		// I have the peer's verification key
		if VerifySignature(cache_peers.list[index_peer].PublicKey[:], signature) {
			// Update his address and the timestamp
			cache_peers.list[index_peer].Addr = addr_sender
			cache_peers.list[index_peer].LastMessageTime = time.Now()
			// Remove the reemited message
			RemoveReemit(index_reemit)
		} else {
			// TODO : Sinon Send Error ?
			fmt.Println("Invalid signature")
			return
		}

	}
}

func HandleDatum(message []byte, nb_byte int, addr_sender net.Addr, conn net.PacketConn) {
	err := CheckHandShake(addr_sender)
	if err != nil {
		if debug {
			fmt.Println(err)
		}
		return
	}

	fmt.Println("Datum Received !")

	hash := message[7 : 7+32]
	value := message[7+32 : 7+getLength(message)]
	check := sha256.Sum256(value)
	if check != [32]byte(hash) {
		fmt.Printf("Invalid checksum : The package will be ignored.\n Given Hash = %x, Expected Hash = %x\n", hash, check)
		return
	}

	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		fmt.Println("Don't find Peer !")
		return
	}

	reqDatum.mutex.Lock()
	if reqDatum.list[0].P != cache_peers.list[index] {
		fmt.Println("Unexpected Peer !")
		reqDatum.mutex.Unlock()
		return
	}
	reqDatum.mutex.Unlock()

	if reqDatum.list[0].TypeReq == 0 {
		download_list([32]byte(hash), value[0], value[1:], conn)
	} else if reqDatum.list[0].TypeReq == 1 {
		download_dl([32]byte(hash), value[0], value[1:], conn)
	}
}

func HandleNoDatum(message []byte, nb_byte int, addr_sender net.Addr) {
	err := CheckHandShake(addr_sender)
	if err != nil {
		if debug {
			fmt.Println(err)
		}

		return
	}

	hash := message[7 : 7+32]
	fmt.Printf("NoDatum for the hash : %x\n", hash)
	os.Exit(1)
}

// TODO : à déplacer + implémenter
func VerifySignature(key []byte, signature []byte) bool {
	// Skip checking if no key found
	if key == nil {
		return true
	}
	return true
}
