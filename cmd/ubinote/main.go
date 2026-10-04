package main

import (
	"os"

	"ubinote-cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
