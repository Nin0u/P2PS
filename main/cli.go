package main

import (
	"errors"
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

		_, err = sendHello(conn, addr, username, false)
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
		execList(client)

	case "hello":
		execSendHello(client, conn, words)

	case "data":
		execGetData(client, conn, words)

	case "data_dl":
		execGetDataDL(client, conn, words)

	case "export":
		execExport(conn, words)

	case "help":
		print_help()

	default:
		fmt.Println("Unknown command ;-;")

	}
}

func cli(client *http.Client, conn net.PacketConn) {
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

func execList(client *http.Client) {
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

func execSendHello(client *http.Client, conn net.PacketConn, words []string) error {
	if len(words) != 2 {
		return errors.New("wrong number of argument")
	}

	addrs_peer, err1 := GetAddresses(client, words[1])
	var err2 error
	var err3 error
	for i := 0; i < len(addrs_peer); i++ {
		addr, err := net.ResolveUDPAddr("udp", addrs_peer[i])
		if err != nil {
			err2 = err
		}

		_, err = sendHello(conn, addr, username, true)
		if err != nil {
			err3 = err
		}
	}

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	if err3 != nil {
		return err3
	}

	return nil
}

func execGetData(client *http.Client, conn net.PacketConn, words []string) (*Peer, error) {
	if len(words) != 2 {
		return nil, errors.New("wrong number of argument")
	}

	cache_peers.mutex.Lock()
	index := FindCachedPeerByName(words[1])
	if index == -1 {
		execSendHello(client, conn, words)
		index = FindCachedPeerByName(words[1])
		if index == -1 {
			return nil, errors.New("peer not found")
		}
	}
	p := &cache_peers.list[index]
	cache_peers.mutex.Unlock()

	hash, err := GetRoot(client, p.Name)
	if err != nil {
		return nil, err
	}

	if p.Root == nil || [32]byte(hash) != p.Root.Hash {
		fmt.Println("(Re)explore")
		p.Root = BuildNode(p.Name, [32]byte(hash), DIRECTORY)
		explore(conn, p)
	}

	PrintNode(p.Root, "")
	return p, nil
}

func execGetDataDL(client *http.Client, conn net.PacketConn, words []string) error {
	if len(words) != 2 && len(words) != 3 {
		return errors.New("wrong number of argument")
	}

	cache_peers.mutex.Lock()
	index := FindCachedPeerByName(words[1])
	if index == -1 {
		execSendHello(client, conn, words)
		index = FindCachedPeerByName(words[1])
		if index == -1 {
			return errors.New("peer not found")
		}
	}
	p := &cache_peers.list[index]
	cache_peers.mutex.Unlock()

	hash, err := GetRoot(client, p.Name)
	if err != nil {
		return err
	}

	if p.Root == nil || p.Root.Hash != [32]byte(hash) {
		p.Root = BuildNode(p.Name, [32]byte(hash), DIRECTORY)
		explore(conn, p)
		PrintNode(p.Root, "")
		return errors.New("Tree has been updated. Please reboot the download")
	}

	if len(words) == 3 {
		path := strings.Split(words[2], "/")
		start_hash, err := FindPath(p.Root, path[1:])
		if err != nil {
			return err
		}
		if len(path) > 1 {
			real_path := strings.Join(path[:len(path)-1], "/")
			err = os.MkdirAll(real_path, os.ModePerm)
			if err != nil {
				return err
			}
		}
		download_multi(conn, p, start_hash, strings.Join(path, "/"))
	} else {
		download_multi(conn, p, [32]byte(hash), p.Name)
	}

	fmt.Println("END !")
	return nil
}

func execExport(conn net.PacketConn, words []string) error {
	if len(words) != 2 {
		return errors.New("wrong number of argument")
	}

	err := export(words[1])
	if err != nil {
		return err
	}

	err = sendRoot(conn)
	if err != nil {
		return err
	}

	return nil
}
