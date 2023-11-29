package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

var rest_commands = []string{"list", "addr", "key", "root"}
var p2p_commands = []string{"hello", "data", "data_dl"}
var desc_rest_commands = []string{
	"                        list all peers",
	"	<peername>           list addresses",
	" 	<peername>           get public key",
	" 	<peername>           get root",
}
var desc_p2p_commands = []string{
	" 	<addr>               send hello",
	" 	<peername>        	 list data of the peer",
	"   <peername>           download data of the peer",
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
			handleGetData(client, conn, words)

		case "data_dl":
			handleGetDataDL(client, conn, words)

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
		fmt.Println("Error getPeers http :", err.Error())
		return
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
		fmt.Println("Error getAddr http :", err.Error())
		return
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
		fmt.Println("Error resolve addr", err.Error())
		return
	}
	_, err = sendHello(conn, addr, username)

	if err != nil {
		fmt.Println("Error send hello :", err.Error())
		return
	}
}

func handleGetRoot(client *http.Client, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	hash, err := GetRoot(client, words[1])

	if err != nil {
		fmt.Println("Error getRoot http : ", err.Error())
		return
	}

	fmt.Printf("%x\n", string(hash))
}

func handleGetData(client *http.Client, conn net.PacketConn, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	//Get the hash of the root
	hash, err := GetRoot(client, words[1])
	if err != nil {
		fmt.Print("Error getRoot : ", err.Error())
		return
	}

	//Get the peers and need to be register in the cache
	index_peer := FindCachedPeerByName(words[1])
	if index_peer == -1 {
		fmt.Println("This peer is not cached ! Please send hello first")
		return
	}

	fmt.Println("COUCOU")

	//Add the root in the tree
	//TODO: Maybe uneccessary if we already have it ? Optimization here !
	cache_peers.list[index_peer].Root = add_node(nil, make([]string, 0), "", [32]byte(hash), DIRECTORY)

	//Add the request datum to the list of reqDatum
	req := buildRequestDatum(cache_peers.list[index_peer], "", [32]byte(hash), 0)
	reqDatum.mutex.Lock()
	reqDatum.list = append(reqDatum.list, req)
	reqDatum.mutex.Unlock()

	//Send the first getDatum()
	_, err = sendGetDatum(conn, cache_peers.list[index_peer].Addr, [32]byte(hash))
	if err != nil {
		fmt.Println("Error sendGetDatum : ", err.Error())
		clearRequestDatum()
		return
	}

	fmt.Println("GetDatum launched !")
}

func handleGetDataDL(client *http.Client, conn net.PacketConn, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	//Get the hash of the root
	hash, err := GetRoot(client, words[1])
	if err != nil {
		fmt.Print("Error getRoot : ", err.Error())
		return
	}

	//Get the peers and need to be register in the cache
	index_peer := FindCachedPeerByName(words[1])
	if index_peer == -1 {
		fmt.Println("This peer is not cached ! Please send hello first")
		return
	}

	//Add the root in the tree
	//TODO: Maybe uneccessary if we already have it ? Optimization here !
	//cache_peers.list[index_peer].Root = add_node(nil, make([]string, 0), "", [32]byte(hash), DIRECTORY)

	//Add the request datum to the list of reqDatum
	req := buildRequestDatum(cache_peers.list[index_peer], cache_peers.list[index_peer].Name, [32]byte(hash), 1)
	reqDatum.mutex.Lock()
	reqDatum.list = append(reqDatum.list, req)
	reqDatum.mutex.Unlock()

	//Send the first getDatum()
	_, err = sendGetDatum(conn, cache_peers.list[index_peer].Addr, [32]byte(hash))
	if err != nil {
		fmt.Println("Error sendGetDatum : ", err.Error())
		clearRequestDatum()
		return
	}

	fmt.Println("GetDatum launched !")
}
