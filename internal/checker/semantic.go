package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/processor"
	"github.com/rejot-dev/semcheck/internal/providers"
)

type CheckResult struct {
	Issues      map[string][]providers.SemanticIssue
	Processed   int
	Passed      int
	Failed      int
	HasFailures bool
}

type RuleComparisonFiles struct {
	SpecFiles []processor.SpecFile
	ImplFiles []string
}

type SemanticChecker struct {
	config     *config.Config
	client     providers.Client[providers.IssueResponse]
	workingDir string
}

func NewSemanticChecker(cfg *config.Config, client providers.Client[providers.IssueResponse], workingDir string) *SemanticChecker {
	return &SemanticChecker{
		config:     cfg,
		client:     client,
		workingDir: workingDir,
	}
}

func (c *SemanticChecker) CheckFiles(ctx context.Context, matches []processor.MatcherResult, matcher *processor.Matcher) (*CheckResult, error) {
	result := &CheckResult{
		Issues: make(map[string][]providers.SemanticIssue),
	}

	// Group files by rule for comparison
	ruleComparisons := c.buildRuleComparisons(matches, matcher)

	for ruleName, comparison := range ruleComparisons {
		rule := c.findRule(ruleName)
		if rule == nil {
			continue
		}

		// Make a single comparison for all files in this rule
		issues, err := c.compareSpecToImpl(ctx, rule, comparison.SpecFiles, comparison.ImplFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to compare rule %s: %w", ruleName, err)
		}

		result.Issues[ruleName] = issues
		result.Processed++

		if len(issues) == 0 {
			result.Passed++
		} else {
			// Check if any issues meet the rule's fail_on threshold for failure
			shouldFailForRule := false
			ruleSeverityLevel := severityLevel(strings.ToUpper(rule.FailOn))

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

	return result, nil
}

// buildRuleComparisons groups files by rule for efficient comparison
func (c *SemanticChecker) buildRuleComparisons(matches []processor.MatcherResult, matcher *processor.Matcher) map[string]*RuleComparisonFiles {
	ruleFiles := make(map[string]*RuleComparisonFiles)

	// Process each matched file - now each match has a single rule and type
	for _, match := range matches {
		if match.Type == processor.FileTypeIgnored {
			continue
		}

		ruleName := match.RuleName

		// Initialize rule comparison if it does not exist
		if ruleFiles[ruleName] == nil {
			ruleFiles[ruleName] = &RuleComparisonFiles{
				SpecFiles: []processor.SpecFile{},
				ImplFiles: []string{},
			}
		}

		comp := ruleFiles[ruleName]

		// Add file to appropriate list based on type
		switch match.Type {
		case processor.FileTypeSpec:
			comp.SpecFiles = append(comp.SpecFiles, processor.SpecFile{
				Path:         match.Path,
				Specifically: match.Specifically,
			})
		case processor.FileTypeImpl:
			comp.ImplFiles = append(comp.ImplFiles, string(match.Path))
		}
	}

	// For each rule, ensure we have all relevant files
	for ruleName, comp := range ruleFiles {
		// Get all counterpart files for specs in this rule
		if len(comp.SpecFiles) > 0 {
			counterparts := processor.NormalizedPathsToStrings(matcher.GetRuleImplFiles(ruleName))
			comp.ImplFiles = c.mergeUnique(comp.ImplFiles, counterparts)
		}

		// Get all counterpart files for impls in this rule
		if len(comp.ImplFiles) > 0 {
			comp.SpecFiles = c.mergeUniqueSpecFiles(comp.SpecFiles, matcher.GetRuleSpecFiles(ruleName))
		}
	}

	return ruleFiles
}

func (c *SemanticChecker) mergeUniqueSpecFiles(slice1, slice2 []processor.SpecFile) []processor.SpecFile {
	seen := make(map[processor.NormalizedPath]bool)
	result := make([]processor.SpecFile, 0)

	// Add all from slice1
	for _, item := range slice1 {
		if !seen[item.Path] {
			seen[item.Path] = true
			result = append(result, item)
		}
	}

	// Add unique items from slice2
	for _, item := range slice2 {
		if !seen[item.Path] {
			seen[item.Path] = true
			result = append(result, item)
		}
	}

	return result
}

// mergeUnique merges two string slices, removing duplicates
func (c *SemanticChecker) mergeUnique(slice1, slice2 []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	// Add all from slice1
	for _, item := range slice1 {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	// Add unique items from slice2
	for _, item := range slice2 {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func (c *SemanticChecker) findRule(name string) *config.Rule {
	for i := range c.config.Rules {
		if c.config.Rules[i].Name == name {
			return &c.config.Rules[i]
		}
	}
	return nil
}

func (c *SemanticChecker) compareSpecToImpl(ctx context.Context, rule *config.Rule, specFiles []processor.SpecFile, implFiles []string) ([]providers.SemanticIssue, error) {
	fmt.Printf("For rule '%s', comparing spec files %s to implementation files %v\n", rule.Name, specFiles, implFiles)
	// Read specification files
	specContents := make([]string, len(specFiles))
	for i, specFile := range specFiles {
		specContent, err := c.readFile(string(specFile.Path))
		if err != nil {
			return nil, fmt.Errorf("failed to read spec file %s: %w", specFile, err)
		}
		// TODO: implement context reduction when specFile.Specifically is set
		// if specFile.Specifically != "" {
		//     // reduce the context to focus on specific sections
		// }
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
	specFilePaths := make([]string, len(specFiles))
	for i, specFile := range specFiles {
		specFilePaths[i] = string(specFile.Path)
	}
	userPrompt := c.buildUserPrompt(rule, specFilePaths, specContents, implFiles, implContents)

	// Get AI analysis
	req := &providers.Request{
		SystemPrompt: SystemPrompt,
		UserPrompt:   userPrompt,
	}

	resp, _, err := c.client.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	return resp.Issues, nil
}

func (c *SemanticChecker) readFile(filePath string) (string, error) {
	// Check if the path is a URL
	if isURL(filePath) {
		return c.readURL(filePath)
	}

	// Handle local file
	fullPath := filepath.Join(c.workingDir, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// isURL checks if a string is a URL
func isURL(str string) bool {
	return strings.Contains(str, "://") && (strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://"))
}

// readURL fetches content from a URL
func (c *SemanticChecker) readURL(url string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log error but don't fail the operation
			fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch URL %s: HTTP %d %s", url, resp.StatusCode, resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read content from URL %s: %w", url, err)
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
	case "NOTICE":
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
