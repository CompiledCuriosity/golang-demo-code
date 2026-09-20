package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"time"
)

type result struct {
	id  int
	sum int
}

func ids(count int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for id := 1; id <= count; id++ {
			number(id)
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

func preview() {
	out := hashes(ids(12))
	taken := 0
	for res := range out {
		fmt.Println("doc", res.id, "->", res.sum)
		taken++
		if taken == 3 {
			return
		}
	}
}

func hash(id int) int {
	sum := []byte(fmt.Sprintf("document-%d", id))
	for round := 0; round < 200000; round++ {
		next := sha256.Sum256(sum)
		sum = next[:]
	}
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
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	for call := 0; call < 1000; call++ {
		preview()
	}

	time.Sleep(500 * time.Millisecond)
	runtime.GC()
	runtime.ReadMemStats(&after)
	fmt.Println("goroutines after 1000 calls:", runtime.NumGoroutine())
	fmt.Fprintln(os.Stderr, "stack in use, KB: before", before.StackInuse/1024, "after", after.StackInuse/1024)
	fmt.Fprintln(os.Stderr, "heap in use, KB:  before", before.HeapInuse/1024, "after", after.HeapInuse/1024)
}
