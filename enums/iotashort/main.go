package main

import "fmt"

const (
	Admin = iota
	Editor
	Viewer
)

const (
	Free = iota
	Paid
)

func main() {
	fmt.Println(Admin, Editor, Viewer)
	fmt.Println(Free, Paid)
}
