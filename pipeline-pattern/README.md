# Go pipeline pattern: stages, the goroutine leak, and the fix

Every program here is one that produced a number or an output shown in the video. The job is always the same: a first stage numbers twelve documents, a second stage hashes them, and `main` (or `preview`) reads the results. Each program is one `main.go`, and the drawn code is the same in all of them; the ones that differ say so below.

Related: [`concurrency-patterns`](../concurrency-patterns) draws the pipeline as one of three shapes, next to fan-out and errgroup.

Run any of them from the repo root:

```
go run ./pipeline-pattern/pipeline
```

Go 1.25 or later builds everything. `leakprofile` and `fixedprofile` read the `goroutineleak` profile, which arrived in Go 1.27, and they panic on an older toolchain. The timings in the video were measured on an eight-core Apple Silicon Mac with go1.27.1; yours will differ in absolute terms, and the two figures that matter (the plain loop against the line) should keep their shape rather than their digits.

## The programs, in the order the video walks them

| Folder | What it shows |
| --- | --- |
| `serial` | The shortcut: one plain loop, no goroutines, no channels. Twelve documents in about 255 ms (printed to stderr). |
| `clean` | The pipeline that works. Two stages, each owning and closing its own output channel, and `printAll` reading to the end. Goroutines: 1, then 1. |
| `cleantimed` | `clean`, timed. About 153 ms against the loop's 255: the first stage numbers the next document while the second hashes this one. |
| `cleanorder` | `clean` with a deferred print in each stage, proving the order the line shuts down in: `ids`, then `hashes`, then the reader. |
| `pipeline` | The leak. `preview` reads three results and returns. Goroutines after it: 3, main and two stuck. |
| `leakwhich` | `pipeline`, instrumented: which envelope each stuck stage holds out (`hashes` holds doc 4, `ids` holds id 5), and that four of twelve were hashed. |
| `leakwait` | `pipeline`, then waiting 500 ms, 5 s and 30 s and counting again. Still 3 every time: not a timing accident. Runs about 36 s. |
| `leakprofile` | `pipeline`, then Go's own `goroutineleak` profile: total 2, both records pointing at a send (Go 1.27). |
| `leakloop` | `preview` called a thousand times. 2001 goroutines, and the stack in use grows from about 352 KB to about 8,576 KB (to stderr). Runs about 50 s. |
| `drain` | Shortcut one: keep reading and throw the rest away. No leak, and all twelve documents hashed to use three. |
| `pipebuf` | Shortcut two: buffer both channels for the whole batch. No leak, and all twelve hashed again. |
| `pipebufsmall` | The same with buffers of 4, backing the line "a buffer only delays the wait": one goroutine still parked. |
| `loopthree` | A plain loop that returns after three. No goroutines, nothing to leak, about 77 ms to the three results. |
| `previewtimed` | The unfixed line, timed to the same three results: about 61 ms. |
| `pipecancel` | The fix. Every send selects on `ctx.Done()`, and the reader defers a cancel. Goroutines after: 1. Four of twelve hashed. |
| `pipedone` | The same fix with a done channel made by hand instead of a context. The same result, because it is the same mechanism. |
| `fixedprofile` | `pipecancel`, then the same leak profile: total 0 (Go 1.27). |
| `fixedloop` | The thousand calls again, against the fixed `preview`. 1 goroutine, stack about 544 KB. Runs about 50 s. |
| `fixedtimed` | The fixed line read to the end and timed, kept as evidence that the fix does not slow the line. |

None of these fails on purpose except by leaking: nothing here crashes or deadlocks, which is the point. The leak prints no error, and the only way to know is to count.

## About the numbers

- `pipeline` reports one live goroutine at the start and three after `preview` returns. `pipecancel` reports one at both points.
- `pipecancel` hashes four of twelve documents: the three that were read, plus the one the second stage had already taken.
- Any change to a program's code can move its timings by about 16 ms, because this workload has two timing modes that code layout picks between. Compare programs from one run of one machine, interleaved, and take medians.
