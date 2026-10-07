// decr: the shortcut "Redis commands are already atomic, so just DECR the
// token count". One DECR per request takes a token atomically; a result of 0
// or more means let in. Ten servers, 50 at once, then two seconds of quiet
// (two tokens owed at 1 per second) and 50 more. Nothing adds the owed tokens.
// Then the same two bursts through the one-script bucket, for comparison.
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

const key = "count:client42"

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

func scriptBurst(ctx context.Context, servers []*redis.Client) int {
	var in atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for r := 0; r < 50; r++ {
		wg.Add(1)
		go func(rdb *redis.Client) {
			defer wg.Done()
			<-start
			if n, _ := allow.Run(ctx, rdb, []string{"bucket:client42"}).Int(); n == 1 {
				in.Add(1)
			}
		}(servers[r%10])
	}
	close(start)
	wg.Wait()
	return int(in.Load())
}

func burst(ctx context.Context, servers []*redis.Client) int {
	var in atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for r := 0; r < 50; r++ {
		wg.Add(1)
		go func(rdb *redis.Client) {
			defer wg.Done()
			<-start
			if n, _ := rdb.Decr(ctx, key).Result(); n >= 0 {
				in.Add(1)
			}
		}(servers[r%10])
	}
	close(start)
	wg.Wait()
	return int(in.Load())
}

func main() {
	ctx := context.Background()
	servers := make([]*redis.Client, 10)
	for i := range servers {
		servers[i] = redis.NewClient(&redis.Options{Addr: "localhost:6379", PoolSize: 5})
		if err := servers[i].Ping(ctx).Err(); err != nil {
			fmt.Fprintln(os.Stderr, "redis:", err)
			os.Exit(1)
		}
	}
	servers[0].Set(ctx, key, 5, 0)
	fmt.Printf("DECR, 50 at once on 5 tokens: %d in\n", burst(ctx, servers))
	time.Sleep(2 * time.Second)
	fmt.Printf("two seconds later, 50 at once: %d in\n", burst(ctx, servers))
	servers[0].Del(ctx, "bucket:client42")
	fmt.Printf("one script, 50 at once on a full bucket: %d in\n", scriptBurst(ctx, servers))
	time.Sleep(2 * time.Second)
	fmt.Printf("two seconds later, 50 at once: %d in\n", scriptBurst(ctx, servers))
}
