// atomic: the same shared bucket and the same ten servers, but the whole
// refill-and-take runs inside Redis as one Lua script. Redis runs one
// script at a time, so no server can read the bucket halfway through
// another server's take.
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	key    = "bucket:client42"
	trials = 200
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

type Server struct {
	rdb *redis.Client
}

func (server *Server) Allow(ctx context.Context) bool {
	allowed, _ := allow.Run(ctx, server.rdb, []string{key}).Int()
	return allowed == 1
}

func main() {
	ctx := context.Background()
	servers := make([]*Server, 10)
	for i := range servers {
		servers[i] = &Server{rdb: redis.NewClient(&redis.Options{Addr: "localhost:6379", PoolSize: 5})}
		if err := servers[i].rdb.Ping(ctx).Err(); err != nil {
			fmt.Fprintln(os.Stderr, "redis:", err)
			os.Exit(1)
		}
	}
	counts := make([]int, trials)
	var took []time.Duration
	var tookMu sync.Mutex
	for t := range counts {
		servers[0].rdb.Del(ctx, key)
		var in atomic.Int64
		var ready, done sync.WaitGroup
		start := make(chan struct{})
		for r := 0; r < 50; r++ {
			ready.Add(1)
			done.Add(1)
			go func(server *Server) {
				defer done.Done()
				ready.Done()
				<-start
				begin := time.Now()
				if server.Allow(ctx) {
					in.Add(1)
				}
				d := time.Since(begin)
				tookMu.Lock()
				took = append(took, d)
				tookMu.Unlock()
			}(servers[r%10])
		}
		ready.Wait()
		close(start)
		done.Wait()
		counts[t] = int(in.Load())
	}
	sorted := append([]int(nil), counts...)
	sort.Ints(sorted)
	exact := 0
	for _, c := range counts {
		if c == 5 {
			exact++
		}
	}
	fmt.Printf("one script inside Redis, ten servers: %d trials of 50 at once, capacity 5\n", trials)
	fmt.Printf("  let in: min %d, max %d\n", sorted[0], sorted[len(sorted)-1])
	fmt.Printf("  trials that let in exactly 5: %d of %d\n", exact, trials)
	sort.Slice(took, func(i, j int) bool { return took[i] < took[j] })
	// these vary run to run, so they are not part of the recorded output
	fmt.Fprintf(os.Stderr, "time per request, 50 at once: median %v, slowest %v\n", took[len(took)/2], took[len(took)-1])
}
