package main

import (
	"crypto/sha256"
	"fmt"
	"time"
)

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
	n := 0
	for id := 1; id <= 12; id++ {
		hash(id)
		n++
	}
	fmt.Println(n, "results in", time.Since(start).Round(time.Millisecond))
}
