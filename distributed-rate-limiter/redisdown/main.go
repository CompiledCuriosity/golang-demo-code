// redisdown: what the one-script limiter does when Redis cannot be reached.
// The same Allow as atomic, pointed at a port where no Redis is listening.
// The script call returns an error, Allow reads that as "not allowed", and
// every request is refused: the limiter fails closed.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const key = "bucket:client42"

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
	server := &Server{rdb: redis.NewClient(&redis.Options{Addr: "localhost:6390",
		DialTimeout: 200 * time.Millisecond, MaxRetries: -1})}
	in := 0
	for i := 0; i < 5; i++ {
		if server.Allow(ctx) {
			in++
		}
	}
	_, err := allow.Run(ctx, server.rdb, []string{key}).Int()
	fmt.Printf("Redis unreachable, 5 requests on a full bucket: %d in\n", in)
	fmt.Printf("the script call fails: %v\n", err != nil)
}
