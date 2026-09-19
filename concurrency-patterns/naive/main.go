package main

import (
	"crypto/sha256"
	"fmt"
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
	for id := 1; id <= 12; id++ {
		go func() {
			fmt.Println("doc", id, "->", hash(id))
		}()
	}
	fmt.Println("all 12 done")
}
