package main

import (
	"fmt"
	"os"

	"github.com/rejot-dev/semcheck/cmd/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
