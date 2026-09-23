package main

import "fmt"

type Role int

func main() {
	fromDB := 42
	var role Role = fromDB
	fmt.Println(role)
}
