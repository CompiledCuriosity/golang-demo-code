// xrate: the same request times as trace, through golang.org/x/time/rate,
// set with the same two numbers: rate.NewLimiter(1, 5).
package main

import (
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

func main() {
	start := time.Unix(0, 0)
	limiter := rate.NewLimiter(1, 5)
	times := []int{0, 0, 0, 0, 0, 0, 0, 2500, 2500, 2800, 12800, 12800, 12800, 12800, 12800, 12800, 12800}
	allowed := 0
	for i, ms := range times {
		now := start.Add(time.Duration(ms) * time.Millisecond)
		ok := limiter.AllowN(now, 1)
		if ok {
			allowed++
		}
		fmt.Printf("R%-2d t=%5.1fs  %v  tokens %.1f\n", i+1, float64(ms)/1000, ok, limiter.TokensAt(now))
	}
	fmt.Printf("allowed %d of %d\n", allowed, len(times))
	// Replay R1 to R10 on a fresh limiter, then read the bucket at 3.0s,
	// two tenths of a second after R10 was refused.
	replay := rate.NewLimiter(1, 5)
	for _, ms := range times[:10] {
		replay.AllowN(start.Add(time.Duration(ms)*time.Millisecond), 1)
	}
	fmt.Printf("after R10, the bucket at 3.0s holds %.1f tokens\n", replay.TokensAt(start.Add(3000*time.Millisecond)))
}
