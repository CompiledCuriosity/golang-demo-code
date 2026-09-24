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
	var queryErr *QueryError
	fmt.Println(errors.As(err, &queryErr))
	fmt.Println(errors.Is(err, ErrNotFound))
}
