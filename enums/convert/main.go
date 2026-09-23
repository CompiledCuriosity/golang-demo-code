package main

import "fmt"

type Role int

const (
	Unknown Role = iota
	Admin
	Editor
	Viewer
)

func (role Role) String() string {
	switch role {
	case Unknown:
		return "Unknown"
	case Admin:
		return "Admin"
	case Editor:
		return "Editor"
	case Viewer:
		return "Viewer"
	}
	return fmt.Sprintf("Role(%d)", int(role))
}

func main() {
	fromDB := 42
	role := Role(fromDB)
	fmt.Println(role)
}
