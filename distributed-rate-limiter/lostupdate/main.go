// lostupdate: the naive read-then-write from ten servers, instrumented: for each
// of the 50 requests, the tokens it read, and what the bucket holds once all 50
// are done. Fifty takes happened; how many does the bucket remember?
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
var reads []float64
var readsMu sync.Mutex

func (server *Server) Allow(ctx context.Context) bool {
	now := float64(time.Now().UnixMicro()) / 1e6
	fields := server.rdb.HMGet(ctx, key, "tokens", "last").Val()
	tokens, last := capacity, now
	if fields[0] != nil {
		tokens, _ = strconv.ParseFloat(fields[0].(string), 64)
		last, _ = strconv.ParseFloat(fields[1].(string), 64)
	}
	readsMu.Lock()
	reads = append(reads, tokens)
	readsMu.Unlock()
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
	full, sawOne := 0, 0
	var allReads []float64
	var ends []float64
	counts := make([]int, trials)
	for t := range counts {
		servers[0].rdb.Del(ctx, key)
		reads = reads[:0]
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
		all, one := true, true
		for _, r := range reads {
			if r < capacity {
				all = false
			}
			if r < 1 {
				one = false
			}
		}
		if all {
			full++
		}
		if one {
			sawOne++
		}
		allReads = append(allReads, reads...)
		end, _ := strconv.ParseFloat(servers[0].rdb.HGet(ctx, key, "tokens").Val(), 64)
		ends = append(ends, end)
	}
	atMostFive := 0
	for _, c := range counts {
		if c == 50 {
			atMostFive++ // 50 let in; a bucket of 5 can never remember more than 5 takes
		}
	}
	sort.Float64s(ends)
	// what the reads saw, across every trial: whole tokens, rounded down
	hist := map[int]int{}
	for _, r := range allReads {
		hist[int(r)]++
	}
	fmt.Fprintf(os.Stderr, "reads by tokens seen (rounded down), %d reads: ", len(allReads))
	for k := 5; k >= 0; k-- {
		fmt.Fprintf(os.Stderr, "%d:%d ", k, hist[k])
	}
	fmt.Fprintln(os.Stderr)
	// every line varies run to run (the instrumentation shifts the timing), so
	// nothing here is part of the recorded output
	fmt.Fprintf(os.Stderr, "trials where all 50 got in, so the bucket remembers at most 5 of 50 takes: %d of %d\n", atMostFive, trials)
	fmt.Fprintf(os.Stderr, "every one of the 50 reads saw at least 1 token: %d of %d trials\n", sawOne, trials)
	fmt.Fprintf(os.Stderr, "all 50 reads saw a full bucket before any write: %d of %d trials\n", full, trials)
	fmt.Fprintf(os.Stderr, "tokens left after 50 takes: min %.2f, median %.2f, max %.2f\n", ends[0], ends[len(ends)/2], ends[len(ends)-1])
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
