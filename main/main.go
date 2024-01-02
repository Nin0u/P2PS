package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var debug bool = false
var username string = ""
var open_gui = false

var timeout_datum_clear, _ = time.ParseDuration("30s")
var sleep_time, _ = time.ParseDuration("30s")

var DatumCacheClearer = sync.OnceFunc(func() {
	fmt.Println("DatumCacheClearer Called")
	go datumCacheClearer()
})

func PeerClearer() {
	for {
		time.Sleep(sleep_time)
		current_time := time.Now()
		cache_peers.mutex.Lock()
		for i := len(cache_peers.list) - 1; i > -1; i-- {
			if current_time.Sub(cache_peers.list[i].LastMessageTime) > timeout_cache {
				RemoveCachedPeer(i)
			}
		}
		cache_peers.mutex.Unlock()
	}
}

func datumCacheClearer() {
	sleep_time, _ := time.ParseDuration("30s")
	for {
		time.Sleep(sleep_time)
		current_time := time.Now()
		datumCache.mutex.Lock()
		for k, v := range datumCache.content {
			if current_time.Sub(v.LastTimeUsed) > timeout_datum_clear {
				// fmt.Printf("[DatumCacheClearer] clearing data with hash %x\n", k)
				delete(datumCache.content, k)
			}
		}
		datumCache.mutex.Unlock()
	}
}

func ConnKeeper(client *http.Client, conn net.PacketConn, addrs []net.Addr) {
	for {
		time.Sleep(sleep_time)
		for i := len(addrs) - 1; i > -1; i-- {
			err := sendHello(conn, addrs[i], username, false)
			if err != nil {
				color.Red("[ConnKeeper] Error while sending hello to %s : %s\n ", addrs[i].String(), err.Error())
				addrs = append(addrs[:i], addrs[i+1:]...)

				if len(addrs) == 0 {
					// TODO : possiblement refaire un getAddresses ?
					color.Red("ERROR : No more addresses for the server. Closing ConnKeeper.")
					return
				}
			}
		}
	}
}

func Recv(client *http.Client, conn net.PacketConn) {
	message := make([]byte, 65535+7+64)

	for {
		nb_byte, addr_sender, err := conn.ReadFrom(message)
		if err != nil {
			color.Red("[Recv] Error : %s\n", err.Error())
			continue
		}

		t := GetType(message)

		// Treat Hello separately because it handles handshake between peers
		if t == Hello {
			HandleHello(client, conn, message, nb_byte, addr_sender, username)
		} else if t == HelloReply {
			HandleHelloReply(client, message, nb_byte, addr_sender)
		} else {
			err := CheckHandShake(addr_sender)
			if err != nil {
				if debug {
					fmt.Println(err)
				}
				continue
			}

			switch t {
			case NoOp:
				fmt.Println("[NoOp] Received")
			case Error:
				HandleError(message, "[Error]")

			case PublicKey:
				HandlePublicKey(conn, message, nb_byte, addr_sender)

			case Root:
				HandleRoot(conn, message, nb_byte, addr_sender)

			case GetDatum:
				HandleGetDatum(conn, message, nb_byte, addr_sender)

			case NatTraversal:
				message_bis := make([]byte, nb_byte)
				copy(message_bis, message)
				go HandleNatTraversal(conn, message_bis, nb_byte, addr_sender)

			case ErrorReply:
				HandleError(message, "[ErrorReply]")

			case Datum:
				HandleDatum(message, nb_byte, addr_sender, conn)

			case NoDatum:
				HandleNoDatum(message, nb_byte, addr_sender)

			case RootReply:
				HandleRootReply(conn, message, nb_byte, addr_sender)

			default:
				color.Magenta("[Recv] Unknown message type %d\n", t)
			}
		}
	}
}

// ! Taken from https://gist.github.com/sevkin/9798d67b2cb9d07cb05f89f14ba682f8
// https://stackoverflow.com/questions/39320371/how-start-web-server-to-open-page-in-browser-in-golang
// open opens the specified URL in the default browser of the user.
func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func main() {
	path := ""

	args := os.Args[1:]
	flag_username := false

	if len(args) < 1 {
		fmt.Println("Program must have at least 1 option --username=<username>")
		return
	}

	for i := 0; i < len(args); i++ {
		if args[i] == "--debug" {
			debug = true
			debug_peer = true
			debug_rest = true
			debug_handler = true
			debug_message = true
			debug_signature = true
			debug_export = true
		} else if len(args[i]) >= 11 && args[i][:11] == "--username=" {
			flag_username = true
			name := strings.Trim(args[i][11:], " ")
			if len(name) != 0 {
				username = name
			} else {
				fmt.Println("Username should not be empty !")
				return
			}
		} else if len(args[i]) > 9 && args[i][:9] == "--export=" {
			p := strings.Trim(args[i][9:], " ")
			if len(p) != 0 {
				path = p
			}
		} else if args[i] == "--help" {
			fmt.Println("Here are the options :")
			fmt.Println("--debug             	Launch debug mode")
			fmt.Println("--usename=<username>	Setup username")
			fmt.Println("--export=<path>		Setup export path")
			fmt.Println("--help			Show help")
			return
		} else if args[i] == "--gui" {
			open_gui = true
		}
	}

	if !flag_username {
		fmt.Println("Program must have the option --username=<username>")
		return
	}

	export(path)
	GenKeys()

	transport := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		log.Fatal(err)
	}

	go Recv(client, conn)
	if open_gui {
		go gui(client, conn)
		open("http://localhost:8080/")
	}
	cli(client, conn)
}
