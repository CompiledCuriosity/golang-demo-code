package main

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

type result struct {
	id  int
	sum byte
}

func hash(id int) byte {
	h := []byte{byte(id)}
	for i := 0; i < 1000000; i++ {
		s := sha256.Sum256(h)
		h = s[:]
	}
	return h[0]
}

func main() {
	start := time.Now()
	jobs := make(chan int)
	results := make(chan result)

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				results <- result{id, hash(id)}
			}
		}()
	}

	go func() {
		for id := 1; id <= 12; id++ {
			jobs <- id
		}
		close(jobs)
	}()

	wg.Wait()
	close(results)

	n := 0
	for range results {
		n++
	}
	fmt.Println(n, "results in", time.Since(start).Round(time.Millisecond))
}
