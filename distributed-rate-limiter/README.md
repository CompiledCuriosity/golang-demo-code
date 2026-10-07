# Distributed rate limiter: ten servers, one bucket, one race

Every program here is one that produced a number shown in the video. The limit is always the same token bucket: a capacity of 5 tokens (the most requests that can get in at once) and a refill rate of 1 token per second, refilled lazily: when a request arrives, the tokens owed since the last request are added, never past the capacity, and then one is taken if there is one. "Ten servers" are ten Redis clients, each with its own connection pool, released together; one client sends 50 requests at once, five to each server. Each program is one `main.go`.

**Most of these need a Redis on `localhost:6379`.** On a Mac: `brew install redis` and `brew services start redis`. Or with Docker: `docker run -d --name redis -p 6379:6379 redis:7`. Only `perserver` and `pinned` run without one.

Run any of them from the repo root:

```
go run ./distributed-rate-limiter/atomic
```

Go 1.25 or later builds everything. Programs that measure a race repeat it (200 trials of the burst, or 1,000 pairs), because a race does not come out the same way twice; the counts that move from run to run, and every timing, are printed to stderr.

## The programs, in the order the video walks them

| Folder | What it shows |
| --- | --- |
| `perserver` | The hook. Ten servers, each keeping its own bucket in its own memory: 50 of 50 let in. One bucket: 5 of 50. No Redis needed. |
| `pinned` | The shortcut "send each client to the same server". Pinned to one server, 50 at once: 5 in. Moved to another server a second later (a restart, or a new server joining): 5 more, where one shared bucket would have refilled just 1. No Redis needed. |
| `naive` | One bucket shared in Redis, but each server reads it (HMGET), works out the refill and the take in its own memory, and writes it back (HSET). **This one fails on purpose**: every one of 200 trials lets in more than 5; stderr shows it is 50, every time. |
| `lostupdate` | The same read-then-write, instrumented, all on stderr: how many tokens the bucket holds once all 50 got in (a median of 4, so it remembers one take of fifty), and how often every read came before any write (only about half the trials, so that is not the reason; writes worked out from old reads overwriting each other are). |
| `interleave` | The race, one step at a time, with the steps forced into the order that loses. On 1 token: A reads 1, B reads 1, both let their request in, both write 0. On a full bucket: A reads 5, B takes three (writes 4, 3, 2), A writes 4 from its old read: 4 in, the bucket remembers 1. Then the same with the script: A's script lets its request in, B's reads 0 and is refused. Then 1,000 real pairs at once on 1 token: with the script, both get in 0 times (stdout); with read-then-write, almost every time (stderr). **The read-then-write halves fail on purpose.** |
| `mutex` | The shortcut "add a lock". A Go lock on each server: more than 5 in, every trial (it is in each server's own memory; the other servers never see it). One lock shared by all ten gives exactly 5, but only because these ten servers live in one program; real servers do not share memory. **The per-server lock fails on purpose.** |
| `redislock` | The shortcut "keep the lock in Redis" (SET NX with an expiry, then read, work it out, write, DEL). It works, 5 in every trial, but every request makes at least four trips, and with 50 at once the average is about 70 trips and about 40 ms per request (stderr). |
| `decr` | The shortcut "Redis commands are atomic, just DECR the count". 50 at once on 5: 5 in. Two seconds later, 50 at once: 0 in, because nothing ever adds the tokens owed. The one-script bucket on the same two bursts: 5, then 2. |
| `watch` | The shortcut "use a transaction" (WATCH, read, work it out, MULTI/EXEC, start over when EXEC fails). It works, 5 in every trial, at about 16 attempts per request (stderr). |
| `atomic` | The fix the video draws. The whole refill-and-take is one Lua script that runs inside Redis, and Redis runs one script at a time, start to finish. Ten servers, 50 at once: exactly 5, in 200 of 200 trials. Per-request time with 50 at once is on stderr (under a millisecond). The script and `Allow` are exactly as they appear on screen. |
| `scripttime` | How long Redis itself spends running that script, from Redis's own command statistics (about 5 microseconds a call): the time every other command waits. It resets Redis's statistics (`CONFIG RESETSTAT`) first, so run it on a Redis you are not using for anything else. |
| `roundtrip` | The price of one shared bucket, all on stderr: the bucket in the server's own memory (tens of nanoseconds a call), read then write to Redis (two trips, tens of microseconds), and the one-script call (one trip, tens of microseconds), with Redis on the same machine. |

## About the numbers

- The script reads the current time from Redis (`redis.call('TIME')`), not from the server that calls it, so every server counts time on one clock.
- go-redis runs a script with EVALSHA and falls back to EVAL the first time a server sends it; after that, each call is one trip.
- The timings depend on the machine and on Redis running locally; the counts do not.
