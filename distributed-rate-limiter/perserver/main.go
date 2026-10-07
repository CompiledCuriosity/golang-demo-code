// perserver: ten servers behind a load balancer, each with its own token
// bucket (capacity 5, 1 token per second, full). One client sends 50 requests
// at one instant, spread round-robin: 5 land on each server.
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
}
