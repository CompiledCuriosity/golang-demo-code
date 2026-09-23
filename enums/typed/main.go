package main

import "fmt"

type Role int

const (
	Admin Role = iota
	Editor
	Viewer
)

func grant(userID int, role Role) {
	fmt.Println("user", userID, "gets role", role)
}

func main() {
	id := 7
	grant(Editor, id)
}
