// mutex: the shortcut. Put a lock around the read-then-write, as the Go
// rate package does. Each server has its own lock, because each server is
// its own program on its own machine. Ten servers, 50 requests at once,
// capacity 5, many trials. Then one lock shared by all ten, which only works
// here because these ten "servers" live in one process.
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
	trials   = 200
)

type Server struct {
	rdb *redis.Client
	mu  *sync.Mutex
}

func (server *Server) Allow(ctx context.Context) bool {
	server.mu.Lock()
	defer server.mu.Unlock()
	now := float64(time.Now().UnixMicro()) / 1e6
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
	server.rdb.HSet(ctx, key, "tokens", tokens, "last", now)
	return allowed
}

func run(ctx context.Context, servers []*Server) []int {
	counts := make([]int, trials)
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
				if server.Allow(ctx) {
					in.Add(1)
				}
			}(servers[r%10])
		}
		ready.Wait()
		close(start)
		done.Wait()
		counts[t] = int(in.Load())
	}
	return counts
}

func report(label string, counts []int) {
	sorted := append([]int(nil), counts...)
	sort.Ints(sorted)
	exact, over := 0, 0
	for _, c := range counts {
		if c == 5 {
			exact++
		}
		if c > 5 {
			over++
		}
	}
	fmt.Printf("%s: %d trials of 50 at once, capacity 5\n", label, len(counts))
	fmt.Printf("  trials that let in exactly 5: %d of %d\n", exact, len(counts))
	fmt.Printf("  trials that let in more than 5: %d of %d\n", over, len(counts))
	// how many more varies run to run, so it is not part of the recorded output
	fmt.Fprintf(os.Stderr, "%s: let in min %d, median %d, max %d\n", label, sorted[0], sorted[len(sorted)/2], sorted[len(sorted)-1])
}

func main() {
	ctx := context.Background()
	own := make([]*Server, 10)
	shared := make([]*Server, 10)
	one := &sync.Mutex{}
	for i := range own {
		rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", PoolSize: 5})
		if err := rdb.Ping(ctx).Err(); err != nil {
			fmt.Fprintln(os.Stderr, "redis:", err)
			os.Exit(1)
		}
		own[i] = &Server{rdb: rdb, mu: &sync.Mutex{}}
		shared[i] = &Server{rdb: rdb, mu: one}
	}
	report("a lock on each server", run(ctx, own))
	report("one lock shared by all ten (one process only)", run(ctx, shared))
}
