// perserver: ten servers behind a load balancer, each with its own token
// bucket (capacity 5, 1 token per second, full). One client sends 50 requests
// at one instant, spread round-robin: 5 land on each server. Then the same 50
// through two load balancer nodes that each keep a bucket.
package main

import (
	"fmt"
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
	servers := make([]*Bucket, 10)
	for i := range servers {
		servers[i] = &Bucket{tokens: capacity, last: start}
	}
	in := 0
	for r := 0; r < 50; r++ {
		if servers[r%10].Allow(start) {
			in++
		}
	}
	fmt.Printf("ten servers, ten buckets: %d of 50 let in\n", in)

	one := &Bucket{tokens: capacity, last: start}
	in = 0
	for r := 0; r < 50; r++ {
		if one.Allow(start) {
			in++
		}
	}
	fmt.Printf("one bucket: %d of 50 let in\n", in)

	// the shortcut "keep the bucket in the load balancer": a load balancer is
	// itself two or more nodes, and each node keeps its own bucket
	nodes := []*Bucket{{tokens: capacity, last: start}, {tokens: capacity, last: start}}
	in = 0
	for r := 0; r < 50; r++ {
		if nodes[r%2].Allow(start) {
			in++
		}
	}
	fmt.Printf("two load balancer nodes, a bucket in each: %d of 50 let in\n", in)

	// the shortcut "split the limit ten ways": each server's bucket gets a tenth,
	// capacity 0.5 and a refill of 0.1 per second: it never holds a whole token
	split := &Bucket{tokens: 0.5, last: start}
	in = 0
	for ms := 0; ms <= 3600*1000; ms += 500 {
		now := start.Add(time.Duration(ms) * time.Millisecond)
		elapsed := now.Sub(split.last).Seconds()
		split.tokens = min(0.5, split.tokens+elapsed*0.1)
		split.last = now
		if split.tokens >= 1 {
			split.tokens--
			in++
		}
	}
	fmt.Printf("a tenth of the limit on one server (capacity 0.5, 0.1 per second), a request every 0.5 s for an hour: %d let in\n", in)

	// rounded up instead: each of the ten servers gets a capacity of 1
	ones := make([]float64, 10)
	for i := range ones {
		ones[i] = 1
	}
	in = 0
	for r := 0; r < 50; r++ {
		if ones[r%10] >= 1 {
			ones[r%10]--
			in++
		}
	}
	fmt.Printf("a share rounded up to 1 token on each of ten servers, 50 at once: %d let in\n", in)
}
