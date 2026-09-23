package main

import "fmt"

type Role string

const (
	Admin  Role = "admin"
	Editor Role = "editor"
	Viewer Role = "viewer"
)

func main() {
	var unset Role
	fmt.Printf("%q\n", unset)
	fromDB := "banana"
	role := Role(fromDB)
	fmt.Println(role)
}
