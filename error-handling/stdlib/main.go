//go:build go1.26

package main

import (
	"errors"
	"fmt"
	"strconv"
)

func main() {
	_, err := strconv.Atoi("forty")
	fmt.Println(errors.Is(err, strconv.ErrSyntax))
	numErr, ok := errors.AsType[*strconv.NumError](err)
	fmt.Println(numErr.Func, numErr.Num, ok)
}
