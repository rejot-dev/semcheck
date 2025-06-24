package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"rejot.dev/semcheck/internal/checker"
	"rejot.dev/semcheck/internal/config"
	"rejot.dev/semcheck/internal/processor"
	"rejot.dev/semcheck/internal/providers"
)

type EvalCase struct {
	Name           string
	ExpectedErrors int
	SpecFile       string
	ImplFile       string
}

func RunEvaluation() error {
	fmt.Println("Running semcheck evaluations...")

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load expectations from CSV
	expectations, err := loadExpectations(filepath.Join(workingDir, "evals", "expectations.csv"))
	if err != nil {
		return fmt.Errorf("failed to load expectations: %w", err)
	}

	// Load config
	cfg, err := config.Load(filepath.Join(workingDir, "evals", "eval-config.yaml"))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create AI client
	client, err := providers.CreateAIClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	// Initialize file matcher
	matcher, err := processor.NewMatcher(cfg, workingDir)
	if err != nil {
		return fmt.Errorf("failed to create matcher: %w", err)
	}

	// Collect all implementation files from rules
	var allFiles []string
	for _, rule := range cfg.Rules {
		allFiles = append(allFiles, rule.Files.Include...)
	}

	// Match files
	matchedResults, err := matcher.MatchFiles(allFiles)
	if err != nil {
		return fmt.Errorf("failed to match files: %w", err)
	}

	// Perform semantic analysis
	semanticChecker := checker.NewSemanticChecker(cfg, client, workingDir)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Second)
	defer cancel()

	checkResult, err := semanticChecker.CheckFiles(ctx, matchedResults)
	if err != nil {
		return fmt.Errorf("semantic analysis failed: %w", err)
	}

	// Compare results to expectations and display accuracy
	return compareAndDisplayResults(checkResult, expectations)
}

func loadExpectations(filePath string) (map[string]int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	expectations := make(map[string]int)
	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}
		if len(record) < 2 {
			continue
		}
		ruleName := record[0]
		expectedErrors, err := strconv.Atoi(record[1])
		if err != nil {
			return nil, fmt.Errorf("invalid expected_errors for rule %s: %v", ruleName, err)
		}
		expectations[ruleName] = expectedErrors
	}

	return expectations, nil
}

func compareAndDisplayResults(checkResult *checker.CheckResult, expectations map[string]int) error {
	fmt.Println("\n--- Evaluation Results ---")

	totalTests := len(expectations)
	passedTests := 0

	for ruleName, expectedErrors := range expectations {
		issues := checkResult.Issues[ruleName]

		// Count only ERROR level issues
		actualErrors := 0
		for _, issue := range issues {
			if strings.ToUpper(issue.Level) == "ERROR" {
				actualErrors++
			}
		}

		passed := actualErrors == expectedErrors
		if passed {
			passedTests++
		}

		status := "❌ FAIL"
		if passed {
			status = "✅ PASS"
		}

		fmt.Printf("%s %s: expected %d errors, got %d errors\n",
			status, ruleName, expectedErrors, actualErrors)

		if len(issues) > 0 {
			fmt.Printf("Issues:\n")
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue.Message)
			}
		}

	}

	fmt.Printf("\nSummary: %d/%d tests passed\n", passedTests, totalTests)

	if passedTests == totalTests {
		fmt.Println("✅ All evaluations passed!")
		return nil
	} else {
		return fmt.Errorf("evaluation failed: %d/%d tests passed", passedTests, totalTests)
	}
}
