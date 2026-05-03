package main

import (
	"fmt"
	"os"
)

var version = "0.0.1-dev"

func main() {
	fmt.Println("vellum", version)
	_ = os.Args
}
