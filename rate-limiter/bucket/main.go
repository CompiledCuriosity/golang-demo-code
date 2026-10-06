// trace: the worked example exactly as the board draws it.
// A token bucket with capacity 5 and a refill rate of 1 token per second,
// full at t=0, refilled lazily: when a request arrives, the tokens owed
// since the last request are added (never above capacity), then one is taken.
package main

import (
	"fmt"
	"math"
	"time"
)

const (
	capacity = 5.0 // tokens: the most at once
	rate     = 1.0 // tokens per second: the most on average
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
	// request times in milliseconds
	times := []int{0, 0, 0, 0, 0, 0, 0, 2500, 2500, 2800, 12800, 12800, 12800, 12800, 12800, 12800, 12800}
	allowed := 0
	for i, ms := range times {
		now := start.Add(time.Duration(ms) * time.Millisecond)
		before := bucket.tokens
		elapsed := now.Sub(bucket.last).Seconds()
		ok := bucket.Allow(now)
		line := fmt.Sprintf("R%-2d t=%5.1fs  had %.1f  +%.1f owed  -> %.1f  ", i+1, float64(ms)/1000, before, elapsed*rate, min(capacity, before+elapsed*rate))
		if ok {
			allowed++
			line += fmt.Sprintf("ALLOW  left %.1f", bucket.tokens)
		} else {
			wait := (1 - bucket.tokens) / rate
			line += fmt.Sprintf("429    wait %.1fs  Retry-After: %d", wait, int(math.Ceil(wait)))
		}
		fmt.Println(line)
	}
	fmt.Printf("allowed %d of %d\n", allowed, len(times))
}
