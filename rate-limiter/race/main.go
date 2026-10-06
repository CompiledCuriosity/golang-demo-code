// race: the drawn Allow, unlocked, called from 8 goroutines at once on one
// bucket. `go run .` prints a count; `go run -race .` reports a DATA RACE
// on bucket.tokens / bucket.last (checked by hand, see the runs table).
package main

import (
	"fmt"
	"sync"
	"time"
)

const (
	capacity = 5.0
	rate     = 1.0
)

type Bucket struct {
	tokens float64
	last   time.Time
}

func (bucket *Bucket) Allow(now time.Time) bool {
	elapsed := now.Sub(bucket.last).Seconds()
	bucket.tokens = min(capacity, bucket.tokens+elapsed*rate)
	bucket.last = now
	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

func main() {
	start := time.Unix(0, 0)
	bucket := &Bucket{tokens: capacity, last: start}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				bucket.Allow(start)
			}
		}()
	}
	wg.Wait()
	fmt.Println("8 goroutines x 1000 calls done")
}
