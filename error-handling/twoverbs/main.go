package main

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

func main() {
	flat := fmt.Errorf("profile 42: %v", ErrNotFound)
	wrapped := fmt.Errorf("profile 42: %w", ErrNotFound)
	fmt.Println(flat)
	fmt.Println(wrapped)
	fmt.Println(errors.Unwrap(flat))
	fmt.Println(errors.Unwrap(wrapped))
}
