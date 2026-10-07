// naive: one shared bucket in Redis (capacity 5, 1 token per second), and
// every server runs the refill-and-take itself: read the bucket, work out
// the tokens, write it back. Ten servers, each with its own connection;
// one client sends 50 requests at one instant, 5 to each server.
// Repeated over many trials, because a race does not lose the same way twice.
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
}

// Allow reads the shared bucket, refills it lazily, takes a token if there
// is one, and writes it back: three steps, two round trips, nothing holding
// the bucket in between.
func (server *Server) Allow(ctx context.Context) bool {
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
	report("read, then write, from ten servers", counts)
}

func report(label string, counts []int) {
	sorted := append([]int(nil), counts...)
	sort.Ints(sorted)
	exact, over := 0, 0
	for _, c := range counts {
		if c == int(capacity) {
			exact++
		}
		if c > int(capacity) {
			over++
		}
	}
	fmt.Printf("%s: %d trials of 50 at once, capacity 5\n", label, len(counts))
	fmt.Printf("  trials that let in exactly 5: %d of %d\n", exact, len(counts))
	fmt.Printf("  trials that let in more than 5: %d of %d\n", over, len(counts))
	// how many more varies run to run, so it is not part of the recorded output
	fmt.Fprintf(os.Stderr, "%s: let in min %d, median %d, max %d\n", label, sorted[0], sorted[len(sorted)/2], sorted[len(sorted)-1])
}
