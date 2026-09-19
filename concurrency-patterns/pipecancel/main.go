package main

import (
	"context"
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

func ids(ctx context.Context, n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for id := 1; id <= n; id++ {
			select {
			case out <- id:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func hashes(ctx context.Context, in <-chan int) <-chan result {
	out := make(chan result)
	go func() {
		defer close(out)
		for id := range in {
			select {
			case out <- result{id, hash(id)}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func main() {
	fmt.Println("goroutines at start:", runtime.NumGoroutine())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := hashes(ctx, ids(ctx, 10))
	for i := 0; i < 3; i++ {
		r := <-out
		fmt.Println("doc", r.id, "->", r.sum)
	}
	cancel()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("goroutines after taking 3:", runtime.NumGoroutine())
}
