package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

var hashed atomic.Int64

func hash(id int) byte {
	h := []byte{byte(id)}
	for i := 0; i < 1000000; i++ {
		s := sha256.Sum256(h)
		h = s[:]
	}
	hashed.Add(1)
	return h[0]
}

func main() {
	start := time.Now()
	g, ctx := errgroup.WithContext(context.Background())
	g.SetLimit(4)

	for id := 1; id <= 12; id++ {
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if id == 7 {
				return errors.New("doc 7: checksum mismatch")
			}
			hash(id)
			return nil
		})
	}

	err := g.Wait()
	fmt.Println("Wait returned:", err)
	fmt.Println("docs hashed:", hashed.Load(), "of 12 in", time.Since(start).Round(time.Millisecond))
}
