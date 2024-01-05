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
	"github.com/fatih/color"
)

type Command struct {
	CommandName string
	Argument    string
	HelpText    string
}

var commands = []Command{
	{CommandName: "list   ", Argument: "                   ", HelpText: "list all peers"},
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

func backSpace(s *string) {
	if len(*s) != 0 {
		if input_cursor != 0 {
			if input_cursor < len(*s) {
				*s = (*s)[:input_cursor-1] + (*s)[input_cursor:]
				input_cursor--
				fmt.Printf("\r%s%s \r%s%s", prompt, *s, prompt, (*s)[:input_cursor])
			} else {
				*s = (*s)[:len(*s)-1]
				input_cursor = len(*s)
				fmt.Printf("\r%s%s \r%s%s", prompt, *s, prompt, *s)
			}
		}
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
		color.Red("[Start] Error getAddr on server : %s\n", err.Error())
		return
	}

	addrs := make([]net.Addr, 0)

	// Check if all the addresses work by sending hello to each of them
	for i := 0; i < len(addr_list); i++ {
		addr, err := net.ResolveUDPAddr("udp", addr_list[i])
		if err != nil {
			color.Red("[Start] Error resolve addr : %s\n", err.Error())
			continue
		}

		err = sendHello(conn, addr, false)
		if err != nil {
			color.Magenta("Error send hello : %s\n", err.Error())
			continue
		}

		addrs = append(addrs, addr)
	}

	go ConnKeeper(client, conn, addrs)
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
		execGetDataDL(client, conn, words, ".")

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
			backSpace(&s)
		case keyboard.KeyBackspace:
			backSpace(&s)

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
		color.Red("[ExecList] Error getPeers http : %s\n", err.Error())
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

	addrs_peer, err := GetAddresses(client, words[1])
	if err != nil {
		color.Red("[ExecList] Error getAddresses : %s\n", err.Error())
		return err
	}

	var flag_ok = false

	for i := 0; i < len(addrs_peer); i++ {
		addr, err := net.ResolveUDPAddr("udp", addrs_peer[i])
		if err != nil {
			continue
		}

		err = sendHello(conn, addr, true)
		if err == nil {
			flag_ok = true
		}
	}

	if !flag_ok {
		return err
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
		cache_peers.mutex.Unlock()
		execSendHello(client, conn, words)
		cache_peers.mutex.Lock()
		index = FindCachedPeerByName(words[1])

		if index == -1 {
			cache_peers.mutex.Unlock()
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
		p.Root = BuildNode(p.Name, [32]byte(hash), DIRECTORY)
		err = Explore(conn, p)
		if err != nil {
			p.Root = nil
			return nil, err
		}
	}

	PrintNode(p.Root, "")
	return p, nil
}

func execGetDataDL(client *http.Client, conn net.PacketConn, words []string, prefix string) error {
	if len(words) != 2 && len(words) != 3 {
		return errors.New("wrong number of argument")
	}

	cache_peers.mutex.Lock()
	index := FindCachedPeerByName(words[1])
	if index == -1 {
		cache_peers.mutex.Unlock()
		execSendHello(client, conn, words)
		cache_peers.mutex.Lock()
		index = FindCachedPeerByName(words[1])

		if index == -1 {
			cache_peers.mutex.Unlock()
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
		err = Explore(conn, p)
		PrintNode(p.Root, "")
		if err != nil {
			p.Root = nil
		}
		fmt.Println("Tree has been updated. Please reboot the download")
		return errors.New("tree has been updated. Please reboot the download")
	}

	start_hash := p.Root.Hash
	path := p.Name
	if len(words) == 3 {
		path_tab := strings.Split(words[2], "/")
		start_hash, err = FindPath(p.Root, path_tab[1:])
		if err != nil {
			return err
		}
		path = words[2]
	}

	path = prefix + "/" + path
	path_tab := strings.Split(path, "/")
	if len(path_tab) > 1 {
		real_path := strings.Join(path_tab[:len(path_tab)-1], "/")
		err = os.MkdirAll(real_path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	download_multi(conn, p, start_hash, strings.Join(path_tab, "/"))

	fmt.Println("DONE !")
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
