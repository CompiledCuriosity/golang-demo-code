// longrun: a client that never stops sending (a request every 500 ms; 0.5 is exact in binary floating point, 0.1 is not and loses a token to rounding over a minute) against
// the bucket (capacity 5, 1 token per second), full at t=0. How many get in
// over one minute, and over one hour? And a sliding window of 60 per 60 s
// (a log of request times) facing 60 requests at one instant.
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

func (b *Bucket) Allow(now time.Time) bool {
	elapsed := now.Sub(b.last).Seconds()
	b.tokens = min(capacity, b.tokens+elapsed*rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// SlidingLog remembers the time of every request let in during the last window.
type SlidingLog struct {
	limit  int
	window time.Duration
	times  []time.Time
}

func (s *SlidingLog) Allow(now time.Time) bool {
	keep := s.times[:0]
	for _, t := range s.times {
		if now.Sub(t) < s.window {
			keep = append(keep, t)
		}
	}
	s.times = keep
	if len(s.times) >= s.limit {
		return false
	}
	s.times = append(s.times, now)
	return true
}

func hammerOpen(d time.Duration) int {
	start := time.Unix(0, 0)
	b := &Bucket{tokens: capacity, last: start}
	in := 0
	for t := time.Duration(0); t < d; t += 500 * time.Millisecond {
		if b.Allow(start.Add(t)) {
			in++
		}
	}
	return in
}

func hammer(d time.Duration) int {
	start := time.Unix(0, 0)
	b := &Bucket{tokens: capacity, last: start}
	in := 0
	for t := time.Duration(0); t <= d; t += 500 * time.Millisecond {
		if b.Allow(start.Add(t)) {
			in++
		}
	}
	return in
}

func main() {
	fmt.Printf("request every 0.5s from 0s to 60s inclusive: %d let in\n", hammer(time.Minute))
	fmt.Printf("request every 0.5s from 0s to 3600s inclusive: %d let in\n", hammer(time.Hour))
	fmt.Printf("request every 0.5s from 0s to just under 60s (half-open minute): %d let in\n", hammerOpen(time.Minute))

	start := time.Unix(0, 0)
	s := &SlidingLog{limit: 60, window: time.Minute}
	in := 0
	for i := 0; i < 60; i++ {
		if s.Allow(start) {
			in++
		}
	}
	fmt.Printf("sliding window (60 per 60s), 60 requests at one instant: %d let in, remembering %d request times\n", in, len(s.times))
	// the hook's edge: 60 at 59.5 s, 60 at 60.5 s
	s2 := &SlidingLog{limit: 60, window: time.Minute}
	in2 := 0
	for _, at := range []time.Duration{59500 * time.Millisecond, 60500 * time.Millisecond} {
		for i := 0; i < 60; i++ {
			if s2.Allow(start.Add(at)) {
				in2++
			}
		}
	}
	fmt.Printf("sliding window, 60 at 59.5s and 60 at 60.5s: %d let in\n", in2)

	// Ten servers behind a load balancer, each with its own bucket
	// (capacity 5, 1 per second, full); one client's burst of 70 requests,
	// spread round-robin across the ten at one instant.
	servers := make([]*Bucket, 10)
	for i := range servers {
		servers[i] = &Bucket{tokens: capacity, last: start}
	}
	spread := 0
	for i := 0; i < 70; i++ {
		if servers[i%10].Allow(start) {
			spread++
		}
	}
	fmt.Printf("10 servers, own bucket each, 70 requests at once round-robin: %d let in\n", spread)

	// The same ten servers, the client sending nonstop for a minute, a
	// request to each server every 0.5 s.
	for i := range servers {
		servers[i] = &Bucket{tokens: capacity, last: start}
	}
	minute := 0
	for t := time.Duration(0); t <= time.Minute; t += 500 * time.Millisecond {
		for _, s := range servers {
			if s.Allow(start.Add(t)) {
				minute++
			}
		}
	}
	fmt.Printf("10 servers, own bucket each, nonstop for a minute: %d let in (one bucket: 65)\n", minute)

	// Split the limit ten ways: each server's bucket gets a tenth of both
	// numbers, capacity 0.5 and 0.1 token per second, full at the start.
	// One request every 0.5 s to one server for an hour.
	tenth := struct {
		tokens float64
		last   time.Time
	}{0.5, start}
	tin := 0
	for t := time.Duration(0); t <= time.Hour; t += 500 * time.Millisecond {
		now := start.Add(t)
		tenth.tokens = min(0.5, tenth.tokens+now.Sub(tenth.last).Seconds()*0.1)
		tenth.last = now
		if tenth.tokens >= 1 {
			tenth.tokens--
			tin++
		}
	}
	fmt.Printf("one server's tenth of the limit (capacity 0.5, 0.1 per second), a request every 0.5 s for an hour: %d let in\n", tin)

	// Pin each client to one server: the client empties its bucket on
	// server A, then a restart (or a new server) moves it to server B,
	// whose bucket for it is new and full.
	serverA := &Bucket{tokens: capacity, last: start}
	onA := 0
	for i := 0; i < 7; i++ {
		if serverA.Allow(start) {
			onA++
		}
	}
	serverB := &Bucket{tokens: capacity, last: start.Add(time.Second)}
	onB := 0
	for i := 0; i < 7; i++ {
		if serverB.Allow(start.Add(time.Second)) {
			onB++
		}
	}
	fmt.Printf("pinned client: 7 at once on server A: %d in; moved to server B one second later, 7 at once: %d in (one bucket would give 1)\n", onA, onB)
}
