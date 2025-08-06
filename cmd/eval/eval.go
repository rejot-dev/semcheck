package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rejot-dev/semcheck/internal/checker"
	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/processor"
	"github.com/rejot-dev/semcheck/internal/providers"
)

type EvalCase struct {
	Name             string
	ExpectedErrors   int
	ExpectedWarnings int
	ExpectedNotice   int
	SpecFile         string
	ImplFile         string
}

type SeverityCount struct {
	Errors   int
	Warnings int
	Notice   int
}

type EvalScore struct {
	TotalTests      int
	PassedTests     int
	ErrorAccuracy   float64
	WarningAccuracy float64
	NoticeAccuracy  float64
	OverallScore    float64
	RuleResults     map[string]RuleResult
}

type RuleResult struct {
	RuleName        string
	Expected        SeverityCount
	Actual          SeverityCount
	ErrorAccuracy   float64
	WarningAccuracy float64
	NoticeAccuracy  float64
	Passed          bool
	Issues          []providers.SemanticIssue
}

type EvalResult struct {
	CommitSHA       string
	Date            string
	Model           string
	Provider        string
	ErrorAccuracy   float64
	WarningAccuracy float64
	InfoAccuracy    float64
	TotalAccuracy   float64
	NumCases        int
	Duration        time.Duration
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
}

func RunEvaluation(specificCases []string) error {
	log.Info("Running semcheck evaluations...")
	startTime := time.Now()

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

	// Filter rules if specific cases are requested
	if len(specificCases) > 0 {
		filteredRules := make([]config.Rule, 0)
		caseSet := make(map[string]bool)
		for _, caseName := range specificCases {
			caseSet[caseName] = true
		}

		for _, rule := range cfg.Rules {
			if caseSet[rule.Name] {
				filteredRules = append(filteredRules, rule)
			}
		}

		if len(filteredRules) == 0 {
			return fmt.Errorf("no matching cases found for: %v", specificCases)
		}

		cfg.Rules = filteredRules
		fmt.Printf("Running specific cases: %v\n", specificCases)
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

	// Filter expectations to match the cases we're running
	if len(specificCases) > 0 {
		filteredExpectations := make(map[string]SeverityCount)
		for _, caseName := range specificCases {
			if exp, exists := expectations[caseName]; exists {
				filteredExpectations[caseName] = exp
			}
		}
		if len(filteredExpectations) > 0 {
			expectations = filteredExpectations
		}
	}

	// Compare results to expectations
	score, err := compareResults(checkResult, expectations)

	if err != nil {
		return fmt.Errorf("failed to compare results: %w", err)
	}

	// Display results
	displayResults(cfg, score, checkResult.TotalUsage)

	// Record results to CSV only if running all cases
	totalDuration := time.Since(startTime)
	// Subtract artificial delay time from total duration to get actual inference time
	delayPerRule := time.Duration(*cfg.InferenceDelay) * time.Second
	totalDelayTime := delayPerRule * time.Duration(checkResult.Processed)
	actualInferenceDuration := totalDuration - totalDelayTime
	if len(specificCases) == 0 {
		return recordResults(cfg, score, actualInferenceDuration, checkResult.TotalUsage)
	} else {
		fmt.Printf("\n➡️  Results not recorded (subset of cases selected)\n")
		return nil
	}
}

func loadExpectations(filePath string) (map[string]SeverityCount, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
		}
	}()

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

		expectedNotice, err := strconv.Atoi(record[3])
		if err != nil {
			return nil, fmt.Errorf("invalid expected_notice for rule %s: %v", ruleName, err)
		}

		expectations[ruleName] = SeverityCount{
			Errors:   expectedErrors,
			Warnings: expectedWarnings,
			Notice:   expectedNotice,
		}
	}

	return expectations, nil
}

func compareResults(checkResult *checker.CheckResult, expectations map[string]SeverityCount) (*EvalScore, error) {
	var totalErrorAccuracy, totalWarningAccuracy, totalNoticeAccuracy float64
	totalTests := len(expectations)
	passedTests := 0
	ruleResults := make(map[string]RuleResult)

	for ruleName, expected := range expectations {
		issues := checkResult.Issues[ruleName]

		// Count issues by severity level
		actual := countIssuesBySeverity(issues)

		// Calculate accuracy for each severity level
		errorAccuracy := calculateAccuracy(expected.Errors, actual.Errors)
		warningAccuracy := calculateAccuracy(expected.Warnings, actual.Warnings)
		noticeAccuracy := calculateAccuracy(expected.Notice, actual.Notice)

		// Test passes if all severity levels are exactly correct
		passed := (actual.Errors == expected.Errors &&
			actual.Warnings == expected.Warnings &&
			actual.Notice == expected.Notice)

		if passed {
			passedTests++
		}

		// Store rule result
		ruleResults[ruleName] = RuleResult{
			RuleName:        ruleName,
			Expected:        expected,
			Actual:          actual,
			ErrorAccuracy:   errorAccuracy,
			WarningAccuracy: warningAccuracy,
			NoticeAccuracy:  noticeAccuracy,
			Passed:          passed,
			Issues:          issues,
		}

		// Accumulate accuracy scores
		totalErrorAccuracy += errorAccuracy
		totalWarningAccuracy += warningAccuracy
		totalNoticeAccuracy += noticeAccuracy
	}

	// Calculate overall scores
	score := EvalScore{
		TotalTests:      totalTests,
		PassedTests:     passedTests,
		ErrorAccuracy:   totalErrorAccuracy / float64(totalTests),
		WarningAccuracy: totalWarningAccuracy / float64(totalTests),
		NoticeAccuracy:  totalNoticeAccuracy / float64(totalTests),
		RuleResults:     ruleResults,
	}
	score.OverallScore = (score.ErrorAccuracy + score.WarningAccuracy + score.NoticeAccuracy) / 3

	return &score, nil
}

func displayResults(cfg *config.Config, score *EvalScore, usage providers.Usage) {
	log.Info("--- Evaluation Results ---")

	for _, result := range score.RuleResults {
		status := "❌ FAIL"
		if result.Passed {
			status = "✅ PASS"
		}

		fmt.Printf("%s %s:\n", status, result.RuleName)
		fmt.Printf("    Errors:   expected %d, got %d (%.1f%% accuracy)\n",
			result.Expected.Errors, result.Actual.Errors, result.ErrorAccuracy*100)
		fmt.Printf("    Warnings: expected %d, got %d (%.1f%% accuracy)\n",
			result.Expected.Warnings, result.Actual.Warnings, result.WarningAccuracy*100)
		fmt.Printf("    Notice:   expected %d, got %d (%.1f%% accuracy)\n",
			result.Expected.Notice, result.Actual.Notice, result.NoticeAccuracy*100)

		if len(result.Issues) > 0 {
			fmt.Printf("    Issues found:\n")
			for _, issue := range result.Issues {
				fmt.Printf("      - (%s) %s", issue.Level, issue.Message)
				fmt.Printf("\n")
			}
		}
		fmt.Printf("\n")
	}

	// Display final scores
	fmt.Printf("=== EVALUATION SUMMARY ===\n")
	fmt.Printf("Provider: %s\n", cfg.Provider)
	fmt.Printf("Model: %s\n", cfg.Model)
	fmt.Printf("Tests Passed: %d/%d (%.1f%%)\n", score.PassedTests, score.TotalTests, float64(score.PassedTests)/float64(score.TotalTests)*100)
	fmt.Printf("Error Accuracy: %.1f%%\n", score.ErrorAccuracy*100)
	fmt.Printf("Warning Accuracy: %.1f%%\n", score.WarningAccuracy*100)
	fmt.Printf("Notice Accuracy: %.1f%%\n", score.NoticeAccuracy*100)
	fmt.Printf("Overall Score: %.1f%%\n", score.OverallScore*100)
	fmt.Printf("Token Usage: %d input + %d output = %d total tokens\n", usage.InputTokens, usage.OutputTokens, usage.TotalTokens)
}

func countIssuesBySeverity(issues []providers.SemanticIssue) SeverityCount {
	count := SeverityCount{}
	for _, issue := range issues {
		switch strings.ToUpper(issue.Level) {
		case "ERROR":
			count.Errors++
		case "WARNING":
			count.Warnings++
		case "NOTICE":
			count.Notice++
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

func getGitCommitSHA() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown", err
	}
	return strings.TrimSpace(string(output)), nil
}

func recordResults(cfg *config.Config, score *EvalScore, duration time.Duration, usage providers.Usage) error {
	resultsFile := "evals/results.csv"

	// Get current commit SHA
	commitSHA, err := getGitCommitSHA()
	if err != nil {
		fmt.Printf("Warning: could not get git commit SHA: %v\n", err)
		commitSHA = "unknown"
	}

	// Create result record
	result := EvalResult{
		CommitSHA:       commitSHA,
		Date:            time.Now().Format(time.RFC3339),
		Model:           cfg.Model,
		Provider:        cfg.Provider,
		ErrorAccuracy:   score.ErrorAccuracy,
		WarningAccuracy: score.WarningAccuracy,
		InfoAccuracy:    score.NoticeAccuracy, // Notice maps to Info
		TotalAccuracy:   score.OverallScore,
		NumCases:        score.TotalTests,
		Duration:        duration,
		InputTokens:     usage.InputTokens,
		OutputTokens:    usage.OutputTokens,
		TotalTokens:     usage.TotalTokens,
	}

	// Check if file exists
	fileExists := true
	if _, err := os.Stat(resultsFile); os.IsNotExist(err) {
		fileExists = false
	}

	// Open file for appending
	file, err := os.OpenFile(resultsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open results file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close results file: %v\n", err)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header if file is new
	if !fileExists {
		header := []string{"commit_sha", "date", "model", "provider", "error_accuracy", "warning_accuracy", "info_accuracy", "total_accuracy", "num_cases", "duration_seconds", "input_tokens", "output_tokens", "total_tokens"}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write CSV header: %w", err)
		}
	}

	// Write result record
	record := []string{
		result.CommitSHA,
		result.Date,
		result.Model,
		result.Provider,
		fmt.Sprintf("%.4f", result.ErrorAccuracy),
		fmt.Sprintf("%.4f", result.WarningAccuracy),
		fmt.Sprintf("%.4f", result.InfoAccuracy),
		fmt.Sprintf("%.4f", result.TotalAccuracy),
		fmt.Sprintf("%d", result.NumCases),
		fmt.Sprintf("%.2f", result.Duration.Seconds()),
		fmt.Sprintf("%d", result.InputTokens),
		fmt.Sprintf("%d", result.OutputTokens),
		fmt.Sprintf("%d", result.TotalTokens),
	}

	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write CSV record: %w", err)
	}

	fmt.Printf("\n✅ Results recorded to %s\n", resultsFile)
	return nil
}
