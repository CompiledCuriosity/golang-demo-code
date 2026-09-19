package main

import (
	"crypto/sha256"
	"fmt"
	"runtime"
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

func ids(n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for id := 1; id <= n; id++ {
			out <- id
		}
	}()
	return out
}

func hashes(in <-chan int) <-chan result {
	out := make(chan result)
	go func() {
		defer close(out)
		for id := range in {
			out <- result{id, hash(id)}
		}
	}()
	return out
}

func main() {
	fmt.Println("goroutines at start:", runtime.NumGoroutine())
	out := hashes(ids(12))
	for i := 0; i < 3; i++ {
		r := <-out
		fmt.Println("doc", r.id, "->", r.sum)
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Println("goroutines after taking 3:", runtime.NumGoroutine())
}
