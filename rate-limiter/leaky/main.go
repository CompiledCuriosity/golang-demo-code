// leaky: a leaky bucket used as a queue. Requests join a queue of size 10
// and leave it at a steady pace, one per second; a request that finds the
// queue full is refused. Sixty requests arrive at one instant (t = 0).
package main

import (
	"fmt"
	"time"
)

type Leaky struct {
	size     int
	interval time.Duration
	nextOut  time.Time   // when the next queued request may leave
	queued   []time.Time // leave times of the requests waiting
}

// Arrive queues a request and returns when it will be let out, or false if
// the queue is full.
func (l *Leaky) Arrive(now time.Time) (time.Time, bool) {
	keep := l.queued[:0]
	for _, t := range l.queued {
		if t.After(now) {
			keep = append(keep, t)
		}
	}
	l.queued = keep
	if len(l.queued) >= l.size {
		return time.Time{}, false
	}
	out := l.nextOut
	if out.Before(now) {
		out = now
	}
	l.nextOut = out.Add(l.interval)
	l.queued = append(l.queued, out)
	return out, true
}

func main() {
	start := time.Unix(0, 0)
	l := &Leaky{size: 10, interval: time.Second, nextOut: start}
	queued, refused := 0, 0
	var outs []float64
	for i := 0; i < 60; i++ {
		out, ok := l.Arrive(start)
		if !ok {
			refused++
			continue
		}
		queued++
		outs = append(outs, out.Sub(start).Seconds())
	}
	fmt.Printf("60 requests at one instant: %d queued, %d refused\n", queued, refused)
	fmt.Printf("let out at (s): %v\n", outs)
	most := 0
	for i := range outs {
		n := 0
		for j := range outs {
			if outs[j] >= outs[i] && outs[j] < outs[i]+1 {
				n++
			}
		}
		most = max(most, n)
	}
	fmt.Printf("most let out in any 1 s: %d; the last queued request waits %.0f s\n", most, outs[len(outs)-1])

	// The same sixty against a token bucket of capacity 1, refill 1 per second.
	tokens, in := 1.0, 0
	for i := 0; i < 60; i++ {
		if tokens >= 1 {
			tokens--
			in++
		}
	}
	fmt.Printf("token bucket, capacity 1: 60 requests at one instant: %d in, %d refused at once\n", in, 60-in)
}
