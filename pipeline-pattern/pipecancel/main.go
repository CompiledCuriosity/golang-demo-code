package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
)

type result struct {
	id  int
	sum int
}

func ids(ctx context.Context, count int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for id := 1; id <= count; id++ {
			number(id)
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

func preview() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := hashes(ctx, ids(ctx, 12))
	taken := 0
	for res := range out {
		fmt.Println("doc", res.id, "->", res.sum)
		taken++
		if taken == 3 {
			return
		}
	}
}

var hashed atomic.Int64

func hash(id int) int {
	sum := []byte(fmt.Sprintf("document-%d", id))
	for round := 0; round < 200000; round++ {
		next := sha256.Sum256(sum)
		sum = next[:]
	}
	hashed.Add(1)
	return int(sum[0])
}

// number stands in for registering the document under its id (a lookup,
// a write): real time at the first desk,
// and no count (docs hashed counts the sorter's work only). Three quarters
// of hash's work, so the clerk is always ahead of the sorter: at equal
// weights the fixed program hashed 3 or 4 of 12 from run to run.
func number(id int) {
	text := []byte(fmt.Sprintf("envelope-%d", id))
	for round := 0; round < 150000; round++ {
		next := sha256.Sum256(text)
		text = next[:]
	}
}

func main() {
	fmt.Println("goroutines at start:", runtime.NumGoroutine())
	preview()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("goroutines after preview:", runtime.NumGoroutine())
	fmt.Println("docs hashed:", hashed.Load(), "of 12")
}
