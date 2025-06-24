package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"rejot.dev/semcheck/internal/checker"
	"rejot.dev/semcheck/internal/config"
	"rejot.dev/semcheck/internal/processor"
	"rejot.dev/semcheck/internal/providers"
)

var (
	configPath      = flag.String("config", "semcheck.yaml", "path to configuration file")
	showHelp        = flag.Bool("help", false, "show help message")
	showVer         = flag.Bool("version", false, "show version")
	showConfig      = flag.Bool("show-config", false, "print full configuration")
	includeAnalysis = flag.Bool("include-analysis", false, "Include additional analysis in results")
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

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return err
	}

	if *showConfig {
		cfg.PrintAsYAML()
		return nil
	}

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no files specified\n")
		showUsage()
		return fmt.Errorf("no files specified")
	}

	fmt.Printf("Processing %d files with config: %s\n", len(files), *configPath)
	fmt.Printf("Provider: %s\n", cfg.Provider)

	// Initialize file matcher
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		return err
	}

	matcher, err := processor.NewMatcher(cfg, workingDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file matcher: %v\n", err)
		return err
	}

	// Match files and show results
	matchedResults, err := matcher.MatchFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error matching files: %v\n", err)
		return err
	}

	processor.DisplayMatchResults(matchedResults)

	// Create AI client for semantic analysis
	client, err := providers.CreateAIClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating AI client: %v\n", err)
		return err
	}

	// Perform semantic analysis
	semanticChecker := checker.NewSemanticChecker(cfg, client, workingDir)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	checkResult, err := semanticChecker.CheckFiles(ctx, matchedResults)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Semantic analysis failed: %v\n", err)
		return err
	}

	// Display results
	reporter := checker.NewStdoutReporter(&checker.StdoutReporterOptions{
		ShowAnalysis: *includeAnalysis,
	})
	reporter.Report(checkResult)

	// Determine exit code based on results
	if checkResult.ShouldFail(cfg) {
		return fmt.Errorf("semantic analysis failed with errors")
	}

	return nil
}

func showUsage() {
	fmt.Printf("Usage: %s [options] <files...>\n\n", os.Args[0])
	fmt.Printf("Semcheck is a tool for semantic checking of code implementations against specifications using LLMs.\n\n")
	fmt.Println("Arguments:")
	fmt.Println("  <files...> - Files to check, either specifications or implementation files. Semcheck will use rules to determine which it is.")
	fmt.Println("Options:")
	flag.PrintDefaults()

}
