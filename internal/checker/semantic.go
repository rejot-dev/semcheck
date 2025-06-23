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

type CheckResult struct {
	Issues    []providers.SemanticIssue
	Processed int
	Passed    int
	Failed    int
}

type SemanticChecker struct {
	config     *config.Config
	client     providers.Client
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
						if issue.Level == "ERROR" {
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

func (c *SemanticChecker) compareSpecToImpl(ctx context.Context, rule *config.Rule, specFile, implFile string) ([]providers.SemanticIssue, error) {
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

	// Filter issues by confidence threshold
	var issues []providers.SemanticIssue
	for _, semanticIssue := range resp.Issues {
		if semanticIssue.Confidence >= rule.ConfidenceThreshold {
			issues = append(issues, semanticIssue)
		}
	}

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
	prompt.WriteString("Focus on semantic correctness, not formatting.\n\n")
	prompt.WriteString("Return issues as structured JSON with the following fields:\n")
	prompt.WriteString("- level: ERROR, WARNING, or INFO\n")
	prompt.WriteString("- message: Brief description of the issue\n")
	prompt.WriteString("- confidence: Your confidence level (0.0-1.0)\n")
	prompt.WriteString("- suggestion: How to fix this issue\n")
	prompt.WriteString("- line_number: The line number of the issue (optional)\n\n")
	prompt.WriteString("If no issues are found, return an empty array.")

	return prompt.String()
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
	errorIssues := make([]providers.SemanticIssue, 0)
	warningIssues := make([]providers.SemanticIssue, 0)
	infoIssues := make([]providers.SemanticIssue, 0)

	for _, issue := range result.Issues {
		switch issue.Level {
		case "ERROR":
			errorIssues = append(errorIssues, issue)
		case "WARNING":
			warningIssues = append(warningIssues, issue)
		case "INFO":
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

func displayIssue(issue providers.SemanticIssue) {
	fmt.Printf("  • %s (confidence: %.1f)\n", issue.Message, issue.Confidence)
	if issue.LineNumber > 0 {
		fmt.Printf("    Line: %d\n", issue.LineNumber)
	}
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
		if issue.Level == "ERROR" {
			return true
		}
	}

	return false
}
