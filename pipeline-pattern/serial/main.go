package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"time"
)

func printAll() {
	for id := 1; id <= 12; id++ {
		number(id)
		fmt.Println("doc", id, "->", hash(id))
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
	start := time.Now()
	printAll()
	fmt.Fprintln(os.Stderr, "plain loop, 12 documents:", time.Since(start).Round(time.Millisecond))
}
