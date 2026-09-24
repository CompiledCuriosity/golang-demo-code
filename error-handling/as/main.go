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
	if errors.Is(err, ErrNotFound) {
		fmt.Println(404, err)
	}
	var qe *QueryError
	if errors.As(err, &qe) {
		fmt.Println("table:", qe.Table)
	}
	_, direct := err.(*QueryError)
	fmt.Println(direct)
}
