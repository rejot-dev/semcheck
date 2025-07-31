package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/rejot-dev/semcheck/cmd/cli"
)

func init() {
	// Configure log format without timestamps
	log.SetTimeFormat("")
	log.SetStyles(log.DefaultStyles())
	// Set appropriate log level - debug messages are hidden by default
	log.SetLevel(log.InfoLevel)
}

func main() {
	if err := cli.Execute(); err != nil {
		log.Error("Command failed", "err", err)
		os.Exit(1)
	}
}
