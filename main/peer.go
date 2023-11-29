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
}

type Cache struct {
	mutex sync.Mutex
	list  []Peer
}

var cache_peers Cache = Cache{list: make([]Peer, 0)}

var timeout, _ = time.ParseDuration("180s")

func Build_peer(message []byte, addr_sender net.Addr) Peer {
	len := getLength(message)
	name_sender := string(message[11 : 7+len])
	p := Peer{Name: name_sender, Addr: addr_sender, LastMessageTime: time.Now()}

	if debug {
		fmt.Printf("Building a new peer with name %s\n", name_sender)
	}
	return p
}

func Add_cached_peer(p Peer) {
	index := FindCachedPeerByName(p.Name)
	if index == -1 {
		if debug {
			fmt.Printf("Adding %s in cache\n", p.Name)
		}
		cache_peers.mutex.Lock()
		cache_peers.list = append(cache_peers.list[:], p)
		cache_peers.mutex.Unlock()
	} else {
		if debug {
			fmt.Printf("%s already in cache. Updating its values\n", p.Name)
		}
		cache_peers.mutex.Lock()
		cache_peers.list[index] = p
		cache_peers.mutex.Unlock()
	}

	fmt.Println("Peer Cached ", cache_peers.list)
}

func removeCachedPeer(index int) {
	if debug {
		fmt.Printf("Removing %s from cache\n", cache_peers.list[index].Name)
	}
	cache_peers.mutex.Lock()
	cache_peers.list = append(cache_peers.list[:index], cache_peers.list[index+1:]...)
	cache_peers.mutex.Unlock()
}

func HandleTimeoutCachePeers() {
	now := time.Now()
	for i := 0; i < len(cache_peers.list); i++ {
		if now.Sub(cache_peers.list[i].LastMessageTime) >= timeout {
			fmt.Printf("%s reached timeout : ", cache_peers.list[i].Name)
			removeCachedPeer(i)
		}
	}
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
		if cache_peers.list[i].Addr.String() == addr.String() {
			return i
		}
	}
	return -1
}

func CheckHandShake(addr_sender net.Addr) error {
	fmt.Println("Addr_sender : ", addr_sender)
	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		return errors.New("handshake error : peer not cached")
	}

	return nil
}
