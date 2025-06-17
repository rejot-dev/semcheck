package main

import (
	"os"

	"rejot.dev/semcheck/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
