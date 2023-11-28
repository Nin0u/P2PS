package main

import (
	"net"
	"sync"
	"time"
)

type Reemit struct {
	mutex sync.Mutex
	list  []Message
}

var timeout_reemit, _ = time.ParseDuration("5s")

var reemit_list = Reemit{list: make([]Message, 1)}

func AddReemit(message Message) {
	reemit_list.mutex.Lock()
	reemit_list.list = append(reemit_list.list[:], message)
	reemit_list.mutex.Unlock()
}

func FindReemitById(id int32) int {
	for i := 0; i < len(reemit_list.list); i++ {
		if reemit_list.list[i].Id == id {
			return i
		}
	}

	return -1
}

func RemoveReemit(id int32) {
	index := FindReemitById(id)
	if index != -1 {
		reemit_list.mutex.Lock()
		reemit_list.list = append(reemit_list.list[:index], reemit_list.list[index+1:]...)
		reemit_list.mutex.Unlock()
	}
}

func UpdateReemit(conn net.PacketConn) {
	now := time.Now()
	for i := 0; i < len(reemit_list.list); i++ {
		if now.Sub(reemit_list.list[i].LastSentTime) > timeout_reemit {
			reemit_list.list[i].LastSentTime = now
			reemit_list.list[i].NbReemit++

			// TODO : error ?
			conn.WriteTo(reemit_list.list[i].build(), reemit_list.list[i].Dest)
		}
	}
}
