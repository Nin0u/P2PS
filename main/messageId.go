package main

import "sync"

type Id struct {
	mutex      sync.Mutex
	current_id int32
}

func (id *Id) incr() {
	id.mutex.Lock()
	id.current_id++
	id.mutex.Unlock()
}

func (id *Id) get() int32 {
	return id.current_id
}
