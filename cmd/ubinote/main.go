package main

import (
	"os"

	"github.com/popey17/Ubinote_cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
