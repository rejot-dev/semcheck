package cmd

import (
	"flag"
	"fmt"
	"os"

	"rejot.dev/semcheck/internal/config"
)

var (
	configPath = flag.String("config", "semcheck.yaml", "path to configuration file")
	showHelp   = flag.Bool("help", false, "show help message")
	showVer    = flag.Bool("version", false, "show version")
)

const version = "0.1.0"

func Execute() error {
	flag.Parse()

	if *showHelp {
		showUsage()
		return nil
	}

	if *showVer {
		fmt.Printf("semcheck version %s\n", version)
		return nil
	}

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no files specified\n")
		showUsage()
		return fmt.Errorf("no files specified")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return err
	}

	fmt.Printf("Processing %d files with config: %s\n", len(files), *configPath)
	fmt.Printf("Provider: %s\n", cfg.Provider)
	for _, file := range files {
		fmt.Printf("  - %s\n", file)
	}

	return nil
}

func showUsage() {
	fmt.Printf("Usage: %s [options] <files...>\n\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
}