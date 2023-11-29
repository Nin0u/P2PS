package main

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type Peer struct {
	Name            string
	Addr            net.Addr
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
var debug_peer bool = true

// This function is called on a HelloReply message
func Build_peer(message []byte, addr_sender net.Addr) Peer {
	len := getLength(message)
	name_sender := string(message[11 : 7+len])
	p := Peer{Name: name_sender, Addr: addr_sender, LastMessageTime: time.Now(), Root: nil}

	if debug_peer {
		fmt.Printf("[BuildPeer] Building a new peer with name %s\n", name_sender)
	}
	return p
}

func Add_cached_peer(p Peer) {
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
	if debug {
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
			return i
		}
	}
	cache_peers.mutex.Unlock()
	return -1
}

func FindCachedPeerByAddr(addr net.Addr) int {
	cache_peers.mutex.Lock()
	for i := 0; i < len(cache_peers.list); i++ {
		if cache_peers.list[i].Addr.String() == addr.String() {
			return i
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
