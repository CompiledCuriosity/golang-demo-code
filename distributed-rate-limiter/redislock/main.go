// redislock: the shortcut "keep the lock in Redis". Each server takes a lock
// key in Redis (SET NX with an expiry), then reads the bucket, works it out,
// writes it back, and deletes the lock. A server that finds the lock taken
// tries again. Ten servers, 50 requests at once, capacity 5, many trials;
// how many get in, and how many trips to Redis each request makes.
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	capacity = 5.0
	rate     = 1.0
	key      = "bucket:client42"
	lockKey  = "lock:client42"
	trials   = 200
)

type Server struct {
	rdb   *redis.Client
	trips *atomic.Int64
}

func (server *Server) Allow(ctx context.Context) bool {
	for {
		server.trips.Add(1)
		if ok, _ := server.rdb.SetNX(ctx, lockKey, "held", 100*time.Millisecond).Result(); ok {
			break
		}
		time.Sleep(50 * time.Microsecond)
	}
	now := float64(time.Now().UnixMicro()) / 1e6
	server.trips.Add(1)
	fields := server.rdb.HMGet(ctx, key, "tokens", "last").Val()
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
	server.trips.Add(2)
	server.rdb.HSet(ctx, key, "tokens", tokens, "last", now)
	server.rdb.Del(ctx, lockKey)
	return allowed
}

func main() {
	ctx := context.Background()
	var trips atomic.Int64
	servers := make([]*Server, 10)
	for i := range servers {
		servers[i] = &Server{rdb: redis.NewClient(&redis.Options{Addr: "localhost:6379", PoolSize: 5}), trips: &trips}
		if err := servers[i].rdb.Ping(ctx).Err(); err != nil {
			fmt.Fprintln(os.Stderr, "redis:", err)
			os.Exit(1)
		}
	}
	counts := make([]int, trials)
	var took []time.Duration
	var tookMu sync.Mutex
	for t := range counts {
		servers[0].rdb.Del(ctx, key, lockKey)
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
	exact := 0
	for _, c := range counts {
		if c == 5 {
			exact++
		}
	}
	fmt.Printf("a lock kept in Redis, ten servers: %d trials of 50 at once, capacity 5\n", trials)
	fmt.Printf("  trials that let in exactly 5: %d of %d\n", exact, trials)
	fmt.Printf("  trips to Redis per request, at least: 4 (lock, read, write, unlock)\n")
	sort.Slice(took, func(i, j int) bool { return took[i] < took[j] })
	// these vary run to run, so they are not part of the recorded output
	fmt.Fprintf(os.Stderr, "trips per request, average: %.1f\n", float64(trips.Load())/float64(trials*50))
	fmt.Fprintf(os.Stderr, "time per request, 50 at once: median %v, slowest %v\n", took[len(took)/2], took[len(took)-1])
}
