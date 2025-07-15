package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rejot-dev/semcheck/internal/checker"
	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/mcp"
	"github.com/rejot-dev/semcheck/internal/processor"
	"github.com/rejot-dev/semcheck/internal/providers"
)

var (
	configPath   = flag.String("config", "semcheck.yaml", "path to configuration file")
	showHelp     = flag.Bool("help", false, "show help message")
	showVer      = flag.Bool("version", false, "show version")
	showConfig   = flag.Bool("show-config", false, "print full configuration")
	hideAnalysis = flag.Bool("hide-analysis", false, "hide additional analysis in results")
	preCommit    = flag.Bool("pre-commit", false, "Runs semcheck on staged files")
	initConfig   = flag.Bool("init", false, "create a semcheck.yaml file interactively")
	githubOutput = flag.Bool("github-output", false, "output GitHub Actions annotations")
	mcpServer    = flag.Bool("mcp-server", false, "start MCP server mode")
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

	if *initConfig {
		return runInit()
	}

	if *mcpServer {
		return runMCPServer()
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return err
	}

	if *showConfig {
		if err := cfg.PrintAsYAML(); err != nil {
			return fmt.Errorf("failed to print config: %w", err)
		}
		return nil
	}
	fmt.Printf("Provider: %s, Model: %s\n", cfg.Provider, cfg.Model)

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

	files := flag.Args()
	var matchedResults []processor.MatcherResult
	if len(files) > 0 && !*preCommit {
		matchedResults, err = matcher.MatchFiles(files)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error matching files: %v\n", err)
			return err
		}
	} else if *preCommit {
		fmt.Println("Running semcheck on staged files...")
		stagedFiles := matcher.GetStagedFiles()
		matchedResults, err = matcher.MatchFiles(stagedFiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error matching staged files: %v\n", err)
			return err
		}
	} else {
		fmt.Println("No file arguments passed, checking all implementation files against all specifications.")
		matchedResults = matcher.GetAllMatcherResults()
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

	checkResult, err := semanticChecker.CheckFiles(ctx, matchedResults, matcher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Semantic analysis failed: %v\n", err)
		return err
	}

	// Display results
	var reporter checker.Reporter
	if *githubOutput {
		reporter = checker.NewGitHubReporter(&checker.GitHubReporterOptions{
			ShowAnalysis: !*hideAnalysis,
			WorkingDir:   workingDir,
		})
	} else {
		reporter = checker.NewStdoutReporter(&checker.StdoutReporterOptions{
			ShowAnalysis: !*hideAnalysis,
		})
	}
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

func runMCPServer() error {
	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	// Check if MCP is configured
	if cfg.MCP == nil || !cfg.MCP.Enabled {
		return fmt.Errorf("MCP server mode requires MCP configuration in config file")
	}

	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %w", err)
	}

	// Create tools/resources handler
	handler := mcp.NewToolsResourcesHandler(cfg, workingDir)

	// Create and start MCP server
	server := mcp.NewServer(cfg.MCP.Address, cfg.MCP.Port, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("MCP server is running. Press Ctrl+C to stop.")
	<-sigChan

	fmt.Println("Shutting down MCP server...")
	return server.Stop()
}
