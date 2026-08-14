package main

import (
	"os"

	"github.com/BeardedTech0o/tether/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
