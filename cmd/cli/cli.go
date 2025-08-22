package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"

	"github.com/rejot-dev/semcheck/internal/checker"
	"github.com/rejot-dev/semcheck/internal/color"
	"github.com/rejot-dev/semcheck/internal/config"
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
	logLevel     = flag.String("log-level", "info", "set log level (info, debug, error, warning)")
)

var (
	ErrorSemanticAnalysisFailed = errors.New("semantic analysis found issues")
)

const version = "1.2.0"

func Execute() error {
	flag.Parse()

	if *showHelp {
		showUsage()
		return nil
	}

	if *logLevel != "" {
		switch *logLevel {
		case "info":
			log.SetLevel(log.InfoLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "warning":
			log.SetLevel(log.WarnLevel)
		default:
			fmt.Fprintf(os.Stderr, "Invalid log level: %s\n", *logLevel)
			return errors.New("invalid log level")
		}

	}

	if *showVer {
		fmt.Printf("v%s\n", version)
		return nil
	}

	if *initConfig {
		return runInit()
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

	titleStyle := lipgloss.NewStyle().
		Italic(true).
		Bold(true).
		Foreground(color.White).
		BorderStyle(lipgloss.RoundedBorder()).
		Padding(0, 1).
		MarginTop(1).
		BorderBottom(true)

	fmt.Println(titleStyle.Render("🤖 Semcheck"))

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
	var selectedFiles []string = nil

	if len(files) > 0 && !*preCommit {
		selectedFiles = files
	} else if *preCommit {
		log.Info("Running semcheck on staged files...")
		selectedFiles = processor.GetStagedFiles(workingDir)
	}

	if selectedFiles == nil {
		log.Info("No file arguments passed, checking all implementation files against all specifications.")
		matchedResults = matcher.GetAllMatcherResults()
	} else {
		matchedResults, err = matcher.MatchFiles(selectedFiles)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error matching files: %v\n", err)
			return err
		}
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
		return ErrorSemanticAnalysisFailed
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
