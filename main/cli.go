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

var commands = []Command{{
	CommandName: "list    ", Argument: "                   ", HelpText: "list all peers"},
	{CommandName: "data   ", Argument: "<peername>         ", HelpText: "list data of the peer"},
	{CommandName: "data_dl", Argument: "<peername> [<path>]", HelpText: "download data of the peer. If a path is given then it will download all the data from this path."},
	{CommandName: "export ", Argument: "<path>             ", HelpText: "export the file/folder you choose"},
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
	} else {
		*s = command_history[history_cursor]
	}

	fmt.Printf("\r%s%s\r%s%s", prompt, blank, prompt, *s)
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
	fmt.Println("--------------- Commands ---------------")
	for i := 0; i < len(commands); i++ {
		fmt.Println(commands[i].CommandName + " " + commands[i].Argument + " " + commands[i].HelpText)
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

	case "hello":
		handleSendHello(client, conn, words)

	case "data":
		handleGetData(client, conn, words)

	case "data_dl":
		handleGetDataDL(client, conn, words)

	case "export":
		treatExport(conn, words)

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

// TODO: Tous les handler devrait renvoyait un booléen, ou un erreur pour savoir si tout c'est bien passé
// ! Neccessaire pour le gui !
func handleSendHello(client *http.Client, conn net.PacketConn, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	addrs_peer, err := GetAddresses(client, words[1])
	if err != nil {
		fmt.Println("Error while fetching peer's addresses")
		return
	}

	for i := 0; i < len(addrs_peer); i++ {
		addr, err := net.ResolveUDPAddr("udp", addrs_peer[i])
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
}

func handleGetData(client *http.Client, conn net.PacketConn, words []string) *Peer {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return nil
	}
	//TODO: SendHello test la reponse
	handleSendHello(client, conn, words)

	index := FindCachedPeerByName(words[1])
	if index == -1 {
		fmt.Println("Peer not found")
		return nil
	}

	p := &cache_peers.list[index]
	hash, err := GetRoot(client, p.Name)
	if err != nil {
		fmt.Println("Error getRoot :", err.Error())
		return nil
	}

	if p.Root == nil || [32]byte(hash) != p.Root.Hash {
		fmt.Println("(Re)explore")
		p.Root = BuildNode(p.Name, [32]byte(hash), DIRECTORY)
		explore(conn, p)
	}

	PrintNode(p.Root, "")
	return p
}

func handleGetDataDL(client *http.Client, conn net.PacketConn, words []string) {
	if len(words) != 2 && len(words) != 3 {
		fmt.Println("Wrong number of argument !")
		return
	}
	//TODO: SendHello test la reponse
	handleSendHello(client, conn, []string{"hello", words[1]})

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

func treatExport(conn net.PacketConn, words []string) {
	if len(words) != 2 {
		fmt.Println("Wrong number of argument !")
		return
	}

	err := export(words[1])
	if err != nil {
		fmt.Println("[treatExport] ", err.Error())
		return
	}

	err = sendRoot(conn)
	if err != nil {
		fmt.Println("[treatExport] ", err.Error())
		return
	}

	return
}
