package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Reemit struct {
	mutex sync.Mutex
	list  []Message
}

var reemit_list = Reemit{list: make([]Message, 0)}
var timeout_reemit, _ = time.ParseDuration("5s")

var debug_reemit bool = true

func AddReemit(message Message) {
	if debug_reemit {
		fmt.Println("[AddReemit] message:", message)
	}

	reemit_list.mutex.Lock()

	if debug_reemit {
		fmt.Println("[AddReemit] Old reemit_list:", reemit_list.list)
	}
	reemit_list.list = append(reemit_list.list, message)

	if debug_reemit {
		fmt.Println("[AddReemit] New reemit_list:", reemit_list.list)
	}

	reemit_list.mutex.Unlock()

	if debug_reemit {
		fmt.Println("[AddReemit] End")
	}
}

func FindReemitById(id int32) int32 {
	if debug_reemit {
		fmt.Println("[FindReemitById] id:", id)
	}

	for i := 0; i < len(reemit_list.list); i++ {
		if reemit_list.list[i].Id == id {
			if debug_reemit {
				fmt.Println("[FindReemitById] Found index:", i)
			}

			return int32(i)
		}
	}

	if debug_reemit {
		fmt.Println("[FindReemitById] Index not found")
	}
	return -1
}

func RemoveReemit(id int32) {
	if debug_reemit {
		fmt.Println("[RemoveReemit] id:", id)
	}

	index := FindReemitById(id)

	if debug_reemit {
		fmt.Println("[RemoveReemit] index:", index)
	}

	if index != -1 {
		reemit_list.mutex.Lock()
		if debug_reemit {
			fmt.Println("[RemoveReemit] Old reemit_list:", reemit_list.list)
		}
		reemit_list.list = append(reemit_list.list[:index], reemit_list.list[index+1:]...)

		if debug_reemit {
			fmt.Println("[RemoveReemit] New reemit_list:", reemit_list.list)
		}
		reemit_list.mutex.Unlock()
	}

	if debug_reemit {
		fmt.Println("[RemoveReemit] End")
	}
}

func UpdateReemit(conn net.PacketConn) {
	if debug_reemit {
		fmt.Println("[UpdateReemit] Begin")
	}

	now := time.Now()
	for i := 0; i < len(reemit_list.list); i++ {
		if now.Sub(reemit_list.list[i].LastSentTime) > timeout_reemit {
			if debug_reemit {
				fmt.Println("[UpdateReemit] reemited id:", reemit_list.list[i].Id)
			}

			reemit_list.list[i].LastSentTime = now
			reemit_list.list[i].NbReemit++

			// TODO : error ?
			conn.WriteTo(reemit_list.list[i].build(), reemit_list.list[i].Dest)
		}
	}

	if debug_reemit {
		fmt.Println("[UpdateReemit] End")
	}
}
