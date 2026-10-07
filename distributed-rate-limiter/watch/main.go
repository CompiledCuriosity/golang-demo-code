// watch: the shortcut "use a Redis transaction". Each server WATCHes the
// bucket, reads it, works it out, and writes it back in MULTI/EXEC; if another
// server wrote the bucket in between, EXEC fails and the server starts over.
// Ten servers, 50 at once, capacity 5, many trials.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
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

func allow(ctx context.Context, rdb *redis.Client, attempts *atomic.Int64) bool {
	for {
		attempts.Add(1)
		allowed := false
		err := rdb.Watch(ctx, func(tx *redis.Tx) error {
			now := float64(time.Now().UnixMicro()) / 1e6
			fields := tx.HMGet(ctx, key, "tokens", "last").Val()
			tokens, last := capacity, now
			if fields[0] != nil {
				tokens, _ = strconv.ParseFloat(fields[0].(string), 64)
				last, _ = strconv.ParseFloat(fields[1].(string), 64)
			}
			tokens = min(capacity, tokens+(now-last)*rate)
			allowed = tokens >= 1
			if allowed {
				tokens--
			}
			_, err := tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
				p.HSet(ctx, key, "tokens", tokens, "last", now)
				return nil
			})
			return err
		}, key)
		if errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return allowed
	}
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
	var attempts atomic.Int64
	exact := 0
	for t := 0; t < trials; t++ {
		servers[0].Del(ctx, key)
		var in atomic.Int64
		var wg sync.WaitGroup
		start := make(chan struct{})
		for r := 0; r < 50; r++ {
			wg.Add(1)
			go func(rdb *redis.Client) {
				defer wg.Done()
				<-start
				if allow(ctx, rdb, &attempts) {
					in.Add(1)
				}
			}(servers[r%10])
		}
		close(start)
		wg.Wait()
		if in.Load() == 5 {
			exact++
		}
	}
	fmt.Printf("WATCH and MULTI, ten servers: %d trials of 50 at once, capacity 5\n", trials)
	fmt.Printf("  trials that let in exactly 5: %d of %d\n", exact, trials)
	// varies run to run, so it is not part of the recorded output
	fmt.Fprintf(os.Stderr, "attempts per request, average: %.1f\n", float64(attempts.Load())/float64(trials*50))
}
