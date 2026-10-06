# Rate limiter: the token bucket's two numbers

Every program here is one that produced a number or an output shown in the video. The bucket is always the same: a capacity of 5 tokens (the most requests that can get in at once) and a refill rate of 1 token per second (the most a client can average, sixty a minute), refilled lazily: when a request arrives, the tokens owed since the last request are added, never past the capacity, and then one is taken if there is one. Each program is one `main.go`.

Run any of them from the repo root:

```
go run ./rate-limiter/bucket
```

Go 1.25 or later builds everything. Every program runs on a fake clock (request times are fixed in the code), so the output is the same on every machine except where a program says it prints a timing.

## The programs, in the order the video walks them

| Folder | What it shows |
| --- | --- |
| `edge` | The hole the video opens on. A counter that resets every minute, limit sixty, lets in 60 requests at 59.5 s and 60 more at 60.5 s: 120 in one second. A bucket of capacity 5 lets in 6 (five at once, then the one token owed as the second passes); a bucket of capacity 60 lets in 61. Then the shortcuts: a per-second counter with a limit of one (two requests at 19.3 s: 1 in; 20.9 s and 21.1 s, either side of a second's edge: both in), and two counters stacked, five a second AND sixty a minute (five at 20.9 s and five at 21.1 s: 10 in within 0.2 s). A bucket of capacity 5 on those same ten requests lets in 5. |
| `bucket` | The bucket the video draws, with the ten-line `Allow` exactly as it appears on screen. Seventeen requests: seven at 0 s, two at 2.5 s, one at 2.8 s, seven at 12.8 s. It prints, for each request, the tokens it found, the tokens owed, the decision, and for a refusal the wait (the Retry-After value, rounded up to whole seconds). Last line: `allowed 12 of 17`. |
| `timer` | The shortcut the video rules out: a timer that adds a token to every bucket every second. A million clients, ten quiet seconds: 10,000,000 bucket writes, against 1 for lazy refill. The time a pass takes is printed to stderr; it is short, which is why the video never calls the timer slow, only work done for nobody. |
| `ratepkg` | The same seventeen requests through Go's own limiter, `golang.org/x/time/rate`, created with the same two numbers: `rate.NewLimiter(1, 5)`. The same decision on every request, `allowed 12 of 17`, and the bucket at 3.0 s, two tenths of a second after the refused request: 1.0 tokens. |
| `race` | The drawn `Allow`, without a lock, called from 8 goroutines at once on one bucket. `go run` prints a line and exits; `go run -race ./rate-limiter/race` reports a DATA RACE. **This one fails on purpose**: it is why the video says the ten lines need a lock before a real server shares them (the `rate` package already has one). |
| `longrun` | The bucket's promise. A request every 0.5 s for a closed minute (0 s to 60 s): 65 in (5 + 60); 64 for the half-open minute; 3,605 in an hour. An exact sliding window (a log of request times) facing 60 at one instant (60 in) and the hook's edge (60 in). Then the catch the video ends on: ten servers with a bucket each let in 50 at once and 650 a minute; split the limit ten ways and each bucket holds half a token, so nothing ever gets in; pin a client to one server, move it, and it gets a fresh, full bucket. |
| `leaky` | A leaky bucket used as a queue: room for ten, one let out per second, sixty requests at one instant. One goes out at once, ten wait in line (the last waits 10 s), 49 are refused, and never more than one is let out in any second. A token bucket of capacity 1 on the same sixty: 1 in, 59 refused. |

## About the numbers

- `bucket` and `ratepkg` make the same decision on every request. One small difference inside: the drawn `Allow` writes the tokens and the time even for a refused request, and `rate.Limiter.AllowN` does not; the decisions and the readings still match.
- `longrun` steps 0.5 s on purpose. Stepping 0.1 s lets in 64 instead of 65 over the minute, because 0.1 is not exact in binary floating point and one token is lost to rounding.
- The refill rate is a ceiling on the average, not the average: the client in `bucket` gets 12 requests in over 12.8 seconds.
