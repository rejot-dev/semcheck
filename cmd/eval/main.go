package main

import (
	"fmt"
	"os"
)

func main() {
	if err := RunEvaluation(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
