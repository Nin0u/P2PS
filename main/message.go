package main

import (
	"crypto/sha256"
	"fmt"
	"net"
)

var current_id int32 = 0

const (
	NoOp                byte = 0
	Error                    = 1
	Hello                    = 2
	PublicKey                = 3
	Root                     = 4
	GetDatum                 = 5
	NatTraversalRequest      = 6
	NatTraversal             = 7

	ErrorReply     = 128
	HelloReply     = 129
	PublicKeyReply = 130
	RootReply      = 131
	Datum          = 132
	NoDatum        = 133
)

func setID(m []byte, id int32) {
	m[0] = byte((id >> 24) % (1 << 8))
	m[1] = byte((id >> 16) % (1 << 8))
	m[2] = byte((id >> 8) % (1 << 8))
	m[3] = byte(id % (1 << 8))
}

func getID(m []byte) int32 {
	return int32(m[0])<<24 + int32(m[1])<<16 + int32(m[2])<<8 + int32(m[3])
}

func setType(m []byte, t byte) {
	m[4] = t
}

func getType(m []byte) byte {
	return m[4]
}

func setLength(m []byte, len uint16) {
	m[5] = byte(len >> 8 % (1 << 8))
	m[6] = byte(len % (1 << 8))
}

func getLength(m []byte) uint16 {
	return uint16(m[5])<<8 + uint16(m[6])
}

func sendHello(conn net.PacketConn, addr net.Addr, name string) (int, error) {
	len := len(name)
	m := make([]byte, 7+len+4)
	setID(m, current_id)

	//TODO: Potentiellement mutex sur current_id
	current_id++

	setType(m, Hello)
	setLength(m, uint16(len+4))

	copy(m[7+4:7+4+len], name)

	if debug {
		fmt.Printf("Hello : %x\n", m)
	}

	return conn.WriteTo(m, addr)
}

func sendHelloReply(conn net.PacketConn, addr net.Addr, name string, id int32) (int, error) {
	len := len(name)
	m := make([]byte, 7+len+4)
	setID(m, id)

	setType(m, HelloReply)
	setLength(m, uint16(len+4))

	copy(m[7+4:7+4+len], name)

	if debug {
		fmt.Printf("HelloReply : %x\n", m)
	}

	return conn.WriteTo(m, addr)
}

// TODO: A changer quand on implémentera les signatures
func sendPublicKeyReply(conn net.PacketConn, addr net.Addr, id int32) (int, error) {
	m := make([]byte, 7)
	setID(m, id)

	setType(m, PublicKeyReply)
	setLength(m, 0)

	if debug {
		fmt.Printf("KeyReply : %x\n", m)
	}

	return conn.WriteTo(m, addr)
}

func sendRootReply(conn net.PacketConn, addr net.Addr, id int32) (int, error) {
	m := make([]byte, 32+7)
	setID(m, id)

	hash := sha256.Sum256([]byte(""))

	setType(m, RootReply)
	setLength(m, 32)

	copy(m[7:7+32], hash[:])

	if debug {
		fmt.Printf("RootReply : %x\n", m)
	}

	return conn.WriteTo(m, addr)
}

func sendGetDatum(conn net.PacketConn, addr net.Addr, hash [32]byte) (int, error) {
	m := make([]byte, 7+32)
	setID(m, current_id)

	//TODO: Potentiellement mutex !
	current_id++

	setType(m, GetDatum)
	setLength(m, 32)

	copy(m[7:7+32], hash[:])

	if debug {
		fmt.Printf("GetDatum : %x\n", m)
	}

	return conn.WriteTo(m, addr)
}

func sendNoDatum(conn net.PacketConn, addr net.Addr, hash [32]byte, id int32) (int, error) {
	m := make([]byte, 7+32)
	setID(m, id)

	setType(m, NoDatum)
	setLength(m, 32)

	copy(m[7:7+32], hash[:])

	if debug {
		fmt.Printf("NoDatum : %x\n", m)
	}

	return conn.WriteTo(m, addr)
}

//TODO: Datum
