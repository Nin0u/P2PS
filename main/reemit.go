package main

import (
	"sync"
)

type Reemit struct {
	mutex sync.Mutex
	list  []Message
}

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

func UpdateReemit() {
	// TODO
}
