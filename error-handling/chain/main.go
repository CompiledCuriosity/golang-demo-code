package main

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

type QueryError struct {
	Table string
	Err   error
}

func (e *QueryError) Error() string {
	return e.Table + ": " + e.Err.Error()
}

func (e *QueryError) Unwrap() error {
	return e.Err
}

func findUser(id int) error {
	return &QueryError{"users", ErrNotFound}
}

func loadProfile(id int) error {
	err := findUser(id)
	if err != nil {
		return fmt.Errorf("profile %d: %w", id, err)
	}
	return nil
}

func main() {
	err := loadProfile(42)
	fmt.Println(err)
	link := err
	for link != nil {
		fmt.Printf("%T\n", link)
		link = errors.Unwrap(link)
	}
}
