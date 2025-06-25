package main

import (
	"os"

	"github.com/rejot-dev/semcheck/cmd/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
