package main

import (
	"crypto/sha256"
	"fmt"
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
		defer fmt.Println("ids: loop ended, closing its output")
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
		defer fmt.Println("hashes: range ended, closing its output")
		for id := range in {
			out <- result{id, hash(id)}
		}
	}()
	return out
}

func printAll() {
	out := hashes(ids(12))
	for range out {
	}
	fmt.Println("printAll: range ended")
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
	fmt.Println("goroutines at start:", runtime.NumGoroutine())
	printAll()
	time.Sleep(500 * time.Millisecond)
	fmt.Println("goroutines after printAll:", runtime.NumGoroutine())
}
