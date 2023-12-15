package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/eiannone/keyboard"
)

type Command struct {
	CommandName string
	Argument    string
	HelpText    string
}

var rest_commands = []Command{
	{CommandName: "list      ", Argument: "                   ", HelpText: "list all peers"},
	{CommandName: "addr      ", Argument: "<peername>         ", HelpText: "list addresses of the peer"},
	{CommandName: "get_key   ", Argument: "<peername>         ", HelpText: "get the peer's public key"},
	{CommandName: "get_root  ", Argument: "<peername>         ", HelpText: "get the peer's root"},
}

var p2p_commands = []Command{
	{CommandName: "hello     ", Argument: "<addr>             ", HelpText: "sends Hello to the given address"},
	{CommandName: "public_key", Argument: "<addr>             ", HelpText: "sends Publickey to the given address"},
	{CommandName: "root      ", Argument: "<addr>             ", HelpText: "sends Root to the given address"},
	{CommandName: "nat       ", Argument: "<peername>         ", HelpText: "nat traversal"},
	{CommandName: "data      ", Argument: "<peername>         ", HelpText: "list data of the peer"},
	{CommandName: "data_dl   ", Argument: "<peername> [<path>]", HelpText: "download data of the peer. If a path is given then it will download all the data from this path."},
}

// Command History
const history_max_size = 5

var history_cursor int = 0
var command_history = make([]string, 0)

const prompt string = "$> "

var input_cursor int = 0

var server_name_peer string = "jch.irif.fr"

func AddCommandHistory(content string) {
	command_history = append(command_history, content)
	if len(command_history) >= history_max_size {
		command_history = command_history[len(command_history)-history_max_size:]
	}

	history_cursor = len(command_history)
}

func moveHistoryCursor(up bool, s *string) {
	if up && history_cursor > 0 {
		history_cursor--
	} else if !up && history_cursor <= len(command_history)-1 {
		history_cursor++
	}

	getHistoryCommand(s)
}

func getHistoryCommand(s *string) {
	blank := ""
	for i := 0; i < len(*s); i++ {
		blank += " "
	}

	if history_cursor == len(command_history) {
		*s = ""
		fmt.Printf("\r%s%s\r%s", prompt, blank, prompt)
	} else {
		*s = command_history[history_cursor]
		fmt.Printf("\r%s%s\r%s%s", prompt, blank, prompt, *s)
	}
}

func moveInputCursor(left bool, s *string) {
	if len(*s) > 0 {
		if left && input_cursor >= 1 {
			input_cursor--
		} else if !left && input_cursor < len(*s) {
			input_cursor++
		}
	}

	if input_cursor > len(*s) {
		fmt.Printf("\r%s%s", prompt, *s)
	} else {
		fmt.Printf("\r%s%s", prompt, (*s)[:input_cursor])
	}
}

func addCharToCommand(c string, s *string) {
	if input_cursor < len(*s) {
		*s = (*s)[:input_cursor] + c + (*s)[input_cursor:]
		input_cursor++
		fmt.Printf("\r%s%s\r%s%s", prompt, *s, prompt, (*s)[:input_cursor])
	} else {
		(*s) += c
		input_cursor = len(*s)
		fmt.Printf("\r%s%s", prompt, *s)
	}
}

func title_print() {
	fmt.Println(" __        __   _                            _      ")
	fmt.Println(" \\ \\      / /__| | ___ ___  _ __ ___   ___  | |_ ___  ")
	fmt.Println("  \\ \\ /\\ / / _ \\ |/ __/ _ \\| '_ ` _ \\ / _ \\ | __/ _ \\ ")
	fmt.Println("   \\ V  V /  __/ | (_| (_) | | | | | |  __/ | || (_) |")
	fmt.Println("  __\\_/\\_/_\\___|_|\\___\\___/|_| |_| |_|\\___|  \\__\\___/")
	fmt.Println(" |  _ \\___ \\|  _ \\/ ___|| |__   __ _ _ __ ___")
	fmt.Println(" | |_) |__) | |_) \\___ \\| '_ \\ / _` | '__/ _ \\ ")
	fmt.Println(" |  __// __/|  __/ ___) | | | | (_| | | |  __/")
	fmt.Println(" |_|  |_____|_|   |____/|_| |_|\\__,_|_|  \\___|")
}

func print_help() {
	fmt.Println("--------------- REST Commands ---------------")
	for i := 0; i < len(rest_commands); i++ {
		fmt.Println(rest_commands[i].CommandName + " " + rest_commands[i].Argument + " " + rest_commands[i].HelpText)
	}
	fmt.Println("\n--------------- P2P Commands ---------------")
	for i := 0; i < len(p2p_commands); i++ {
		fmt.Println(p2p_commands[i].CommandName + " " + p2p_commands[i].Argument + " " + p2p_commands[i].HelpText)
	}
	fmt.Println("\n(Type help to display this list)")
	fmt.Println("Press escape to exit the program.")
}

func start(client *http.Client, conn net.PacketConn) {
	fmt.Println("Connecting to server :", server)
	addr_list, err := GetAddresses(client, server_name_peer)
	if err != nil {
		fmt.Println("Error getAddr on server :", err.Error())
		return
	}

	conkeeper_addrs := make([]net.Addr, 0)

	// Check if all the addresses work by sending hello to each of them
	for i := 0; i < len(addr_list); i++ {
		addr, err := net.ResolveUDPAddr("udp", addr_list[i])
		if err != nil {
			fmt.Println("Error resolve addr", err.Error())
			return
		}

		_, err = sendHello(conn, addr, username)
		if err != nil {
			fmt.Println("Error send hello :", err.Error())
			continue
		}

		conkeeper_addrs = append(conkeeper_addrs, addr)
	}

	if len(conkeeper_addrs) == 0 {
		fmt.Println("ERROR : No addresses for the server.")
		return
	}

	go ConnKeeper(client, conn, conkeeper_addrs)
	go PeerClearer()
}

func execCommand(client *http.Client, conn net.PacketConn, content string) {
	AddCommandHistory(content)
	words := strings.Split(content, " ")

	switch words[0] {
	case "list":
		handleList(client)

	case "addr":
		handleListAddr(client, words)

	case "get_key":
		handleGetKey(client, words)

	case "get_root":
		handleGetRoot(client, words)

	case "hello":
		handleSendHello(conn, words)

	case "data":
		handleGetData(client, conn, words)
	case "data_dl":
		handleGetDataDL(client, conn, words)

	case "help":
		print_help()

	default:
		fmt.Println("Unknown command ;-;")

	}
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

	start(client, conn)
	fmt.Print(prompt)

	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer func() { keyboard.Close() }()

	s := ""
	for {
		char, key, _ := keyboard.GetKey()
		switch key {
		case keyboard.KeyArrowUp:
			moveHistoryCursor(true, &s)
		case keyboard.KeyArrowDown:
			moveHistoryCursor(false, &s)

		case keyboard.KeyArrowLeft:
			moveInputCursor(true, &s)
		case keyboard.KeyArrowRight:
			moveInputCursor(false, &s)

		case keyboard.KeyEsc:
			fmt.Println("\n[Exit]")
			return

		case keyboard.KeyEnter:
			fmt.Println("")
			execCommand(client, conn, s)
			fmt.Printf("\n%s", prompt)
			s = ""

		case keyboard.KeyBackspace2:
			if len(s) != 0 {
				if input_cursor != 0 {
					if input_cursor < len(s) {
						s = s[:input_cursor-1] + s[input_cursor:]
						input_cursor--
						fmt.Printf("\r%s%s \r%s%s", prompt, s, prompt, s[:input_cursor])
					} else {
						s = s[:len(s)-1]
						input_cursor = len(s)
						fmt.Printf("\r%s%s \r%s%s", prompt, s, prompt, s)
					}
				}
			}

		case keyboard.KeySpace: // Default case doesn't work with space idk why
			addCharToCommand(" ", &s)

		case keyboard.KeyCtrlC:
			fmt.Println(runtime.NumGoroutine())

		default:
			addCharToCommand(string(char), &s)
		}
	}
}

func handleList(client *http.Client) {
	list, err := GetPeers(client)
	if err != nil {
		fmt.Println("Error getPeers http :", err.Error())
		return
	}

	fmt.Println("Here are the registered peers :")
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

	fmt.Println("Here are the addresses of ", words[1], ":")

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

func handleGetKey(client *http.Client, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	key, err := GetKey(client, words[1])
	if err != nil {
		fmt.Println("Error getKey http : ", err.Error())
		return
	}

	fmt.Printf("Public key of peer %s is : %x\n", words[1], string(key))
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

	fmt.Printf("Root of peer %s is : %x\n", words[1], string(hash))
}

func handleGetData(client *http.Client, conn net.PacketConn, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}
	//TODO: SendHello

	index := FindCachedPeerByName(words[1])
	if index == -1 {
		fmt.Println("Peer not found")
		return
	}

	p := &cache_peers.list[index]
	hash, err := GetRoot(client, p.Name)
	if err != nil {
		fmt.Println("Error getRoot :", err.Error())
		return
	}

	fmt.Println("Hash GetRoot =", hash)
	if p.Root != nil {
		fmt.Println("Hash Root    =", p.Root.Hash)
	}
	if p.Root == nil || [32]byte(hash) != p.Root.Hash {
		fmt.Println("(Re)explore")
		p.Root = BuildNode(p.Name, [32]byte(hash), DIRECTORY)
		explore(conn, p)
	}

	PrintNode(p.Root, "")
}

func handleGetDataDL(client *http.Client, conn net.PacketConn, words []string) {
	if len(words) != 2 && len(words) != 3 {
		fmt.Println("Wrong number of argument !")
		return
	}
	//TODO: SendHello

	index := FindCachedPeerByName(words[1])
	if index == -1 {
		fmt.Println("Peer not found")
		return
	}

	p := &cache_peers.list[index]

	hash, err := GetRoot(client, p.Name)
	if err != nil {
		fmt.Println("Error getRoot :", err.Error())
		return
	}

	if p.Root == nil || p.Root.Hash != [32]byte(hash) {
		fmt.Println("Update du tree ! (Relancer le download)")
		p.Root = BuildNode(p.Name, [32]byte(hash), DIRECTORY)
		explore(conn, p)
		PrintNode(p.Root, "")
		return
	}

	if len(words) == 3 {
		path := strings.Split(words[2], "/")
		start_hash, err := FindPath(p.Root, path[1:])
		if err != nil {
			fmt.Println("Error FindPath :", err.Error())
			return
		}
		if len(path) > 1 {
			real_path := strings.Join(path[:len(path)-1], "/")
			fmt.Println("Real path :", real_path)
			err = os.MkdirAll(real_path, os.ModePerm)
			if err != nil {
				fmt.Println("Error mkdir all handle dl :", err.Error())
				return
			}
		}
		download_multi(conn, p, start_hash, strings.Join(path, "/"))
	} else {
		download_multi(conn, p, [32]byte(hash), p.Name)
	}

	fmt.Println("END !")
}
