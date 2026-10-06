// timer: the shortcut a viewer reaches for (a timer that drops one token into
// every client's bucket each second) against the lazy refill (two numbers per
// client, touched only when that client sends a request).
// One million clients, ten quiet seconds, then one request from one client.
package main

import (
	"fmt"
	"os"
	"time"
)

const (
	clients  = 1_000_000
	capacity = 5.0
	rate     = 1.0
)

type Bucket struct {
	tokens float64
	last   time.Time
}

func main() {
	start := time.Unix(0, 0)

	// The timer way: every second, top up every bucket.
	timerBuckets := make([]float64, clients)
	writes := 0
	t0 := time.Now()
	for sec := 1; sec <= 10; sec++ {
		for i := range timerBuckets {
			timerBuckets[i] = min(capacity, timerBuckets[i]+rate)
			writes++
		}
	}
	fmt.Fprintf(os.Stderr, "timer pass, 10 ticks over %d buckets: %v\n", clients, time.Since(t0))
	fmt.Printf("timer: %d bucket writes in 10 quiet seconds (%d per second)\n", writes, writes/10)

	// The lazy way: nothing happens until a request arrives.
	lazy := make([]Bucket, clients)
	for i := range lazy {
		lazy[i] = Bucket{tokens: 0, last: start}
	}
	lazyWrites := 0
	now := start.Add(10 * time.Second)
	b := &lazy[42]
	b.tokens = min(capacity, b.tokens+now.Sub(b.last).Seconds()*rate)
	b.last = now
	lazyWrites++
	fmt.Printf("lazy:  %d bucket write in the same 10 seconds (the one request); that bucket now holds %.1f tokens\n", lazyWrites, b.tokens)
	fmt.Printf("state per client: %d numbers, tokens and last time\n", 2)
}
