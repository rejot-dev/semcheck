package checker

import (
	"fmt"
	"strings"

	"github.com/rejot-dev/semcheck/internal/providers"
)

// GitHubReporter implements Reporter interface for GitHub Actions annotations
type GitHubReporter struct {
	options *GitHubReporterOptions
}

type GitHubReporterOptions struct {
	ShowAnalysis bool
	WorkingDir   string
}

// NewGitHubReporter creates a new GitHub Actions reporter
func NewGitHubReporter(options *GitHubReporterOptions) *GitHubReporter {
	return &GitHubReporter{
		options: options,
	}
}

// Report outputs GitHub Actions annotations for semantic analysis results
func (r *GitHubReporter) Report(result *CheckResult) {
	if result.Processed == 0 {
		r.outputNotice("Nothing found to analyze.")
		return
	}

	totalIssues := 0
	for _, issues := range result.Issues {
		totalIssues += len(issues)
	}

	if totalIssues == 0 {
		r.outputNotice("🎉 No issues found! All implementations match their specifications.")
		return
	}

	// Output summary
	r.outputNotice(fmt.Sprintf("📊 Found %d total issues across %d rules", totalIssues, len(result.Issues)))

	// Output annotations for each issue
	for ruleName, issues := range result.Issues {
		for _, issue := range issues {
			r.outputAnnotation(ruleName, issue)
		}
	}
}

// outputAnnotation outputs a GitHub Actions annotation for a single issue
func (r *GitHubReporter) outputAnnotation(ruleName string, issue providers.SemanticIssue) {
	// Determine annotation level based on issue level
	level := "warning"
	if strings.ToUpper(issue.Level) == "ERROR" {
		level = "error"
	} else if strings.ToUpper(issue.Level) == "NOTICE" {
		level = "notice"
	}

	// Build the annotation message
	message := fmt.Sprintf("[%s] %s", ruleName, issue.Message)

	// Add confidence if available
	if issue.Confidence > 0 {
		message += fmt.Sprintf(" (confidence: %.1f%%)", issue.Confidence*100)
	}

	// Add reasoning if enabled and available
	if r.options.ShowAnalysis && issue.Reasoning != "" {
		message += fmt.Sprintf("\n\nReasoning:\n%s", issue.Reasoning)
	}

	// Add suggestion if available
	if issue.Suggestion != "" {
		message += fmt.Sprintf("\n\nSuggestion:\n%s", issue.Suggestion)
	}

	// Escape the message for GitHub Actions
	escapedMessage := r.escapeForGitHubActions(message)

	// Output the annotation
	annotation := fmt.Sprintf("::%s file=%s,line=1,col=1::%s", level, issue.File, escapedMessage)
	fmt.Println(annotation)
}

// outputNotice outputs a GitHub Actions notice message
func (r *GitHubReporter) outputNotice(message string) {
	escapedMessage := r.escapeForGitHubActions(message)
	fmt.Printf("::notice ::%s\n", escapedMessage)
}

// escapeForGitHubActions escapes special characters for GitHub Actions annotations
func (r *GitHubReporter) escapeForGitHubActions(message string) string {
	// Replace newlines with %0A
	message = strings.ReplaceAll(message, "\n", "%0A")
	// Replace carriage returns with %0D
	message = strings.ReplaceAll(message, "\r", "%0D")
	return message
}
