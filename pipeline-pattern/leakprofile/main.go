package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"regexp"
	"runtime/pprof"
	"strings"
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

// report prints the goroutineleak profile without its addresses, which change
// from build to build: the total, then the function and line of each record.
func report() {
	var raw bytes.Buffer
	pprof.Lookup("goroutineleak").WriteTo(&raw, 1)
	lines := strings.Split(raw.String(), "\n")
	fmt.Println(lines[0])
	where := regexp.MustCompile(`(main\.\w+)\.func1\+\S+\s+\S*/(main\.go:\d+)`)
	for _, line := range lines[1:] {
		if found := where.FindStringSubmatch(line); found != nil {
			fmt.Println(" ", found[1], "at", found[2])
		}
	}
}

func main() {
	preview()
	time.Sleep(500 * time.Millisecond)
	report()
}
