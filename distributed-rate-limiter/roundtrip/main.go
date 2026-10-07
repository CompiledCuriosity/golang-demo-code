// roundtrip: what the shared bucket costs. The bucket in the server's own
// memory (#125's ten-line Allow) against the one-script call to Redis on
// this same machine, the shortest a network round trip gets.
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	capacity = 5.0
	rate     = 1.0
	key      = "bucket:client42"
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

var allow = redis.NewScript(`
local capacity, rate = 5, 1
local time = redis.call('TIME')
local now = time[1] + time[2] / 1000000
local bucket = redis.call('HMGET', KEYS[1], 'tokens', 'last')
local tokens = tonumber(bucket[1]) or capacity
local last = tonumber(bucket[2]) or now
tokens = math.min(capacity, tokens + (now - last) * rate)
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'last', now)
return allowed
`)

func median(d []time.Duration) time.Duration {
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	return d[len(d)/2]
}

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintln(os.Stderr, "redis:", err)
		os.Exit(1)
	}
	const n = 1_000_000
	bucket := &Bucket{tokens: capacity, last: time.Now()}
	begin := time.Now()
	for i := 0; i < n; i++ {
		bucket.Allow(time.Now())
	}
	local := time.Since(begin) / n

	allow.Load(ctx, rdb)
	const m = 20_000
	took := make([]time.Duration, m)
	for i := range took {
		t := time.Now()
		allow.Run(ctx, rdb, []string{key})
		took[i] = time.Since(t)
	}
	remote := median(took)

	// read then write, the two-trip take, for comparison
	took2 := make([]time.Duration, m)
	for i := range took2 {
		t := time.Now()
		rdb.HMGet(ctx, key, "tokens", "last")
		rdb.HSet(ctx, key, "tokens", 4.0, "last", 0.0)
		took2[i] = time.Since(t)
	}
	fmt.Fprintf(os.Stderr, "read then write, two trips: %v median per call\n", median(took2))
	fmt.Fprintf(os.Stderr, "in memory: %v per call; one script in Redis on this machine: %v median per call (%.0fx)\n",
		local, remote, float64(remote)/float64(local))
}
