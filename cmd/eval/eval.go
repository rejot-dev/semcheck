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

	"github.com/rejot-dev/semcheck/internal/checker"
	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/processor"
	"github.com/rejot-dev/semcheck/internal/providers"
)

type EvalCase struct {
	Name             string
	ExpectedErrors   int
	ExpectedWarnings int
	ExpectedInfo     int
	SpecFile         string
	ImplFile         string
}

type SeverityCount struct {
	Errors   int
	Warnings int
	Info     int
}

type EvalScore struct {
	TotalTests      int
	PassedTests     int
	ErrorAccuracy   float64
	WarningAccuracy float64
	InfoAccuracy    float64
	OverallScore    float64
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

	checkResult, err := semanticChecker.CheckFiles(ctx, matchedResults, matcher)
	if err != nil {
		return fmt.Errorf("semantic analysis failed: %w", err)
	}

	// Compare results to expectations and display accuracy
	return compareAndDisplayResults(cfg, checkResult, expectations)
}

func loadExpectations(filePath string) (map[string]SeverityCount, error) {
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

	expectations := make(map[string]SeverityCount)
	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}
		if len(record) < 4 {
			continue
		}
		ruleName := record[0]

		expectedErrors, err := strconv.Atoi(record[1])
		if err != nil {
			return nil, fmt.Errorf("invalid expected_errors for rule %s: %v", ruleName, err)
		}

		expectedWarnings, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, fmt.Errorf("invalid expected_warnings for rule %s: %v", ruleName, err)
		}

		expectedInfo, err := strconv.Atoi(record[3])
		if err != nil {
			return nil, fmt.Errorf("invalid expected_info for rule %s: %v", ruleName, err)
		}

		expectations[ruleName] = SeverityCount{
			Errors:   expectedErrors,
			Warnings: expectedWarnings,
			Info:     expectedInfo,
		}
	}

	return expectations, nil
}

func compareAndDisplayResults(cfg *config.Config, checkResult *checker.CheckResult, expectations map[string]SeverityCount) error {
	fmt.Println("\n--- Evaluation Results ---")

	var totalErrorAccuracy, totalWarningAccuracy, totalInfoAccuracy float64
	totalTests := len(expectations)
	passedTests := 0

	for ruleName, expected := range expectations {
		issues := checkResult.Issues[ruleName]

		// Count issues by severity level
		actual := countIssuesBySeverity(issues)

		// Calculate accuracy for each severity level
		errorAccuracy := calculateAccuracy(expected.Errors, actual.Errors)
		warningAccuracy := calculateAccuracy(expected.Warnings, actual.Warnings)
		infoAccuracy := calculateAccuracy(expected.Info, actual.Info)

		// Test passes if all severity levels are exactly correct
		passed := (actual.Errors == expected.Errors &&
			actual.Warnings == expected.Warnings &&
			actual.Info == expected.Info)

		if passed {
			passedTests++
		}

		status := "❌ FAIL"
		if passed {
			status = "✅ PASS"
		}

		fmt.Printf("%s %s:\n", status, ruleName)
		fmt.Printf("    Errors:   expected %d, got %d (%.1f%% accuracy)\n",
			expected.Errors, actual.Errors, errorAccuracy*100)
		fmt.Printf("    Warnings: expected %d, got %d (%.1f%% accuracy)\n",
			expected.Warnings, actual.Warnings, warningAccuracy*100)
		fmt.Printf("    Info:     expected %d, got %d (%.1f%% accuracy)\n",
			expected.Info, actual.Info, infoAccuracy*100)

		if len(issues) > 0 {
			fmt.Printf("    Issues found:\n")
			for _, issue := range issues {
				fmt.Printf("      - (%s) %s", issue.Level, issue.Message)
				fmt.Printf("\n")
			}
		}
		fmt.Printf("\n")

		// Accumulate accuracy scores
		totalErrorAccuracy += errorAccuracy
		totalWarningAccuracy += warningAccuracy
		totalInfoAccuracy += infoAccuracy
	}

	// Calculate overall scores
	score := EvalScore{
		TotalTests:      totalTests,
		PassedTests:     passedTests,
		ErrorAccuracy:   totalErrorAccuracy / float64(totalTests),
		WarningAccuracy: totalWarningAccuracy / float64(totalTests),
		InfoAccuracy:    totalInfoAccuracy / float64(totalTests),
	}
	score.OverallScore = (score.ErrorAccuracy + score.WarningAccuracy + score.InfoAccuracy) / 3

	// Display final scores
	fmt.Printf("=== EVALUATION SUMMARY ===\n")
	fmt.Printf("Provider: %s\n", cfg.Provider)
	fmt.Printf("Model: %s\n", cfg.Model)
	fmt.Printf("Tests Passed: %d/%d (%.1f%%)\n", passedTests, score.TotalTests, float64(score.PassedTests)/float64(score.TotalTests)*100)
	fmt.Printf("Error Accuracy: %.1f%%\n", score.ErrorAccuracy*100)
	fmt.Printf("Warning Accuracy: %.1f%%\n", score.WarningAccuracy*100)
	fmt.Printf("Info Accuracy: %.1f%%\n", score.InfoAccuracy*100)
	fmt.Printf("Overall Score: %.1f%%\n", score.OverallScore*100)

	return nil
}

func countIssuesBySeverity(issues []providers.SemanticIssue) SeverityCount {
	count := SeverityCount{}
	for _, issue := range issues {
		switch strings.ToUpper(issue.Level) {
		case "ERROR":
			count.Errors++
		case "WARNING":
			count.Warnings++
		case "INFO":
			count.Info++
		}
	}
	return count
}

func calculateAccuracy(expected, actual int) float64 {
	if expected == 0 && actual == 0 {
		return 1.0 // Perfect when both are zero
	}
	if expected == 0 {
		return 0.0 // Failed when expected zero but got some
	}

	// Calculate accuracy based on how close actual is to expected
	diff := abs(expected - actual)
	maxError := max(expected, actual)
	accuracy := 1.0 - float64(diff)/float64(maxError)

	if accuracy < 0 {
		accuracy = 0
	}
	return accuracy
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
