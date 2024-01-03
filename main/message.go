package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Id struct {
	mutex      sync.Mutex
	current_id int32
}

func (id *Id) get() int32 {
	id.mutex.Lock()
	ret := id.current_id
	id.current_id++
	id.mutex.Unlock()
	return ret
}

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

func (m *Message) print() {
	fmt.Printf("Id = %d, Type = %b, Len = %d, Body = %x, Sign = %x\n", m.Id, m.Type, m.Length, m.Body, m.Signature)
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

func ComputeRTO(conn net.PacketConn, addr net.Addr) time.Duration {
	RTT := 2 * time.Second
	RTTvar := 0 * time.Second
	alpha := 7 * time.Second / 8
	beta := 3 * time.Second / 8

	for i := 0; i < 10; i++ {
		start_t := time.Now()
		sendHello(conn, addr, false)
		end_t := time.Now()
		to := end_t.Sub(start_t)
		delta := (to - RTT).Abs()
		RTT = alpha*RTT + (1-alpha)*to
		RTTvar = beta*RTTvar + (1-beta)*delta
	}

	return RTT + (4 * RTTvar)
}

func sendHello(conn net.PacketConn, addr net.Addr, send_NT bool) error {
	if debug_message {
		fmt.Println("[sendHello] Called")
	}
	len_name := len(username)
	message := Message{
		Id:      id.get(),
		Dest:    addr,
		Type:    Hello,
		Length:  uint16(len_name + 4),
		Body:    make([]byte, len_name+4),
		Timeout: time.Second,
	}

	cache_peers.mutex.Lock()
	index_peer, index_addr := FindCachedPeerByAddr(addr)
	if index_peer != -1 {
		message.Timeout = cache_peers.list[index_peer].Addr[index_addr].RTO
	}
	cache_peers.mutex.Unlock()

	copy(message.Body[4:], username)
	sign := computeSignature(message.build())
	message.Signature = sign
	if debug_message {
		fmt.Printf("[sendHello] Hello : ")
		message.print()
	}

	n, err := sync_map.Reemit(conn, addr, &message, message.Id, 3)
	if err != nil {
		// n = -1 if the reemit timed out. Then we send (or not) a NatTraversal
		if n == -1 {
			if send_NT {
				if debug_message {
					color.Magenta("[sendHello] reemit timeout proceed to NatTraversal\n")
				}

				// Message is not reliable. Have to reemit the NATTraversal until it's ok
				return sendAllNatRequest(conn, addr)
			}

			return err
		} else {
			if debug_message {
				color.Red("[sendHello] Error : %s\n", err.Error())
			}
		}
	}

	if debug_message {
		fmt.Printf("[sendHello] message sent after %d tries\n", n+1)
	}

	return nil
}

func sendHelloReply(conn net.PacketConn, addr net.Addr, id int32) (int32, error) {
	if debug_message {
		fmt.Println("[sendHelloReply] Called")
	}
	len := len(username)
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   HelloReply,
		Length: uint16(len + 4),
		Body:   make([]byte, len+4),
	}

	copy(message.Body[4:], username)
	sign := computeSignature(message.build())
	message.Signature = sign

	if debug_message {
		fmt.Printf("[sendHelloReply] HelloReply : ")
		message.print()
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
		fmt.Printf("[sendPublicKeyReply] PublicKeyReply : ")
		message.print()
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

	map_export.Mutex.Lock()
	if rootExport == nil {
		hash := sha256.Sum256([]byte(""))
		copy(message.Body[:], hash[:])
	} else {
		copy(message.Body[:], rootExport.Hash[:])
	}
	map_export.Mutex.Unlock()

	sign := computeSignature(message.build())
	message.Signature = sign

	if debug_message {
		fmt.Printf("[sendRootReply] RootReply : ")
		message.print()
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
		Timeout: time.Second,
	}

	cache_peers.mutex.Lock()
	index_peer, index_addr := FindCachedPeerByAddr(addr)
	if index_peer != -1 {
		message.Timeout = cache_peers.list[index_peer].Addr[index_addr].RTO
	}
	cache_peers.mutex.Unlock()

	if debug_message {
		fmt.Printf("[sendGetDatum] id = %d hash = %x\n", message.Id, hash)
	}

	copy(message.Body[:], hash[:])

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
		fmt.Printf("[sendNoDatum] NoDatum : ")
		message.print()
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
			name := [32]byte{}
			copy(name[:], []byte(node.Children[i].Name))
			if debug_message {
				fmt.Println("[sendDatum] Name :", node.Children[i].Name)
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
			color.Red("[sendDatum] Error open chunk %s : %s\n", node.Path, err.Error())
			return -1, err
		}
		chunk := make([]byte, 1024)
		n, err := file.ReadAt(chunk, node.Num)
		if err != nil && err != io.EOF {
			color.Red("[sendDatum] Error read chunk %s : %s\n", node.Path, err.Error())
			return -1, err
		}

		message.Body = append(message.Body, chunk[:n]...)
	}
	message.Length = uint16(len(message.Body))

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

func GetServerAddrs() ([]AddrRTO, error) {
	cache_peers.mutex.Lock()
	index := FindCachedPeerByName(server_name_peer)
	if index == -1 {
		color.Red("[getServerAddrs] Error finding server name")
		cache_peers.mutex.Unlock()
		return nil, errors.New("finding server name")
	}
	addrs_server := make([]AddrRTO, len(cache_peers.list[index].Addr))
	copy(addrs_server, cache_peers.list[index].Addr)
	cache_peers.mutex.Unlock()

	return addrs_server, nil
}

func sendAllNatRequest(conn net.PacketConn, addr_peer net.Addr) error {
	addrs_server, err := GetServerAddrs()
	if err != nil {
		return err
	}

	var e error = nil
	for i := 0; i < len(addrs_server); i++ {
		fmt.Println("[sendAllNatRequest] Sending a NAT Request")

		_, err := sendNatRequest(conn, addr_peer, addrs_server[i])
		if err != nil {
			e = err
			color.Red("[sendAllNatRequest] Error : %s\n", err.Error())
		}
	}

	return e
}

func sendNatRequest(conn net.PacketConn, addr_peer net.Addr, addr_server AddrRTO) (int, error) {
	message := Message{
		Id:      id.get(),
		Type:    NatTraversalRequest,
		Timeout: addr_server.RTO,
	}

	message.Dest = addr_server.Addr

	ip, err := netip.ParseAddrPort(addr_peer.String())
	if err != nil {
		color.Red("[sendNatRequest] Error parse addr : %s\n", err.Error())
		return -1, err
	}

	ip_byte := ip.Addr().AsSlice()
	port := ip.Port()
	message.Body = make([]byte, 0)
	message.Body = append(message.Body, ip_byte...)
	message.Body = append(message.Body, byte(port>>8%(1<<8)))
	message.Body = append(message.Body, byte(port%(1<<8)))
	message.Length = uint16(len(message.Body))

	if debug_message {
		fmt.Printf("[sendNatRequest] NatRequest : ")
		message.print()
	}

	return nat_sync_map.Reemit(conn, addr_server.Addr, &message, addr_peer, 3)
}

func sendRoot(conn net.PacketConn) error {
	message := Message{
		Id:     id.get(),
		Type:   Root,
		Body:   make([]byte, 32),
		Length: 32,
	}

	map_export.Mutex.Lock()
	copy(message.Body, rootExport.Hash[:])
	map_export.Mutex.Unlock()

	if debug_message {
		fmt.Print("[sendRoot] Root :")
		message.print()
	}

	addrs_server, err := GetServerAddrs()
	if err != nil {
		return err
	}

	sign := computeSignature(message.build())
	message.Signature = sign

	for i := 0; i < len(addrs_server); i++ {
		message.Timeout = addrs_server[i].RTO
		_, err := sync_map.Reemit(conn, addrs_server[i].Addr, &message, message.Id, 3)
		if err != nil {
			color.Red("[SendRoot] Error : %s\n", err.Error())
			return err
		}
	}
	return nil
}
