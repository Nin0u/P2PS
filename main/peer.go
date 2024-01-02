package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Peer struct {
	Name            string
	Addr            []net.Addr
	PublicKey       [64]byte
	LastMessageTime time.Time
	Root            *Node
}

type Cache struct {
	mutex sync.Mutex
	list  []Peer
}

var cache_peers Cache = Cache{list: make([]Peer, 0)}
var timeout_cache, _ = time.ParseDuration("180s") // Can't be const unfortunately
var debug_peer bool = false

func BuildPeer(c *http.Client, message []byte, addr_sender net.Addr, key []byte) Peer {
	len_message := getLength(message)
	name_sender := string(message[11 : 7+len_message])
	p := Peer{Name: name_sender, LastMessageTime: time.Now(), Root: nil}
	if key != nil {
		p.PublicKey = [64]byte(key)
	}

	p.Addr = make([]net.Addr, 0)
	p.Addr = append(p.Addr, addr_sender)

	if debug_peer {
		fmt.Printf("[BuildPeer] Building a new peer with name %s\n", name_sender)
	}
	return p
}

func AddAddrToPeer(p *Peer, addr net.Addr) {
	for i := 0; i < len(p.Addr); i++ {
		if addr.String() == p.Addr[i].String() {
			return
		}
	}

	p.Addr = append(p.Addr, addr)
}

func PrintCachedPeers() {
	for i := 0; i < len(cache_peers.list); i++ {
		fmt.Println("\t{", cache_peers.list[i].Name, cache_peers.list[i].Addr, "}")
	}
}

// Tries to find the peer's name in cache.
// If not found adds it
// If found updates its addresses and LastMessageTime
func AddCachedPeer(p Peer) {
	if debug_peer {
		cache_peers.mutex.Lock()
		fmt.Println("[AddCachedPeer] Old Cached Peers")
		PrintCachedPeers()
		cache_peers.mutex.Unlock()
	}

	cache_peers.mutex.Lock()
	index := FindCachedPeerByName(p.Name)
	if index == -1 {
		if debug_peer {
			fmt.Printf("[AddCachedPeer] Adding %s in cache\n", p.Name)
		}
		cache_peers.list = append(cache_peers.list[:], p)
		cache_peers.mutex.Unlock()
	} else {
		if debug_peer {
			fmt.Printf("[AddCachedPeer] %s already in cache. Updating its values\n", p.Name)
		}

		for i := 0; i < len(p.Addr); i++ {
			AddAddrToPeer(&cache_peers.list[index], p.Addr[i])
		}
		cache_peers.list[index].LastMessageTime = p.LastMessageTime
		cache_peers.mutex.Unlock()
	}

	if debug_peer {
		cache_peers.mutex.Lock()
		fmt.Println("[AddCachedPeer] New Cached Peers")
		PrintCachedPeers()
		cache_peers.mutex.Unlock()
	}
}

func RemoveCachedPeer(index int) {
	if debug_peer {
		fmt.Printf("Removing %s from cache\n", cache_peers.list[index].Name)
	}
	cache_peers.list = append(cache_peers.list[:index], cache_peers.list[index+1:]...)
}

func FindCachedPeerByName(name string) int {
	for i := 0; i < len(cache_peers.list); i++ {
		if cache_peers.list[i].Name == name {
			return i
		}
	}
	return -1
}

func FindCachedPeerByAddr(addr net.Addr) int {
	for i := 0; i < len(cache_peers.list); i++ {
		for j := 0; j < len(cache_peers.list[i].Addr); j++ {
			if cache_peers.list[i].Addr[j].String() == addr.String() {
				return i
			}
		}
	}
	return -1
}

func CheckHandShake(addr_sender net.Addr) error {
	if debug_peer {
		fmt.Println("[CheckHandShake] addr:", addr_sender)
	}

	cache_peers.mutex.Lock()
	defer cache_peers.mutex.Unlock()

	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		if debug_peer {
			color.Magenta("[CheckHandShake] Handshake error\n")
		}

		return errors.New("handshake error : peer not cached")
	}

	cache_peers.list[index].LastMessageTime = time.Now()

	return nil
}
