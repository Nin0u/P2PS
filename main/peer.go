package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
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

/*
Tries to find the peer's name in cache.
If not found adds it
If found updates its addresses and LastMessageTime
*/
func AddCachedPeer(p Peer) {
	if debug_peer {
		cache_peers.mutex.Lock()
		fmt.Println("[AddCachedPeer] Old Cached Peers ", cache_peers.list)
		cache_peers.mutex.Unlock()
		fmt.Println("[AddCachedPeer] Calling FindCachePeerByName")
	}

	index := FindCachedPeerByName(p.Name)
	if index == -1 {
		if debug_peer {
			fmt.Printf("[AddCachedPeer] Adding %s in cache\n", p.Name)
		}
		cache_peers.mutex.Lock()
		cache_peers.list = append(cache_peers.list[:], p)
		cache_peers.mutex.Unlock()
	} else {
		if debug_peer {
			fmt.Printf("[AddCachedPeer] %s already in cache. Updating its values\n", p.Name)
		}
		cache_peers.mutex.Lock()
		for i := 0; i < len(p.Addr); i++ {
			AddAddrToPeer(&cache_peers.list[index], p.Addr[i])
		}
		cache_peers.list[index].LastMessageTime = p.LastMessageTime
		cache_peers.mutex.Unlock()
	}

	if debug_peer {
		cache_peers.mutex.Lock()
		fmt.Println("[AddCachedPeer] New Cached Peers ", cache_peers.list)
		cache_peers.mutex.Unlock()
	}
}

func removeCachedPeer(index int) {
	if debug_peer {
		fmt.Printf("Removing %s from cache\n", cache_peers.list[index].Name)
	}
	cache_peers.list = append(cache_peers.list[:index], cache_peers.list[index+1:]...)
}

func FindCachedPeerByName(name string) int {
	cache_peers.mutex.Lock()
	for i := 0; i < len(cache_peers.list); i++ {
		if cache_peers.list[i].Name == name {
			cache_peers.mutex.Unlock()
			return i
		}
	}
	cache_peers.mutex.Unlock()
	return -1
}

func FindCachedPeerByAddr(addr net.Addr) int {
	for i := 0; i < len(cache_peers.list); i++ {
		for j := 0; j < len(cache_peers.list[i].Addr); j++ {
			if cache_peers.list[i].Addr[j].String() == addr.String() {
				cache_peers.mutex.Unlock()
				return i
			}
		}
	}
	return -1
}

func UpdatePeerLastMessageTime(index int) {
	cache_peers.mutex.Lock()
	cache_peers.list[index].LastMessageTime = time.Now()
	cache_peers.mutex.Unlock()
}

func CheckHandShake(addr_sender net.Addr) (int, error) {
	if debug_peer {
		fmt.Println("[CheckHandShake] addr:", addr_sender)
	}

	cache_peers.mutex.Lock()
	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		if debug_peer {
			fmt.Println("[CheckHandShake] Handshake error")
		}

		return index, errors.New("handshake error : peer not cached")
	}
	UpdatePeerLastMessageTime(index)
	cache_peers.mutex.Unlock()

	return index, nil
}
