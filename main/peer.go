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
	PublicKey       [64]uint8
	LastMessageTime time.Time
	Root            *Node
}

type Cache struct {
	mutex sync.Mutex
	list  []Peer
}

var cache_peers Cache = Cache{list: make([]Peer, 0)}
var timeout_cache, _ = time.ParseDuration("180s")
var debug_peer bool = false

func BuildPeer(c *http.Client, message []byte, addr_sender net.Addr) Peer {
	len_message := getLength(message)
	name_sender := string(message[11 : 7+len_message])
	p := Peer{Name: name_sender, LastMessageTime: time.Now(), Root: nil}
	p.Addr = make([]net.Addr, 0)

	addresses, err := GetAddresses(c, name_sender)
	if err != nil {
		fmt.Println("[BuildPeer]", err)
	}

	for i := 0; i < len(addresses); i++ {
		ad, err := net.ResolveUDPAddr("udp", addresses[i])
		if err != nil {
			fmt.Println("[BuildPeer] Error resolve addr", err.Error())
		}
		p.Addr = append(p.Addr, ad)
	}

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
		cache_peers.list[index] = p
		cache_peers.mutex.Unlock()
	}

	if debug_peer {
		cache_peers.mutex.Lock()
		fmt.Println("[AddCachedPeer] New Cached Peers ", cache_peers.list)
		cache_peers.mutex.Unlock()
	}
}

func removeCachedPeer(index int) {
	cache_peers.mutex.Lock()
	if debug_peer {
		fmt.Printf("Removing %s from cache\n", cache_peers.list[index].Name)
	}
	cache_peers.list = append(cache_peers.list[:index], cache_peers.list[index+1:]...)
	cache_peers.mutex.Unlock()
}

func HandletimeoutCachePeers() {
	now := time.Now()
	for i := 0; i < len(cache_peers.list); i++ {
		if now.Sub(cache_peers.list[i].LastMessageTime) >= timeout_cache {
			fmt.Printf("%s reached timeout_cache : ", cache_peers.list[i].Name)
			removeCachedPeer(i)
		}
	}
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
	cache_peers.mutex.Lock()
	for i := 0; i < len(cache_peers.list); i++ {
		for j := 0; j < len(cache_peers.list[i].Addr); j++ {
			if cache_peers.list[i].Addr[j].String() == addr.String() {
				cache_peers.mutex.Unlock()
				return i
			}
		}
	}
	cache_peers.mutex.Unlock()
	return -1
}

func CheckHandShake(addr_sender net.Addr) error {
	if debug_peer {
		fmt.Println("[CheckHandShake] addr:", addr_sender)
	}

	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		if debug_peer {
			fmt.Println("[CheckHandShake] Handshake error")
		}

		return errors.New("handshake error : peer not cached")
	}

	return nil
}
