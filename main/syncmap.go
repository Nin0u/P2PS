package main

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
)

// SyncMap is a structure that helps with packet reemissions and acknowledgements
// Keys are Id(for messages in general) and net.Addr(for NATs) and value are sync.WaitGroup
// When a packet have to be ack-ed, the function that sends waits on a waitgroup that is stored in the map.
// The function that recieves the ack shall unblock the sending function.
type SyncMap[K comparable] struct {
	content map[K]*sync.WaitGroup
	mutex   sync.Mutex
}

var sync_map SyncMap[int32] = SyncMap[int32]{content: make(map[int32]*sync.WaitGroup)}
var nat_sync_map SyncMap[net.Addr] = SyncMap[net.Addr]{content: make(map[net.Addr]*sync.WaitGroup)}

func (this *SyncMap[K]) SetSyncMap(id K, wg *sync.WaitGroup) {
	this.mutex.Lock()
	this.content[id] = wg
	this.mutex.Unlock()
}

func (this *SyncMap[K]) GetSyncMap(id K) (*sync.WaitGroup, bool) {
	this.mutex.Lock()
	wg, ok := this.content[id]
	this.mutex.Unlock()
	return wg, ok
}

func (this *SyncMap[K]) DeleteSyncMap(id K) {
	this.mutex.Lock()
	delete(this.content, id)
	this.mutex.Unlock()
}

// This function unblocks the waitgroup associated with a key
// Should be called by the receiving function
// To prevent double wg.done() that causes crashes we have to make sure that this function locks the map
func (this *SyncMap[K]) Unblock(key K) {
	this.mutex.Lock()
	wg, b := this.content[key]
	if b {
		wg.Done()
		delete(this.content, key)
	}
	this.mutex.Unlock()
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

func (this *SyncMap[K]) Reemit(conn net.PacketConn, addr net.Addr, message *Message, key K, nb_timeout int) (int, error) {
	var wg = &sync.WaitGroup{}
	wg.Add(1)
	this.SetSyncMap(key, wg)

	for i := 0; i < nb_timeout; i++ {
		_, err := conn.WriteTo(message.build(), addr)
		if err != nil {
			if debug_message {
				color.Red("[Reemit] Erreur : %s\n", err.Error())
			}
			return i, err
		}

		has_timedout := waitTimeout(wg, message.Timeout)

		if has_timedout {
			if debug_message {
				color.Magenta("[reemit] Timeout on id : %d %p\n", message.Id, wg)
			}
			message.Timeout *= 2
		} else {
			return i, nil
		}
	}

	// If someone do a wg.done() before -> we have received the packet and have timeout, it's weird but acceptable
	this.Unblock(key)

	return -1, errors.New("\n[reemit] Timeout exceeded")
}
