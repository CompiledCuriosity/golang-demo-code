package main

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNotFound = errors.New("not found")

func main() {
	err := fmt.Errorf("profile 42: %v", ErrNotFound)
	other := errors.New("template not found")
	fmt.Println(strings.Contains(err.Error(), "not found"))
	fmt.Println(strings.Contains(other.Error(), "not found"))
}
