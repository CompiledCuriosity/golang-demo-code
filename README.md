# Go demo code

Runnable Go source for the videos on [Compiled Curiosity](https://www.youtube.com/@CompiledCuriosity).

Each video has its own folder holding the programs it walks through and a README that explains them in more detail than the video has room for. Every folder README links back to its video.

The repo is one Go module, so everything runs from the repo root:

```
go run ./<video>/<program>
```

Go 1.25 or later (the worker-pool code uses `sync.WaitGroup.Go`). Some programs deadlock or crash on purpose; the folder README says which ones and why.

## Videos

- [`concurrency-patterns`](concurrency-patterns) - fan-out and fan-in, the pipeline, and errgroup
- [`worker-pool`](worker-pool) - running only N jobs at once, and picking N
- [`pipeline-pattern`](pipeline-pattern) - stages in a line, the goroutine leak when the reader leaves early, and the fix
