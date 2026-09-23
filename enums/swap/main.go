package main

import "fmt"

const (
	Admin = iota
	Editor
	Viewer
)

func grant(userID int, role int) {
	fmt.Println("user", userID, "gets role", role)
}

func main() {
	id := 7
	grant(Editor, id)
}
