// Command cpuwork runs the same pool shape over work that burns a core
// instead of waiting on the wire: 400 jobs, 40,000 chained SHA-256 rounds
// each. The worker count is behind a flag, so you can watch it stop
// helping once you pass the core count.
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"sync"
	"time"
)

func hashJob(job int) [32]byte {
	sum := sha256.Sum256([]byte{byte(job)})
	for range 40000 {
		sum = sha256.Sum256(sum[:])
	}
	return sum
}

func main() {
	workers := flag.Int("workers", 1, "how many workers pull from the jobs channel")
	flag.Parse()

	start := time.Now()

	jobs := make(chan int, 400)
	results := make(chan [32]byte)
	var wg sync.WaitGroup

	for range *workers {
		wg.Go(func() {
			for job := range jobs {
				results <- hashJob(job)
			}
		})
	}

	for job := range 400 {
		jobs <- job
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for range results {
	}
	fmt.Printf("workers=%d: %dms\n", *workers, time.Since(start).Milliseconds())
}
