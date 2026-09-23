package main

import "fmt"

const (
	Admin  = 0
	Editor = 1
	Viewer = 2
)

func main() {
	role := 0
	if role == Admin {
		fmt.Println("can delete")
	}
	fmt.Println(Admin, Editor, Viewer)
}
