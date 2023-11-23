package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net"
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

func HandleHello(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, name string) {
	if nb_byte < 7+4+1 { // On veut que le sender ait un nom non vide quand même...
		fmt.Println("The sender has no name, the message will be ignored.")
		return
	}

	// On renvoie au sender un HelloReply
	_, err := sendHelloReply(conn, addr_sender, name, getID(message))
	if err != nil {
		log.Fatal(err)
		return
	}
}

func HandlePublicKey(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr) {
	_, err := sendPublicKeyReply(conn, addr_sender, getID(message))
	if err != nil {
		log.Fatal(err)
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

func HandleHelloReply(message []byte, nb_byte int, addr_sender net.Addr) {}

func HandleDatum(message []byte, nb_byte int, addr_sender net.Addr) {
	hash := message[7 : 7+32]
	value := message[7+32 : 7+getLength(message)]
	check := sha256.Sum256(value)
	if check != [32]byte(hash) {
		fmt.Printf("Invalid checksum : The package will be ignored.\n Given Hash = %x, Expected Hash = %x\n", hash, check)
		return
	}

	switch value[0] {
	case CHUNK:
		fmt.Printf("Chunk recieved : %x\n", value[1:])
	case TREE:
		fmt.Println("Tree recieved. Children's hashes are : ")
		for i := 1; i < len(value); i += 32 {
			fmt.Printf("- %x\n", value[i:i+32])
		}
	case DIRECTORY:
		fmt.Println("Directory recieved. Contents' hashes are : ")
		for i := 1; i < len(value); i += 64 {
			fmt.Printf("- Name = %x, Hash = %x \n", value[i:i+32], value[i+32:i+63])
		}
	default:
		fmt.Printf("Undefined data type : %d\n", value[0])
	}
}

func HandleNoDatum(message []byte, nb_byte int, addr_sender net.Addr) {
	hash := message[7 : 7+32]
	fmt.Printf("NoDatum for the hash : %x\n", hash)
}
