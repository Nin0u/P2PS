package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
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

	// TODO : error ?
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

// ! Code found on https://stackoverflow.com/questions/32840687/timeout-for-waitgroup-wait/32840688#32840688
// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

func reemit(conn net.PacketConn, addr net.Addr, message *Message) (int, error) {
	defer DeleteSyncMap(message.Id)
	var wg sync.WaitGroup
	wg.Add(1)
	SetSyncMap(message.Id, &wg)
	// The user will wait 7seconds max before aborting
	for i := 0; i < 3; i++ {
		_, err := conn.WriteTo(message.build(), addr)
		if err != nil {
			return i, err
		}

		// Timeout peut etre pour éviter de bloquer indéfiniment
		has_timedout := waitTimeout(&wg, message.Timeout)

		if has_timedout {
			message.Timeout *= 2
		} else {
			return i, nil
		}
	}

	return -1, errors.New("[reemit] Timeout exceeded")
}

func sendHello(conn net.PacketConn, addr net.Addr, name string) (int, error) {
	len := len(name)
	message := Message{
		Id:      id.get(),
		Dest:    addr,
		Type:    Hello,
		Length:  uint16(len + 4),
		Body:    make([]byte, len+4),
		Timeout: time.Second,
	}

	id.incr()

	// TODO : error ?
	copy(message.Body[4:], name)

	if debug {
		fmt.Printf("Hello : %x\n", message.build())
	}

	return reemit(conn, addr, &message)
}

func sendHelloReply(conn net.PacketConn, addr net.Addr, name string, id int32) (int32, error) {
	len := len(name)
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   HelloReply,
		Length: uint16(len + 4),
		Body:   make([]byte, len+4),
	}

	// TODO : error ?
	copy(message.Body[4:], name)

	if debug {
		fmt.Printf("HelloReply : %x\n", message.build())
	}

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

// TODO: A changer quand on implémentera les signatures
func sendPublicKeyReply(conn net.PacketConn, addr net.Addr, id int32) (int32, error) {
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   PublicKeyReply,
		Length: 0,
	}

	if debug {
		fmt.Printf("KeyReply : %x\n", message.build())
	}

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

func sendRootReply(conn net.PacketConn, addr net.Addr, id int32) (int32, error) {
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   RootReply,
		Length: 32,
		Body:   make([]byte, 32),
	}
	hash := sha256.Sum256([]byte(""))

	copy(message.Body[:], hash[:])

	if debug {
		fmt.Printf("RootReply : %x\n", message.build())
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
		Timeout: time.Second * 1,
	}
	id.incr()
	copy(message.Body[:], hash[:])

	if debug {
		fmt.Printf("GetDatum : %x\n", message.build())
	}

	return reemit(conn, addr, &message)
}

func sendNoDatum(conn net.PacketConn, addr net.Addr, hash [32]byte, id int32) (int32, error) {
	message := Message{
		Id:     id,
		Dest:   addr,
		Type:   NoDatum,
		Length: 32,
		Body:   make([]byte, 32),
	}

	copy(message.Body[:], hash[:])

	if debug {
		fmt.Printf("NoDatum : %x\n", message.build())
	}

	_, err := conn.WriteTo(message.build(), addr)
	return message.Id, err
}

//TODO: Datum
