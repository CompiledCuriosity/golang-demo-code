// scripttime: how long Redis spends running the one script, by its own count.
// Resets Redis's command statistics, runs the script 10,000 times, and reads
// back the microseconds per call that Redis records for EVALSHA. This is the
// time every other command waits while one script runs.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
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

func main() {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Fprintln(os.Stderr, "redis:", err)
		os.Exit(1)
	}
	allow.Load(ctx, rdb)
	rdb.ConfigResetStat(ctx)
	for i := 0; i < 10_000; i++ {
		allow.Run(ctx, rdb, []string{"bucket:client42"})
	}
	for _, line := range strings.Split(rdb.Info(ctx, "commandstats").Val(), "\n") {
		if strings.HasPrefix(line, "cmdstat_evalsha:") {
			// varies run to run, so it is not part of the recorded output
			fmt.Fprintln(os.Stderr, strings.TrimSpace(line))
		}
	}
}
