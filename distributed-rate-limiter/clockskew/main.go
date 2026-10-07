// clockskew: why the script reads the time from Redis. Two servers take turns
// (no race: one request at a time), each working out what is owed by its OWN
// clock. Server B's clock runs 2 seconds fast. A burst of 5 on server A drains
// the bucket; then, at the same real moment, 5 more arrive at server B.
// Then the same two bursts through the one-script bucket, which reads the
// time from Redis, so both servers count time on one clock.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	capacity = 5.0
	rate     = 1.0
	key      = "bucket:client42"
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

// take is the read-then-write take, with the server's clock passed in
func take(ctx context.Context, rdb *redis.Client, now float64) bool {
	fields := rdb.HMGet(ctx, key, "tokens", "last").Val()
	tokens, last := capacity, now
	if fields[0] != nil {
		tokens, _ = strconv.ParseFloat(fields[0].(string), 64)
		last, _ = strconv.ParseFloat(fields[1].(string), 64)
	}
	tokens = min(capacity, tokens+(now-last)*rate)
	allowed := tokens >= 1
	if allowed {
		tokens--
	}
	rdb.HSet(ctx, key, "tokens", tokens, "last", now)
	return allowed
}

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintln(os.Stderr, "redis:", err)
		os.Exit(1)
	}
	real := float64(time.Now().UnixMicro()) / 1e6
	clockA, clockB := real, real+2 // B's clock runs 2 seconds fast

	rdb.Del(ctx, key)
	inA, inB := 0, 0
	for i := 0; i < 5; i++ {
		if take(ctx, rdb, clockA) {
			inA++
		}
	}
	for i := 0; i < 5; i++ {
		if take(ctx, rdb, clockB) {
			inB++
		}
	}
	fmt.Println("each server on its own clock, B's 2 seconds fast:")
	fmt.Printf("  5 at server A: %d in; then, at the same moment, 5 at server B: %d in\n", inA, inB)

	rdb.Del(ctx, key)
	inA, inB = 0, 0
	for i := 0; i < 5; i++ {
		if n, _ := allow.Run(ctx, rdb, []string{key}).Int(); n == 1 {
			inA++
		}
	}
	for i := 0; i < 5; i++ {
		if n, _ := allow.Run(ctx, rdb, []string{key}).Int(); n == 1 {
			inB++
		}
	}
	fmt.Println("one script, the time read from Redis:")
	fmt.Printf("  5 at server A: %d in; then 5 at server B: %d in\n", inA, inB)
}
