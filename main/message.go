package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"
)

type Message struct {
	Id        int32
	Dest      net.Addr
	Type      byte
	Length    uint16
	Body      []byte
	Signature []byte
	Timeout   time.Duration
}

const (
	NoOp                byte = 0
	Error               byte = 1
	Hello               byte = 2
	PublicKey           byte = 3
	Root                byte = 4
	GetDatum            byte = 5
	NatTraversalRequest byte = 6
	NatTraversal        byte = 7

	ErrorReply     byte = 128
	HelloReply     byte = 129
	PublicKeyReply byte = 130
	RootReply      byte = 131
	Datum          byte = 132
	NoDatum        byte = 133
)

var id = Id{current_id: 0}
var debug_message bool = false

func (m *Message) build() []byte {
	message := make([]byte, 7+m.Length)

	// Write id
	message[0] = byte((m.Id >> 24) % (1 << 8))
	message[1] = byte((m.Id >> 16) % (1 << 8))
	message[2] = byte((m.Id >> 8) % (1 << 8))
	message[3] = byte(m.Id % (1 << 8))

	// Write type
	message[4] = m.Type

	// Write length
	message[5] = byte(m.Length >> 8 % (1 << 8))
	message[6] = byte(m.Length % (1 << 8))

	copy(message[7:], m.Body)

	// Write signature if not nil
	if m.Signature != nil {
		message = append(message, m.Signature...)
	}

	return message
}

func getID(m []byte) int32 {
	return int32(m[0])<<24 + int32(m[1])<<16 + int32(m[2])<<8 + int32(m[3])
}

func GetType(m []byte) byte {
	return m[4]
}

func getLength(m []byte) uint16 {
	return uint16(m[5])<<8 + uint16(m[6])
}

func sendHello(conn net.PacketConn, addr net.Addr, name string) (int, error) {
	if debug_message {
		fmt.Println("[sendHello] Called")
	}
	len_name := len(name)
	message := Message{
		Id:      id.get(),
		Dest:    addr,
		Type:    Hello,
		Length:  uint16(len_name + 4),
		Body:    make([]byte, len_name+4),
		Timeout: time.Second,
	}

	copy(message.Body[4:], name)
	sign := computeSignature(message.build())
	message.Signature = sign
	if debug_message {
		fmt.Printf("[sendHello] Hello : %x\n", message.build())
	}

	n, err := sync_map.Reemit(conn, addr, &message, message.Id, 3)
	if err != nil {
		if n == -1 {
			if debug_message {
				fmt.Println("[sendHello] reemit timeout proceed to NatTraversal")
			}

			// Message is not reliable. Have to reemit the NATTraversal until it's ok
			var wg sync.WaitGroup
			wg.Add(1)
			nat_sync_map.SetSyncMap(addr, &wg)
			return sendAllNatRequest(conn, addr)
		} else {
			if debug_message {
				fmt.Println("[sendHello] Error :", err)
			}
		}
	}

	if debug_message {
		fmt.Printf("[sendHello] message sent after %d tries\n", n)
	}

	return 0, nil
}

func sendHelloReply(conn net.PacketConn, addr net.Addr, name string, id int32) (int32, error) {
	if debug_message {
		fmt.Println("[sendHelloReply] Called")
	}
	len := len(name)
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   HelloReply,
		Length: uint16(len + 4),
		Body:   make([]byte, len+4),
	}

	copy(message.Body[4:], name)
	sign := computeSignature(message.build())
	message.Signature = sign

	if debug_message {
		fmt.Printf("[sendHelloReply] HelloReply : %x\n", message.build())
	}

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

func sendPublicKeyReply(conn net.PacketConn, addr net.Addr, id int32) (int32, error) {
	if debug_message {
		fmt.Println("[sendPublicKeyReply] Called")
	}
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   PublicKeyReply,
		Length: 64,
		Body:   make([]byte, 64),
	}

	publicKey.X.FillBytes(message.Body[:32])
	publicKey.Y.FillBytes(message.Body[32:])

	sign := computeSignature(message.build())
	message.Signature = sign

	if debug_message {
		fmt.Printf("[sendPublicKeyReply] PublicKeyReply : %x\n", message.build())
	}

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

func sendRootReply(conn net.PacketConn, addr net.Addr, id int32) (int32, error) {
	if debug_message {
		fmt.Println("[sendRootReply] Called")
	}
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   RootReply,
		Length: 32,
		Body:   make([]byte, 32),
	}
	hash := [32]byte{}
	if rootExport == nil {
		hash = sha256.Sum256([]byte(""))
	} else {
		hash = rootExport.Hash
	}

	message.Body = hash[:]
	sign := computeSignature(message.build())
	message.Signature = sign

	if debug_message {
		fmt.Printf("[sendRootReply] RootReply : %x\n", message.build())
	}

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

func sendGetDatum(conn net.PacketConn, addr net.Addr, hash [32]byte) (int, error) {
	message := Message{
		Id:      id.get(),
		Dest:    addr,
		Type:    GetDatum,
		Length:  32,
		Body:    make([]byte, 32),
		Timeout: time.Second * 1, //time.Millisecond * 10,
	}

	if debug_message {
		fmt.Printf("[sendGetDatum] id = %d hash = %x\n", message.Id, hash)
	}

	copy(message.Body[:], hash[:])

	if debug {
		fmt.Printf("[sendGetDatum] GetDatum : %x\n", message.build())
	}

	return sync_map.Reemit(conn, addr, &message, message.Id, 5)
}

func sendNoDatum(conn net.PacketConn, addr net.Addr, hash [32]byte, id int32) (int32, error) {
	if debug_message {
		fmt.Println("[sendNoDatum] Called")
	}
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   NoDatum,
		Length: 32,
		Body:   make([]byte, 32),
	}

	copy(message.Body[:], hash[:])

	if debug_message {
		fmt.Printf("[sendNoDatum] NoDatum : %x\n", message.build())
	}

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

func sendDatum(conn net.PacketConn, addr net.Addr, hash [32]byte, id int32, node *ExportNode) (int32, error) {
	if debug_message {
		fmt.Println("[sendDatum] Called")
	}

	message := Message{
		Id:   id,
		Dest: addr,
		Type: Datum,
	}

	message.Body = make([]byte, 0)
	message.Body = append(message.Body, hash[:]...)

	message.Body = append(message.Body, node.Type)
	if node.Type == DIRECTORY {
		for i := 0; i < len(node.Children); i++ {
			path := strings.Split(node.Children[i].Path, "/")
			name := [32]byte{}
			copy(name[:], path[len(path)-1])
			if debug_message {
				fmt.Println("[sendDatum] Name :", name)
			}
			message.Body = append(message.Body, name[:]...)
			message.Body = append(message.Body, node.Children[i].Hash[:]...)
		}
	} else if node.Type == TREE {
		for i := 0; i < len(node.Children); i++ {
			message.Body = append(message.Body, node.Children[i].Hash[:]...)
		}
	} else {
		file, err := os.OpenFile(node.Path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			fmt.Println("[sendDatum] Error open chunk", node.Path, err.Error())
			return -1, err
		}
		chunk := make([]byte, 1024)
		n, err := file.ReadAt(chunk, node.Num)
		if err != nil && err != io.EOF {
			fmt.Println("[sendDatum] Error read chunk", node.Path, err.Error())
			return -1, err
		}

		message.Body = append(message.Body, chunk[:n]...)
	}
	message.Length = uint16(len(message.Body))

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

func sendAllNatRequest(conn net.PacketConn, addr_peer net.Addr) (int, error) {
	index := FindCachedPeerByName(server_name_peer)
	addrs_server := make([]net.Addr, 0)
	cache_peers.mutex.Lock()
	copy(addrs_server, cache_peers.list[index].Addr)
	cache_peers.mutex.Unlock()
	for i := 0; i < len(addrs_server); i++ {
		ret, err := sendNatRequest(conn, addr_peer, addrs_server[i])
		if err != nil {
			return ret, err
		}
	}

	return 0, nil
}

func sendNatRequest(conn net.PacketConn, addr_peer net.Addr, addr_server net.Addr) (int, error) {
	message := Message{
		Id:   id.get(),
		Type: NatTraversalRequest,
	}

	message.Dest = addr_server

	ip, err := netip.ParseAddrPort(addr_peer.String())
	if err != nil {
		fmt.Println("[sendNatRequest] Error parse addr :", err.Error())
		return -1, nil
	}

	ip_byte := ip.Addr().AsSlice()
	port := ip.Port()
	message.Body = make([]byte, 0)
	message.Body = append(message.Body, ip_byte...)
	message.Body = append(message.Body, byte(port>>8%(1<<8)))
	message.Body = append(message.Body, byte(port%(1<<8)))
	message.Length = uint16(len(message.Body))

	return nat_sync_map.Reemit(conn, addr_server, &message, addr_peer, 3)
}
