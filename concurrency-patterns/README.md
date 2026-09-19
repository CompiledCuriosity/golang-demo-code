# Go concurrency patterns: fan-out, pipeline, errgroup

Source for [Go Concurrency Patterns in 11 Minutes: Fan-Out, Pipeline, errgroup](https://youtu.be/CxrEkqyvf5c).

Every program here is the one that produced the output shown in the video. The job is always the same: hash twelve documents, where hashing one document means a million chained SHA-256 rounds, so the work is real and the timings mean something.

Related: [`worker-pool`](../worker-pool) takes the same fan-out shape and asks how many workers there should be, and how to measure the answer.

Run any of them from the repo root:

```
go run ./concurrency-patterns/fanout
```

## The programs, in the order the video walks them

| Folder | What it shows |
| --- | --- |
| `naive` | One goroutine per document and no waiting. Prints `all 12 done` and not one result. |
| `serial` | The baseline. Twelve documents, one at a time. |
| `fanout` | Fan-out and fan-in. One jobs channel, four workers, one results channel, and a closer goroutine that waits on the WaitGroup. |
| `noclose` | The first deadlock. Nobody closes `results`, so main parks forever on a channel receive. |
| `inlineclose` | The second deadlock. Main does the waiting itself, so it is not reading, so the workers block sending and the wait never returns. |
| `pipeline` | Stages in a line, each owning and closing its own output channel. Main takes three of twelve results and leaks two goroutines. |
| `pipecancel` | The same pipeline with a context. Every send selects on the cancel, and the leak is gone. |
| `errgroup` | `SetLimit(4)` for the bound and `WithContext` for the first error that cancels the rest. |

`noclose` and `inlineclose` are meant to fail. Each one ends in `fatal error: all goroutines are asleep - deadlock!` and exits with status 2. The interesting part is the goroutine state the runtime prints just below that line, because it is different in the two cases and it is what tells them apart:

```
goroutine 1 [chan receive]:          noclose
goroutine 1 [sync.WaitGroup.Wait]:   inlineclose
```

## About the numbers

The timings in the video were measured on the machine that recorded it, an eight-core Apple Silicon Mac. Yours will differ in absolute terms. What should hold is the shape: four workers land near three times faster than serial rather than four times, because each document takes longer when four are hashing at once.

`pipeline` reports one live goroutine at the start and three after main stops reading. `pipecancel` reports one at both points. `errgroup` returns `doc 7: checksum mismatch` with six of twelve documents hashed, because the first error cancels the five that had not started.
