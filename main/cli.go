package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

var list_command = []string{"list", "addr", "hello", "root", "data"}
var desc_command = []string{
	"                   list all peers",
	" <peers>           list addresses",
	"<addr>            send hello",
	" <peers>           get root",
	" <addr> <hash>     get the real data of the hash",
}

func cli(client *http.Client, conn net.PacketConn) {
	sc := bufio.NewScanner(os.Stdin)

	fmt.Printf("Bienvenue dans la super interface ! :)\n")
	fmt.Println()
	for i := 0; i < len(list_command); i++ {
		fmt.Printf("%s %s\n", list_command[i], desc_command[i])
	}
	fmt.Println()

	// Main loop
	fmt.Fprint(os.Stdout, "$> ")
	for sc.Scan() {
		content := sc.Text()
		words := strings.Split(content, " ")
		switch words[0] {
		case "list":
			handleList(client)
			break
		case "addr":
			handleListAddr(client, words)
			break
		case "hello":
			handleSendHello(conn, words)
			break
		case "root":
			handleGetRoot(client, words)
			break
		case "data":
			handleGetData(conn, words)
			break
		default:
			fmt.Println("Unknown command ;-;")
			break
		}
		fmt.Fprint(os.Stdout, "$> ")

	}
}

func handleList(client *http.Client) {
	list, err := getPeers(client)
	if err != nil {
		log.Fatal("Erreur getPeers http :", err)
	}

	fmt.Println("Voici la liste des pairs :")
	for i := 0; i < len(list); i++ {
		fmt.Println(list[i])
	}
}

func handleListAddr(client *http.Client, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	list, err := getAddresses(client, words[1])
	if err != nil {
		log.Fatal("Erreur getAddr http", err)
	}

	fmt.Println("Voici la liste des addresses de ", words[1])

	for i := 0; i < len(list); i++ {
		fmt.Println(list[i])
	}
}

// TODO: Gerer une liste de pair au lieu de faire comme ça
func handleSendHello(conn net.PacketConn, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	addr, err := net.ResolveUDPAddr("udp", words[1])
	if err != nil {
		log.Fatal("Error resolve addr", err)
	}
	_, err = sendHello(conn, addr, username)

	if err != nil {
		log.Fatal("Error send hello", err)
	}
}

func handleGetRoot(client *http.Client, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	hash, err := getRoot(client, words[1])

	if err != nil {
		log.Fatal("Erreur getRoot http", err)
	}

	fmt.Printf("%x\n", string(hash))
}

func handleGetData(conn net.PacketConn, words []string) {
	if len(words) != 3 {
		fmt.Println("Wrong number of argument !")
		return
	}

	addr, err := net.ResolveUDPAddr("udp", words[1])
	if err != nil {
		log.Fatal("Error resolve addr", err)
	}

	_, err = sendGetDatum(conn, addr, [32]byte([]byte(words[2][:32])))

	if err != nil {
		log.Fatal("Error send hello", err)
	}
}
