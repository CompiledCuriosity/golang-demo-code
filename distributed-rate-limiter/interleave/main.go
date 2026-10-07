// interleave: the race, one step at a time. The shared bucket holds 1 token.
// Two servers each get one request from the same client at the same moment.
// Each server reads the bucket, decides, and writes the bucket back, and the
// steps are forced into the order that loses: A reads, B reads, A writes,
// B writes. Then the same two requests through the one-script version.
// Then many real (unforced) pairs sent at once, both ways.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	capacity = 5.0
	rate     = 1.0
	key      = "bucket:client42"
	pairs    = 1000
)

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

type read struct{ tokens, last, now float64 }

func readBucket(ctx context.Context, rdb *redis.Client) read {
	now := float64(time.Now().UnixMicro()) / 1e6
	fields := rdb.HMGet(ctx, key, "tokens", "last").Val()
	tokens, _ := strconv.ParseFloat(fields[0].(string), 64)
	last, _ := strconv.ParseFloat(fields[1].(string), 64)
	return read{tokens, last, now}
}

func decideAndWrite(ctx context.Context, rdb *redis.Client, r read) bool {
	tokens := min(capacity, r.tokens+(r.now-r.last)*rate)
	allowed := tokens >= 1
	if allowed {
		tokens--
	}
	rdb.HSet(ctx, key, "tokens", tokens, "last", r.now)
	return allowed
}

func oneToken(ctx context.Context, rdb *redis.Client) {
	t := rdb.Time(ctx).Val()
	rdb.HSet(ctx, key, "tokens", 1.0, "last", float64(t.UnixMicro())/1e6)
}

func stored(ctx context.Context, rdb *redis.Client) float64 {
	v, _ := strconv.ParseFloat(rdb.HGet(ctx, key, "tokens").Val(), 64)
	return v
}

func verdict(b bool) string {
	if b {
		return "let in"
	}
	return "refused"
}

func main() {
	ctx := context.Background()
	a := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	b := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := a.Ping(ctx).Err(); err != nil {
		fmt.Fprintln(os.Stderr, "redis:", err)
		os.Exit(1)
	}

	fmt.Println("forced order, read then write on each server:")
	oneToken(ctx, a)
	ra := readBucket(ctx, a)
	fmt.Printf("  A reads %.0f token\n", ra.tokens)
	rb := readBucket(ctx, b)
	fmt.Printf("  B reads %.0f token\n", rb.tokens)
	inA := decideAndWrite(ctx, a, ra)
	fmt.Printf("  A: %s, writes %.2f\n", verdict(inA), stored(ctx, a))
	inB := decideAndWrite(ctx, b, rb)
	fmt.Printf("  B: %s, writes %.2f\n", verdict(inB), stored(ctx, a))
	fmt.Printf("  two let in on one token; the bucket reads %.2f\n", stored(ctx, a))

	fmt.Println("forced order on a full bucket: A reads, B takes three, A writes:")
	a.Del(ctx, key)
	t := a.Time(ctx).Val()
	a.HSet(ctx, key, "tokens", 5.0, "last", float64(t.UnixMicro())/1e6)
	ra = readBucket(ctx, a)
	fmt.Printf("  A reads %.0f tokens\n", ra.tokens)
	in := 0
	for i := 0; i < 3; i++ {
		if decideAndWrite(ctx, b, readBucket(ctx, b)) {
			in++
		}
		fmt.Printf("  B: let in, writes %.0f\n", stored(ctx, a))
	}
	if decideAndWrite(ctx, a, ra) {
		in++
	}
	fmt.Printf("  A: let in, writes %.0f from its read of 5\n", stored(ctx, a))
	fmt.Printf("  %d let in; the bucket reads %.0f: it remembers %.0f take\n", in, stored(ctx, a), 5-stored(ctx, a))

	fmt.Println("one script each, same order of arrival:")
	oneToken(ctx, a)
	inA, _ = allow.Run(ctx, a, []string{key}).Bool()
	fmt.Printf("  A's script: %s, bucket %.2f\n", verdict(inA), stored(ctx, a))
	inB, _ = allow.Run(ctx, b, []string{key}).Bool()
	fmt.Printf("  B's script: %s, bucket %.2f\n", verdict(inB), stored(ctx, a))

	both, bothScript := 0, 0
	for i := 0; i < pairs; i++ {
		oneToken(ctx, a)
		if runPair(func(rdb *redis.Client) bool { return decideAndWrite(ctx, rdb, readBucket(ctx, rdb)) }, a, b) == 2 {
			both++
		}
		oneToken(ctx, a)
		if runPair(func(rdb *redis.Client) bool { ok, _ := allow.Run(ctx, rdb, []string{key}).Bool(); return ok }, a, b) == 2 {
			bothScript++
		}
	}
	fmt.Printf("%d pairs sent at once on 1 token, unforced:\n", pairs)
	fmt.Printf("  one script: both let in %d times\n", bothScript)
	// the read-then-write count varies run to run, so it is not part of the recorded output
	fmt.Fprintf(os.Stderr, "read then write: both let in %d of %d pairs\n", both, pairs)
}

func runPair(allowFn func(*redis.Client) bool, a, b *redis.Client) int {
	var wg sync.WaitGroup
	var mu sync.Mutex
	in := 0
	start := make(chan struct{})
	for _, rdb := range []*redis.Client{a, b} {
		wg.Add(1)
		go func(rdb *redis.Client) {
			defer wg.Done()
			<-start
			if allowFn(rdb) {
				mu.Lock()
				in++
				mu.Unlock()
			}
		}(rdb)
	}
	close(start)
	wg.Wait()
	return in
}
