// edge: "60 requests per minute" as a fixed-window counter versus a token
// bucket (capacity 5, refill 1 token per second = 60 per minute on average).
// A client sends 60 requests at 0:59.5 and 60 more at 1:00.5, one second apart.
package main

import (
	"fmt"
	"time"
)

// FixedWindow counts requests per window (a calendar minute, or a second)
// and resets at each new window.
type FixedWindow struct {
	limit  int
	size   time.Duration
	window int64 // which window the count belongs to
	count  int
}

func (f *FixedWindow) Allow(now time.Time) bool {
	w := now.UnixNano() / int64(f.size)
	if w != f.window {
		f.window, f.count = w, 0
	}
	if f.count >= f.limit {
		return false
	}
	f.count++
	return true
}

const rate = 1.0 // tokens per second

type Bucket struct {
	capacity float64
	tokens   float64
	last     time.Time
}

func (b *Bucket) Allow(now time.Time) bool {
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(b.capacity, b.tokens+elapsed*rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func main() {
	start := time.Unix(0, 0)
	fw := &FixedWindow{limit: 60, size: time.Minute}
	b5 := &Bucket{capacity: 5, tokens: 5, last: start}
	b60 := &Bucket{capacity: 60, tokens: 60, last: start}
	total := [3]int{}
	for _, at := range []time.Duration{59500 * time.Millisecond, 60500 * time.Millisecond} {
		now := start.Add(at)
		ok := [3]int{}
		for i := 0; i < 60; i++ {
			for j, allow := range []func(time.Time) bool{fw.Allow, b5.Allow, b60.Allow} {
				if allow(now) {
					ok[j]++
				}
			}
		}
		for j := range ok {
			total[j] += ok[j]
		}
		fmt.Printf("60 requests at %.1fs: fixed window %d in, bucket(cap 5) %d in, bucket(cap 60) %d in\n", at.Seconds(), ok[0], ok[1], ok[2])
	}
	fmt.Printf("total of 120 in one second: fixed window %d, bucket(cap 5) %d, bucket(cap 60) %d\n", total[0], total[1], total[2])

	// The shortcut: shrink the window to one second, limit one. One counter,
	// one run, as the strip draws it: two requests together at 19.3 s, then
	// one at 20.9 s and one at 21.1 s.
	sec := &FixedWindow{limit: 1, size: time.Second}
	pair := 0
	for i := 0; i < 2; i++ {
		if sec.Allow(start.Add(19300 * time.Millisecond)) {
			pair++
		}
	}
	fmt.Printf("per-second counter (limit 1): 2 requests at 19.3s: %d in\n", pair)
	edge := 0
	for _, at := range []time.Duration{20900 * time.Millisecond, 21100 * time.Millisecond} {
		if sec.Allow(start.Add(at)) {
			edge++
		}
	}
	fmt.Printf("same counter: requests at 20.9s and 21.1s: %d in, 0.2s apart\n", edge)
	// Two counters stacked, the way a gateway lets you stack limits:
	// five per second AND sixty per minute. Five requests at 20.9 s, five at 21.1 s.
	perSec := &FixedWindow{limit: 5, size: time.Second}
	perMin := &FixedWindow{limit: 60, size: time.Minute}
	both := 0
	for _, at := range []time.Duration{20900 * time.Millisecond, 21100 * time.Millisecond} {
		for i := 0; i < 5; i++ {
			now := start.Add(at)
			if perSec.Allow(now) && perMin.Allow(now) {
				both++
			}
		}
	}
	fmt.Printf("two counters (5 per second and 60 per minute): 5 at 20.9s and 5 at 21.1s: %d in, 0.2s apart\n", both)

	// The same ten requests against a bucket of capacity 5, 1 per second, full.
	b := &Bucket{capacity: 5, tokens: 5, last: start}
	bin := 0
	for _, at := range []time.Duration{20900 * time.Millisecond, 21100 * time.Millisecond} {
		for i := 0; i < 5; i++ {
			if b.Allow(start.Add(at)) {
				bin++
			}
		}
	}
	fmt.Printf("bucket (capacity 5, 1 per second): the same 5 at 20.9s and 5 at 21.1s: %d in\n", bin)
}
