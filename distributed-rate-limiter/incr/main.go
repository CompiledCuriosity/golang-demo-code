// incr: the shortcut "count up with INCR in a key that resets every window".
// To match the bucket's average (5 per 5 seconds = 1 per second), the window is
// 5 seconds and the limit 5: a request at time t increments the counter for
// its window (count:client42:<t/5s>) and gets in while the count is at most 5.
// That is a fixed window, a different limiter: five at 4.9 s and five at 5.1 s
// land in two different windows.
package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

func allowAt(ctx context.Context, rdb *redis.Client, ms int) bool {
	key := fmt.Sprintf("count:client42:%d", ms/5000)
	n := rdb.Incr(ctx, key).Val()
	return n <= 5
}

// Bucket is the token bucket (capacity 5, 1 per second), for the same ten requests
type Bucket struct {
	tokens float64
	last   time.Time
}

func (bucket *Bucket) Allow(now time.Time) bool {
	elapsed := now.Sub(bucket.last).Seconds()
	bucket.tokens = min(5, bucket.tokens+elapsed*1)
	bucket.last = now
	if bucket.tokens < 1 {
		return false
	}
	bucket.tokens--
	return true
}

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintln(os.Stderr, "redis:", err)
		os.Exit(1)
	}
	rdb.Del(ctx, "count:client42:0", "count:client42:1")
	in := 0
	for _, ms := range []int{4900, 4900, 4900, 4900, 4900, 5100, 5100, 5100, 5100, 5100} {
		if allowAt(ctx, rdb, ms) {
			in++
		}
	}
	fmt.Printf("INCR per 5-second window, limit 5 (1 per second on average): five at 4.9 s and five at 5.1 s: %d in, 0.2 s apart\n", in)
	// no race: ten servers, 50 at once inside one window, many trials
	servers := make([]*redis.Client, 10)
	for i := range servers {
		servers[i] = redis.NewClient(&redis.Options{Addr: "localhost:6379", PoolSize: 5})
	}
	exact := 0
	for t := 0; t < 200; t++ {
		rdb.Del(ctx, "count:client42:2")
		var n atomic.Int64
		var wg sync.WaitGroup
		go1 := make(chan struct{})
		for r := 0; r < 50; r++ {
			wg.Add(1)
			go func(c *redis.Client) {
				defer wg.Done()
				<-go1
				if allowAt(ctx, c, 12000) {
					n.Add(1)
				}
			}(servers[r%10])
		}
		close(go1)
		wg.Wait()
		if n.Load() == 5 {
			exact++
		}
	}
	fmt.Printf("INCR, ten servers, 50 at once inside one window: exactly 5 in, %d of 200 trials\n", exact)

	start := time.Unix(0, 0)
	b := &Bucket{tokens: 5, last: start}
	in = 0
	for _, ms := range []int{4900, 4900, 4900, 4900, 4900, 5100, 5100, 5100, 5100, 5100} {
		if b.Allow(start.Add(time.Duration(ms) * time.Millisecond)) {
			in++
		}
	}
	fmt.Printf("a token bucket (capacity 5, full) on the same ten requests: %d in\n", in)
}
