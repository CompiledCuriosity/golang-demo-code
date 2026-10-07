// pinned: the shortcut "send each client to the same server". Ten servers,
// each with its own bucket (capacity 5, 1 per second, full). The client is
// pinned to server 1 and sends 50 at once: 5 in. Server 1 restarts (or a new
// server joins) and the client is moved to server 2 a second later, and sends
// 50 at once again. One shared bucket would have refilled one token.
package main

import (
	"fmt"
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

func burst(b *Bucket, at time.Time) int {
	in := 0
	for r := 0; r < 50; r++ {
		if b.Allow(at) {
			in++
		}
	}
	return in
}

func main() {
	start := time.Unix(0, 0)
	server1 := &Bucket{tokens: capacity, last: start}
	server2 := &Bucket{tokens: capacity, last: start}
	shared := &Bucket{tokens: capacity, last: start}
	fmt.Printf("pinned to server 1, 50 at once: %d in\n", burst(server1, start))
	fmt.Printf("moved to server 2 one second later, 50 at once: %d in\n", burst(server2, start.Add(time.Second)))
	a := burst(shared, start)
	b := burst(shared, start.Add(time.Second))
	fmt.Printf("one shared bucket, the same two bursts: %d in, then %d in\n", a, b)
}
