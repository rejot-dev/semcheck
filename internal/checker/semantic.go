package checker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rejot.dev/semcheck/internal/config"
	"rejot.dev/semcheck/internal/processor"
	"rejot.dev/semcheck/internal/providers"
)

type IssueLevel int

const (
	IssueLevelInfo IssueLevel = iota
	IssueLevelWarning
	IssueLevelError
)

func (l IssueLevel) String() string {
	switch l {
	case IssueLevelInfo:
		return "info"
	case IssueLevelWarning:
		return "warning"
	case IssueLevelError:
		return "error"
	default:
		return "unknown"
	}
}

type Issue struct {
	Level       IssueLevel
	Message     string
	File        string
	Rule        string
	Confidence  float64
	Suggestion  string
}

type CheckResult struct {
	Issues    []Issue
	Processed int
	Passed    int
	Failed    int
}

type SemanticChecker struct {
	config    *config.Config
	client    providers.Client
	workingDir string
}

func NewSemanticChecker(cfg *config.Config, client providers.Client, workingDir string) *SemanticChecker {
	return &SemanticChecker{
		config:     cfg,
		client:     client,
		workingDir: workingDir,
	}
}

func (c *SemanticChecker) CheckFiles(ctx context.Context, matchedFiles []processor.MatcherResult) (*CheckResult, error) {
	result := &CheckResult{}

	// Group files by rules for efficient processing
	ruleGroups := c.groupFilesByRules(matchedFiles)

	for ruleName, group := range ruleGroups {
		rule := c.findRule(ruleName)
		if rule == nil {
			continue
		}

		for _, implFile := range group.implementationFiles {
			for _, specFile := range group.specificationFiles {
				issues, err := c.compareSpecToImpl(ctx, rule, specFile, implFile)
				if err != nil {
					return nil, fmt.Errorf("failed to compare %s to %s: %w", specFile, implFile, err)
				}

				result.Issues = append(result.Issues, issues...)
				result.Processed++

				if len(issues) == 0 {
					result.Passed++
				} else {
					// Check if any issues are errors
					hasErrors := false
					for _, issue := range issues {
						if issue.Level == IssueLevelError {
							hasErrors = true
							break
						}
					}
					if hasErrors {
						result.Failed++
					} else {
						result.Passed++
					}
				}
			}
		}
	}

	return result, nil
}

type ruleFileGroup struct {
	implementationFiles []string
	specificationFiles  []string
}

func (c *SemanticChecker) groupFilesByRules(matchedFiles []processor.MatcherResult) map[string]*ruleFileGroup {
	groups := make(map[string]*ruleFileGroup)

	for _, file := range matchedFiles {
		if file.Type == processor.FileTypeIgnored {
			continue
		}

		for _, ruleName := range file.MatchedRules {
			if groups[ruleName] == nil {
				groups[ruleName] = &ruleFileGroup{}
			}

			switch file.Type {
			case processor.FileTypeSpec:
				groups[ruleName].specificationFiles = append(groups[ruleName].specificationFiles, file.Path)
			case processor.FileTypeImpl:
				groups[ruleName].implementationFiles = append(groups[ruleName].implementationFiles, file.Path)
			}
		}
	}

	return groups
}

func (c *SemanticChecker) findRule(name string) *config.Rule {
	for i := range c.config.Rules {
		if c.config.Rules[i].Name == name {
			return &c.config.Rules[i]
		}
	}
	return nil
}

func (c *SemanticChecker) compareSpecToImpl(ctx context.Context, rule *config.Rule, specFile, implFile string) ([]Issue, error) {
	// Read specification file
	specContent, err := c.readFile(specFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file %s: %w", specFile, err)
	}

	// Read implementation file
	implContent, err := c.readFile(implFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read implementation file %s: %w", implFile, err)
	}

	// Create AI prompt for comparison
	prompt := c.buildComparisonPrompt(rule, specFile, specContent, implFile, implContent)

	// Get AI analysis
	req := &providers.Request{
		Prompt:      prompt,
		MaxTokens:   2000,
		Temperature: 0.1,
	}

	resp, err := c.client.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// Parse AI response into issues
	issues := c.parseAIResponse(resp.Content, rule, implFile)

	return issues, nil
}

func (c *SemanticChecker) readFile(filePath string) (string, error) {
	fullPath := filepath.Join(c.workingDir, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (c *SemanticChecker) buildComparisonPrompt(rule *config.Rule, specFile, specContent, implFile, implContent string) string {
	var prompt strings.Builder

	prompt.WriteString("You are a code reviewer analyzing whether an implementation matches its specification.\n\n")
	
	if rule.Prompt != "" {
		prompt.WriteString("Special instructions for this rule:\n")
		prompt.WriteString(rule.Prompt)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString(fmt.Sprintf("SPECIFICATION FILE: %s\n", specFile))
	prompt.WriteString("```\n")
	prompt.WriteString(specContent)
	prompt.WriteString("\n```\n\n")

	prompt.WriteString(fmt.Sprintf("IMPLEMENTATION FILE: %s\n", implFile))
	prompt.WriteString("```\n")
	prompt.WriteString(implContent)
	prompt.WriteString("\n```\n\n")

	prompt.WriteString("Please analyze whether the implementation correctly follows the specification.\n")
	prompt.WriteString("Report any issues found using this exact format:\n\n")
	prompt.WriteString("ISSUE: [ERROR|WARNING|INFO]\n")
	prompt.WriteString("MESSAGE: [Brief description of the issue]\n")
	prompt.WriteString("CONFIDENCE: [0.0-1.0]\n")
	prompt.WriteString("SUGGESTION: [How to fix this issue]\n")
	prompt.WriteString("---\n\n")
	prompt.WriteString("If no issues are found, respond with: NO_ISSUES_FOUND\n")
	prompt.WriteString("Focus on semantic correctness, not formatting.")

	return prompt.String()
}

func (c *SemanticChecker) parseAIResponse(response string, rule *config.Rule, implFile string) []Issue {
	var issues []Issue

	if strings.Contains(response, "NO_ISSUES_FOUND") {
		return issues
	}

	// Split response into individual issue blocks
	blocks := strings.Split(response, "---")

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		issue := c.parseIssueBlock(block, rule, implFile)
		if issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues
}

func (c *SemanticChecker) parseIssueBlock(block string, rule *config.Rule, implFile string) *Issue {
	lines := strings.Split(block, "\n")
	issue := &Issue{
		File: implFile,
		Rule: rule.Name,
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "ISSUE:") {
			levelStr := strings.TrimSpace(strings.TrimPrefix(line, "ISSUE:"))
			switch strings.ToUpper(levelStr) {
			case "ERROR":
				issue.Level = IssueLevelError
			case "WARNING":
				issue.Level = IssueLevelWarning
			case "INFO":
				issue.Level = IssueLevelInfo
			default:
				issue.Level = IssueLevelWarning
			}
		} else if strings.HasPrefix(line, "MESSAGE:") {
			issue.Message = strings.TrimSpace(strings.TrimPrefix(line, "MESSAGE:"))
		} else if strings.HasPrefix(line, "CONFIDENCE:") {
			confidenceStr := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
			if conf, err := parseFloat(confidenceStr); err == nil {
				issue.Confidence = conf
			}
		} else if strings.HasPrefix(line, "SUGGESTION:") {
			issue.Suggestion = strings.TrimSpace(strings.TrimPrefix(line, "SUGGESTION:"))
		}
	}

	// Only return issue if it has required fields and meets confidence threshold
	if issue.Message != "" && issue.Confidence >= rule.ConfidenceThreshold {
		return issue
	}

	return nil
}

func parseFloat(s string) (float64, error) {
	// Simple float parsing - in production you'd use strconv.ParseFloat
	var result float64
	n, err := fmt.Sscanf(s, "%f", &result)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("invalid float: %s", s)
	}
	if result < 0 {
		result = 0
	}
	if result > 1 {
		result = 1
	}
	return result, nil
}

// DisplayCheckResults formats and displays the semantic analysis results
func DisplayCheckResults(result *CheckResult) {
	fmt.Println("\n--- Semantic Analysis Results ---")

	if result.Processed == 0 {
		fmt.Println("No spec-implementation pairs found to analyze.")
		return
	}

	fmt.Printf("Analyzed %d spec-implementation pairs\n", result.Processed)
	fmt.Printf("✅ Passed: %d\n", result.Passed)
	if result.Failed > 0 {
		fmt.Printf("❌ Failed: %d\n", result.Failed)
	}

	if len(result.Issues) == 0 {
		fmt.Println("\n🎉 No issues found! All implementations match their specifications.")
		return
	}

	// Group issues by level
	errorIssues := make([]Issue, 0)
	warningIssues := make([]Issue, 0)
	infoIssues := make([]Issue, 0)

	for _, issue := range result.Issues {
		switch issue.Level {
		case IssueLevelError:
			errorIssues = append(errorIssues, issue)
		case IssueLevelWarning:
			warningIssues = append(warningIssues, issue)
		case IssueLevelInfo:
			infoIssues = append(infoIssues, issue)
		}
	}

	// Display errors
	if len(errorIssues) > 0 {
		fmt.Printf("\n🚨 Errors (%d):\n", len(errorIssues))
		for _, issue := range errorIssues {
			displayIssue(issue)
		}
	}

	// Display warnings
	if len(warningIssues) > 0 {
		fmt.Printf("\n⚠️  Warnings (%d):\n", len(warningIssues))
		for _, issue := range warningIssues {
			displayIssue(issue)
		}
	}

	// Display info
	if len(infoIssues) > 0 {
		fmt.Printf("\n💡 Info (%d):\n", len(infoIssues))
		for _, issue := range infoIssues {
			displayIssue(issue)
		}
	}

	fmt.Printf("\nSummary: %d errors, %d warnings, %d info\n", 
		len(errorIssues), len(warningIssues), len(infoIssues))
}

func displayIssue(issue Issue) {
	fmt.Printf("  • %s (confidence: %.1f)\n", issue.Message, issue.Confidence)
	fmt.Printf("    File: %s | Rule: %s\n", issue.File, issue.Rule)
	if issue.Suggestion != "" {
		fmt.Printf("    💡 %s\n", issue.Suggestion)
	}
	fmt.Println()
}

// ShouldFail determines if the check results should cause the tool to exit with error
func (r *CheckResult) ShouldFail(config *config.Config) bool {
	if !config.FailOnIssues {
		return false
	}

	// Fail if there are any error-level issues
	for _, issue := range r.Issues {
		if issue.Level == IssueLevelError {
			return true
		}
	}

	return false
}