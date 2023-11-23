package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net"
)

// TODO : Il va falloir une fonction ici pour vérifier les signatures lorsqu'on les implémentera et l'appeler dans chaque handle

func handleError(message []byte) {
	len := getLength(message)
	fmt.Printf("Error :%s\n", message[7:7+len])
}

func handleHello(conn net.PacketConn, message []byte, nb_byte int, addr_sender net.Addr, name string) {
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

func handlePublicKey(conn net.PacketConn, message []byte, nb_byte int, addr_sender *net.UDPAddr) {
	_, err := sendPublicKeyReply(conn, addr_sender, getID(message))
	if err != nil {
		log.Fatal(err)
		return
	}
}

func handleRoot(conn net.PacketConn, message []byte, nb_byte int, addr_sender *net.UDPAddr) {
	_, err := sendRootReply(conn, addr_sender, getID(message))
	if err != nil {
		log.Fatal(err)
		return
	}
}

func handleGetDatum(conn net.PacketConn, message []byte, nb_byte int, addr_sender *net.UDPAddr) {
	hash := message[7 : 7+32]
	_, err := sendNoDatum(conn, addr_sender, [32]byte(hash), getID(message))
	if err != nil {
		log.Fatal(err)
		return
	}
}

func handleErrorReply(message []byte) {
	len := getLength(message)
	fmt.Printf("Error :%s\n", message[7:7+len])
}

func handleHelloReply(message []byte, nb_byte int, addr_sender *net.UDPAddr) {
	return
}

func handleDatum(message []byte, nb_byte int, addr_sender *net.UDPAddr) {
	hash := message[7 : 7+32]
	value := message[7+32 : 7+getLength(message)]
	check := sha256.Sum256(value)
	if check != [32]byte(hash) {
		fmt.Printf("Invalid checksum : The package will be ignored.\n Given Hash = %x, Expected Hash = %x\n", hash, check)
		return
	}

	// TODO : Comment couper Value pour avoir les hash et les autres trucs ?

	return
}

func handleNoDatum(message []byte, nb_byte int, addr_sender *net.UDPAddr) {
	hash := message[7 : 7+32]
	fmt.Printf("NoDatum for the hash : %x\n", hash)
	return
}
