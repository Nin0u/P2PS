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

type AddrRTO struct {
	Addr net.Addr
	RTO  time.Duration
}

type Peer struct {
	Name            string
	Addr            []AddrRTO
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

	p.Addr = make([]AddrRTO, 0)
	p.Addr = append(p.Addr, AddrRTO{Addr: addr_sender, RTO: time.Second})

	if debug_peer {
		fmt.Printf("[BuildPeer] Building a new peer with name %s\n", name_sender)
	}
	return p
}

func (p *Peer) AddAddr(addr AddrRTO) {
	for i := 0; i < len(p.Addr); i++ {
		if addr.Addr.String() == p.Addr[i].Addr.String() {
			return
		}
	}

	p.Addr = append(p.Addr, addr)
}

func (p *Peer) AddRTO(addr net.Addr, rto time.Duration) {
	for i := 0; i < len(p.Addr); i++ {
		if addr.String() == p.Addr[i].Addr.String() {
			p.Addr[i].RTO = rto
			return
		}
	}
}

func PrintCachedPeers() {
	for i := 0; i < len(cache_peers.list); i++ {
		fmt.Println("\t{", cache_peers.list[i].Name, cache_peers.list[i].Addr, "}")
	}
}

// Tries to find the peer's name in cache.
// If not found adds it and return true
// If found updates its addresses and LastMessageTime and return false
func AddCachedPeer(p Peer) bool {
	flag := false
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
		flag = true
	} else {
		if debug_peer {
			fmt.Printf("[AddCachedPeer] %s already in cache. Updating its values\n", p.Name)
		}

		for i := 0; i < len(p.Addr); i++ {
			(&cache_peers.list[index]).AddAddr(p.Addr[i])
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

	return flag
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

// The first int is the index of the peer
// The second is the index of the address in the peers' addresses
func FindCachedPeerByAddr(addr net.Addr) (int, int) {
	for i := 0; i < len(cache_peers.list); i++ {
		for j := 0; j < len(cache_peers.list[i].Addr); j++ {
			if cache_peers.list[i].Addr[j].Addr.String() == addr.String() {
				return i, j
			}
		}
	}
	return -1, -1
}

func CheckHandShake(addr_sender net.Addr) error {
	if debug_peer {
		fmt.Println("[CheckHandShake] addr:", addr_sender)
	}

	cache_peers.mutex.Lock()
	defer cache_peers.mutex.Unlock()

	index, _ := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		if debug_peer {
			color.Magenta("[CheckHandShake] Handshake error\n")
		}

		return errors.New("handshake error : peer not cached")
	}

	cache_peers.list[index].LastMessageTime = time.Now()

	return nil
}
