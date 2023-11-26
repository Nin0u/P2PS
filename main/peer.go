package main

import (
	"errors"
	"fmt"
	"net"
	"time"
)

type Peer struct {
	Name            string
	Addr            net.Addr
	PublicKey       [64]uint8
	LastMessageTime time.Time
}

var cache_peers []Peer = make([]Peer, 1)

var timeout, _ = time.ParseDuration("180s")

func Build_peer(message []byte, addr_sender net.Addr) Peer {
	len := getLength(message)
	name_sender := string(message[11 : 7+len])
	p := Peer{Name: name_sender, Addr: addr_sender, LastMessageTime: time.Now()}

	if debug {
		fmt.Printf("Building a new peer with name %s", name_sender)
	}
	return p
}

func Add_cached_peer(p Peer) {
	index := FindCachedPeerByName(p.Name)
	if index == -1 {
		if debug {
			fmt.Printf("Adding %s in cache\n", p.Name)
		}
		cache_peers = append(cache_peers[:], p)
	} else {
		if debug {
			fmt.Printf("%s already in cache. Updating its values\n", p.Name)
		}
		cache_peers[index] = p
	}
}

func removeCachedPeer(index int) {
	if debug {
		fmt.Printf("Removing %s from cache\n", cache_peers[index].Name)
	}
	cache_peers = append(cache_peers[:index], cache_peers[index+1:]...)
}

func HandleTimeoutCachePeers() {
	now := time.Now()
	for i := 0; i < len(cache_peers); i++ {
		if now.Sub(cache_peers[i].LastMessageTime) >= timeout {
			fmt.Printf("%s reached timeout : ", cache_peers[i].Name)
			removeCachedPeer(i)
		}
	}
}

func FindCachedPeerByName(name string) int {
	for i := 0; i < len(cache_peers); i++ {
		if cache_peers[i].Name == name {
			return i
		}
	}
	return -1
}

func FindCachedPeerByAddr(addr net.Addr) int {
	for i := 0; i < len(cache_peers); i++ {
		if cache_peers[i].Addr == addr {
			return i
		}
	}
	return -1
}

func CheckHandShake(addr_sender net.Addr) error {
	index := FindCachedPeerByAddr(addr_sender)
	if index == -1 {
		return errors.New("handshake error : peer not cached")
	}

	return nil
}
