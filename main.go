package main

import (
	"os"

	"github.com/lazynop/lazynf/cmd"
)

var version = "0.0.1-dev"

func main() {
	root := cmd.NewRoot(version)
	if err := root.Execute(); err != nil {
		os.Exit(cmd.Exit(err))
	}
}
