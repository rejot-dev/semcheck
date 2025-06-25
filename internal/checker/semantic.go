package checker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"rejot.dev/semcheck/internal/config"
	"rejot.dev/semcheck/internal/processor"
	"rejot.dev/semcheck/internal/providers"
)

type CheckResult struct {
	Issues      map[string][]providers.SemanticIssue
	Processed   int
	Passed      int
	Failed      int
	HasFailures bool
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

func compareKey(ruleName string, path string) string {
	return fmt.Sprintf("%s:%s", ruleName, path)
}

func (c *SemanticChecker) CheckFiles(ctx context.Context, matches []processor.MatcherResult) (*CheckResult, error) {
	result := &CheckResult{
		Issues: make(map[string][]providers.SemanticIssue),
	}

	compared := make(map[string]bool)

	for _, match := range matches {
		if match.Type == processor.FileTypeIgnored {
			continue
		}

		for _, ruleName := range match.MatchedRules {
			rule := c.findRule(ruleName)
			if rule == nil {
				continue
			}

			// Check if we've already compared this file
			// Assuming that if either specification or implementation file was used in a previous comparison
			// it's been sufficiently analyzed already
			// FIXME: this might not be entirely optimal
			if compared[compareKey(ruleName, match.Path)] {
				continue
			}

			var issues []providers.SemanticIssue
			var err error
			if match.Type == processor.FileTypeSpec {
				issues, err = c.compareSpecToImpl(ctx, rule, []string{match.Path}, match.Counterparts)
			} else {
				issues, err = c.compareSpecToImpl(ctx, rule, match.Counterparts, []string{match.Path})
			}

			// Register files as compared
			compared[compareKey(ruleName, match.Path)] = true
			for _, counterpart := range match.Counterparts {
				compared[compareKey(ruleName, counterpart)] = true
			}

			if err != nil {
				return nil, fmt.Errorf("failed to compare %s to %s: %w", match.Path, match.Counterparts, err)
			}

			result.Issues[ruleName] = append(result.Issues[ruleName], issues...)
			result.Processed++

			if len(issues) == 0 {
				result.Passed++
			} else {
				// Check if any issues meet the rule's severity threshold for failure
				shouldFailForRule := false
				ruleSeverityLevel := severityLevel(strings.ToUpper(rule.Severity))

				for _, issue := range issues {
					issueSeverityLevel := severityLevel(issue.Level)
					if issueSeverityLevel >= ruleSeverityLevel {
						shouldFailForRule = true
						result.HasFailures = true
						break
					}
				}

				if shouldFailForRule {
					result.Failed++
				} else {
					result.Passed++
				}
			}

		}
	}

	return result, nil
}

func (c *SemanticChecker) findRule(name string) *config.Rule {
	for i := range c.config.Rules {
		if c.config.Rules[i].Name == name {
			return &c.config.Rules[i]
		}
	}
	return nil
}

func (c *SemanticChecker) compareSpecToImpl(ctx context.Context, rule *config.Rule, specFiles []string, implFiles []string) ([]providers.SemanticIssue, error) {
	fmt.Printf("Comparing spec file %s to implementation files %v\n", specFiles, implFiles)
	// Read specification files
	specContents := make([]string, len(specFiles))
	for i, specFile := range specFiles {
		specContent, err := c.readFile(specFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read spec file %s: %w", specFile, err)
		}
		specContents[i] = specContent
	}

	// Read implementation files
	var implContents []string
	for _, implFile := range implFiles {
		implContent, err := c.readFile(implFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read implementation file %s: %w", implFile, err)
		}
		implContents = append(implContents, implContent)
	}

	// Create AI user prompt for comparison
	userPrompt := c.buildUserPrompt(rule, specFiles, specContents, implFiles, implContents)

	// Get AI analysis
	req := &providers.Request{
		SystemPrompt: SystemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    3000,
		Temperature:  0.1,
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

func (c *SemanticChecker) buildUserPrompt(rule *config.Rule, specFiles []string, specContents []string, implFiles []string, implContents []string) string {
	data := PromptData{
		RulePrompt:   rule.Prompt,
		SpecFiles:    specFiles,
		SpecContents: specContents,
		ImplFiles:    implFiles,
		ImplContent:  implContents,
	}

	// Build user prompt
	userTmpl := template.Must(template.New("user").Parse(UserPromptTemplate))
	var userResult strings.Builder
	// TODO: actual error handling?
	if err := userTmpl.Execute(&userResult, data); err != nil {
		return ""
	}

	return userResult.String()
}

// severityLevel returns the numeric value for severity comparison
func severityLevel(level string) int {
	switch level {
	case "INFO":
		return 1
	case "WARNING":
		return 2
	case "ERROR":
		return 3
	default:
		return 0
	}
}

// ShouldFail determines if the check results should cause the tool to exit with error
func (r *CheckResult) ShouldFail(config *config.Config) bool {
	if config.FailOnIssues == nil || !*config.FailOnIssues {
		return false
	}

	return r.HasFailures
}
