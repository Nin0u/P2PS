package main

import "sync"

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
