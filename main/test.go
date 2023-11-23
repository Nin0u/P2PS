package main

import "sync"

func test_mutex() {
	var wg sync.WaitGroup
	for c := 0; c < 2; c++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100000; i++ {
				id.incr()
			}
			defer wg.Done()
		}()
	}
	wg.Wait()
}
