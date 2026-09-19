# Go worker pools: running only N jobs at once

Source for [Golang Worker Pool: How Do You Run Only N Jobs At Once?](https://youtu.be/ztSsFJQyM28).

Every program here is the one that produced the output shown in the video. The job is always the same: make a thousand calls to an API that only allows twenty-five at a time, and count how many come back rejected.

Related: [`concurrency-patterns`](../concurrency-patterns) builds the same fan-out shape around a different question, and its `noclose` program is this folder's deadlock in mirror image - there the missing close is on the results channel, not the jobs channel.

## Start the API first

Five of these programs call a stand-in API, so it has to be running before you start them. In one terminal:

```
go run ./worker-pool/apiserver
```

It listens on `:8080`, allows 25 requests at a time, takes 50 ms to answer one, and replies 429 to everything else. Leave it running and use a second terminal for the programs below.

## The programs, in the order the video walks them

| Folder | What it shows |
| --- | --- |
| `apiserver` | The stand-in API. Twenty-five slots held in a buffered channel, and a `select` with a `default` that turns everyone else away. |
| `flood` | One goroutine per call, all thousand at once. Prints `975 of 1000 rejected`. |
| `pool` | The pattern itself. Twenty-five workers ranging over one jobs channel. Same thousand calls, `0 of 1000 rejected`. |
| `nojobsclose` | `pool` with `close(jobs)` deleted. Every job finishes and then the program hangs and dies. |
| `nojobsclose-verify` | The same bug with a counter in the harvest loop, to prove the work really does all finish first. |
| `cpuwork` | The same pool shape over work that burns a core instead of waiting on the wire. `-workers` decides how many. |
| `sweep` | `pool` with `-workers` and a timer, so one program produces the 8 / 25 / 50 comparison. |

`cpuwork` and `sweep` take a flag:

```
go run ./worker-pool/cpuwork -workers 8
go run ./worker-pool/sweep -workers 25
```

## The two that are meant to fail

`nojobsclose` and `nojobsclose-verify` both end in `fatal error: all goroutines are asleep - deadlock!` and exit with status 2. That is the point of them.

Nothing about the work goes wrong. All thousand calls are made and all thousand results are collected, which is what `nojobsclose-verify` prints before it dies:

```
received 500
received 1000
fatal error: all goroutines are asleep - deadlock!
```

What goes wrong is the ending. Without `close(jobs)` the workers drain the channel and then turn back for more, and a range over an open channel waits. So twenty-five workers wait on an empty jobs channel, the helper goroutine waits on the WaitGroup, main waits on the results channel, and nothing is left to wake anyone. The runtime notices that every goroutine is asleep and pulls the plug.

The line just below the fatal error is the one worth reading, because it names who was stuck where:

```
goroutine 1 [chan receive]:   main, parked on the results channel
```

Worth knowing: that crash only happens because nothing else in the process was alive. In a real server other goroutines keep running, so there is no fatal error and no stack dump. The workers just leak, quietly, for as long as the process lives.

## About the numbers

Measured on the machine that recorded the video, an eight-core Apple Silicon Mac running go1.26.2, with the stand-in API running locally. Yours will differ in absolute terms; what should hold is the shape.

The rejected counts are not timing-dependent. `flood` loses exactly the 975 calls that did not grab one of the 25 slots, every run.

CPU-bound work, 400 jobs of 40,000 chained SHA-256 rounds:

| workers | time |
| --- | --- |
| 1 | 848 ms |
| 8 | 164 ms |
| 100 | 162 ms |

Eight is this machine's `runtime.NumCPU()`. Going from one worker to eight is roughly five times faster; going from eight to a hundred buys nothing at all, because eight cores still only run eight computations at a time and the other ninety-two workers queue.

Network-bound work, the same thousand calls through `sweep`:

| workers | result |
| --- | --- |
| 8 | 0 rejected, 6618 ms |
| 25 | 0 rejected, 2164 ms |
| 50 | 975 rejected, 55 ms |

Eight workers leave seventeen of the API's twenty-five slots idle the whole run. Twenty-five matches the limit and is three times faster with nothing rejected. Fifty "finishes" in 55 ms only by throwing away most of the work: a rejection comes back in about a millisecond, so the extra workers burn through the queue collecting 429s before a slot ever frees.

That is the difference between the two kinds of N. For CPU-bound jobs the number comes from your machine, and `runtime.NumCPU()` is the answer. For network jobs it comes from the other end of the wire: the published rate limit, the connection cap on the database. Here it was twenty-five.

## One name to clear away

`sync.Pool` is a different thing that happens to have a similar name. It recycles allocations to take pressure off the garbage collector; it does not run jobs and it has no worker count. Nothing in this folder uses it.
