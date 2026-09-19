// Command apiserver is the stand-in API the worker-pool programs call.
// It allows 25 requests at a time, takes 50 ms to answer one, and turns
// everything else away with a 429.
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

var slots = make(chan struct{}, 25)

func handle(w http.ResponseWriter, r *http.Request) {
	select {
	case slots <- struct{}{}:
		defer func() { <-slots }()
		time.Sleep(50 * time.Millisecond)
		fmt.Fprint(w, "ok")
	default:
		w.WriteHeader(http.StatusTooManyRequests)
	}
}

func main() {
	http.HandleFunc("/", handle)
	fmt.Println("stand-in API on :8080 - 25 at a time, 50 ms each, else 429")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
