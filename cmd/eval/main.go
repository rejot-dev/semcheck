package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	casesFlag = flag.String("cases", "", "comma-separated list of specific cases to run (e.g., case1,case2,case3)")
	helpFlag  = flag.Bool("help", false, "show help message")
)

func main() {
	flag.Parse()

	if *helpFlag {
		showUsage()
		return
	}

	var specificCases []string
	if *casesFlag != "" {
		specificCases = strings.Split(*casesFlag, ",")
		for i, c := range specificCases {
			specificCases[i] = strings.TrimSpace(c)
		}
	}

	if err := RunEvaluation(specificCases); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func showUsage() {
	fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
	fmt.Printf("Semcheck evals.\n\n")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("\nNote: When running specific cases with --cases, results are not recorded to results.csv")
}
