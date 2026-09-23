package main

import "fmt"

type Role int

const (
	Unknown Role = iota
	Admin
	Editor
	Viewer
)

func (role Role) IsValid() bool {
	return role >= Admin && role <= Viewer
}

func main() {
	fromDB := 42
	role := Role(fromDB)
	fmt.Println(role.IsValid())

	var unset Role
	fmt.Println(unset.IsValid())

	fmt.Println(Editor.IsValid())
}
