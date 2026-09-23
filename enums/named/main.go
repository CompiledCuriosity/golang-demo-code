package main

import "fmt"

type Role int

const (
	Admin Role = iota
	Editor
	Viewer
)

func (role Role) String() string {
	switch role {
	case Admin:
		return "Admin"
	case Editor:
		return "Editor"
	case Viewer:
		return "Viewer"
	}
	return fmt.Sprintf("Role(%d)", int(role))
}

func grant(userID int, role Role) {
	fmt.Println("user", userID, "gets role", role)
}

func main() {
	id := 7
	grant(id, Editor)
}
