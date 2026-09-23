package main

import "fmt"

const (
	Admin  = iota
	Editor = iota
	Viewer = iota
)

func main() {
	fmt.Println(Admin, Editor, Viewer)
}
