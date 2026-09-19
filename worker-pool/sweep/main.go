// Command sweep is pool with the worker count behind a flag and a timer,
// so one program produces the 8 / 25 / 50 worker comparison.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// call makes one GET and returns the status code.
func call(job int) int {
	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/?job=%d", job))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func main() {
	workers := flag.Int("workers", 25, "how many workers pull from the jobs channel")
	flag.Parse()

	start := time.Now()

	jobs := make(chan int, 1000)
	results := make(chan int)
	var wg sync.WaitGroup

	for range *workers {
		wg.Go(func() {
			for job := range jobs {
				results <- call(job)
			}
		})
	}

	for job := range 1000 {
		jobs <- job
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	rejected := 0
	for code := range results {
		if code != 200 {
			rejected++
		}
	}
	fmt.Printf("%d workers: %d rejected, %dms\n", *workers, rejected, time.Since(start).Milliseconds())
}
