// Command nojobsclose-verify is nojobsclose with a counter in the harvest
// loop, to prove that the work all finishes before the deadlock.
//
// This program is MEANT TO FAIL. It prints "received 500" and
// "received 1000" and then dies the same way nojobsclose does: the results
// all arrive, and only then does everything park.
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
	// close(jobs) still missing, on purpose.

	go func() {
		wg.Wait()
		close(results)
	}()

	rejected, received := 0, 0
	for code := range results {
		if code != 200 {
			rejected++
		}
		received++
		if received%500 == 0 {
			fmt.Println("received", received)
		}
	}
	fmt.Println(rejected, "of 1000 rejected")
}
