package main

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

func findUser(id int) error {
	return ErrNotFound
}

func loadProfile(id int) error {
	err := findUser(id)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := loadProfile(42)
	if err == ErrNotFound {
		fmt.Println(404, err)
		return
	}
	fmt.Println(500, err)
}
