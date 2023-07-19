package main

import (
	"fmt"
	"os"
)

func main() {
	os.Exit(0) // want "os.Exit used in function 'main' of package 'main'"
}

func demo() {
	fmt.Println("hello, world")
}
