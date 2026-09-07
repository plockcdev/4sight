package main

import (
	"os"

	"foursight.dev/4sight/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
