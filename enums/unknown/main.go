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

type User struct {
	Name string
	Role Role
}

func main() {
	user := User{Name: "sam"}
	fmt.Println(user.Name, user.Role)
}
