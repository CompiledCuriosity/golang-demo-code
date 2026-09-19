// Command flood makes all 1000 calls at once, one goroutine per call.
// The API allows 25 at a time, so almost everything comes back rejected.
package main

import (
	"fmt"
	"net/http"
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
	codes := make(chan int)

	for job := range 1000 {
		go func() {
			codes <- call(job)
		}()
	}

	rejected := 0
	for range 1000 {
		if <-codes != 200 {
			rejected++
		}
	}
	fmt.Println(rejected, "of 1000 rejected")
}
