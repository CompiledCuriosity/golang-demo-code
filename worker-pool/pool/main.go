// Command pool makes the same 1000 calls through a worker pool of 25.
// Same work as flood, but only 25 calls are ever in flight, so nothing
// is rejected.
package main

import (
	"fmt"
	"net/http"
	"sync"
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
	jobs := make(chan int, 1000)
	results := make(chan int)
	var wg sync.WaitGroup

	for range 25 {
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
	fmt.Println(rejected, "of 1000 rejected")
}
