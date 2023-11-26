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

var rest_commands = []string{"list", "addr", "key", "root"}
var p2p_commands = []string{"hello", "data", "cache"}
var desc_rest_commands = []string{
	"                        list all peers",
	"	<peername>           list addresses",
	" 	<peername>           get public key",
	" 	<peername>           get root",
}
var desc_p2p_commands = []string{
	" 	<addr>               send hello",
	" 	<addr> <hash>        get the real data of the hash",
}

func title_print() {
	fmt.Println(" __        __   _                            _      ")
	fmt.Println(" \\ \\      / /__| | ___ ___  _ __ ___   ___  | |_ ___  ")
	fmt.Println("  \\ \\ /\\ / / _ \\ |/ __/ _ \\| '_ ` _ \\ / _ \\ | __/ _ \\ ")
	fmt.Println("   \\ V  V /  __/ | (_| (_) | | | | | |  __/ | || (_) |")
	fmt.Println("  __\\_/\\_/_\\___|_|\\___\\___/|_| |_| |_|\\___|  \\__\\___/")
	fmt.Println(" |  _ \\___ \\|  _ \\/ ___|| |__   ")
	fmt.Println(" | |_) |__) | |_) \\___ \\| '_ \\ / _` | '__/ _ \\ ")
	fmt.Println(" |  __// __/|  __/ ___) | | | | (_| | | |  __/")
	fmt.Println(" |_|  |_____|_|   |____/|_| |_|\\__,_|_|  \\___|")
}

func print_help() {
	fmt.Println("--------------- REST Commands ---------------")
	for i := 0; i < len(rest_commands); i++ {
		fmt.Printf("%s %s\n", rest_commands[i], desc_rest_commands[i])
	}
	fmt.Println("\n--------------- P2P Commands ---------------")
	for i := 0; i < len(p2p_commands); i++ {
		fmt.Printf("%s %s\n", p2p_commands[i], desc_p2p_commands[i])
	}
	fmt.Println("\n(Type help to display this list)")
}

func cli(client *http.Client, conn net.PacketConn) {
	sc := bufio.NewScanner(os.Stdin)

	fmt.Print("Please enter your username : ")
	if sc.Scan() {
		username = sc.Text()
	}

	title_print()
	fmt.Println()
	print_help()
	fmt.Println()

	// Main loop
	fmt.Fprint(os.Stdout, "$> ")
	for sc.Scan() {
		content := sc.Text()
		words := strings.Split(content, " ")
		switch words[0] {
		case "list":
			handleList(client)

		case "addr":
			handleListAddr(client, words)

		case "hello":
			handleSendHello(conn, words)

		case "root":
			handleGetRoot(client, words)

		case "data":
			handleGetData(conn, words)

		case "help":
			print_help()

		default:
			fmt.Println("Unknown command ;-;")

		}
		fmt.Fprint(os.Stdout, "$> ")

	}
}

func handleList(client *http.Client) {
	list, err := GetPeers(client)
	if err != nil {
		log.Fatal("Error getPeers http :", err)
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

	list, err := GetAddresses(client, words[1])
	if err != nil {
		log.Fatal("Error getAddr http", err)
	}

	fmt.Println("Here are the addresses of ", words[1])

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

	hash, err := GetRoot(client, words[1])

	if err != nil {
		log.Fatal("Error getRoot http", err)
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
