package checker

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/rejot-dev/semcheck/internal/providers"
)

var (
	boldCyan   = color.New(color.Bold, color.FgCyan)
	muted      = color.New(color.FgHiBlack)
	foreground = color.New(color.FgWhite)
	boldGreen  = color.New(color.Bold, color.FgGreen)
	boldRed    = color.New(color.Bold, color.FgRed)
	red        = color.New(color.FgRed)
	boldYellow = color.New(color.Bold, color.FgYellow)
	yellow     = color.New(color.FgYellow)
	boldBlue   = color.New(color.Bold, color.FgBlue)
	blue       = color.New(color.FgBlue)
	bold       = color.New(color.Bold)
)

// StdoutReporter implements Reporter interface for console output
type StdoutReporter struct {
	options *StdoutReporterOptions
}

type StdoutReporterOptions struct {
	ShowAnalysis bool
	TextWidth    int
}

// NewStdoutReporter creates a new stdout reporter
func NewStdoutReporter(options *StdoutReporterOptions) *StdoutReporter {
	// Set default value for TextWidth if not specified
	if options.TextWidth == 0 {
		options.TextWidth = 80
	}

	return &StdoutReporter{
		options: options,
	}
}

// Report formats and displays the semantic analysis results to stdout
func (r *StdoutReporter) Report(result *CheckResult) {
	fmt.Print("\n")
	boldCyan.Println("🔍 SEMANTIC ANALYSIS RESULTS")

	if result.Processed == 0 {
		muted.Println("Nothing found to analyze.")
		return
	}

	totalIssues := 0
	for _, issues := range result.Issues {
		totalIssues += len(issues)
	}

	if totalIssues == 0 {
		fmt.Print("\n")
		boldGreen.Println("🎉 No issues found! All implementations match their specifications.")
		return
	}

	// Group issues by level
	errorIssues := make([]providers.SemanticIssue, 0)
	warningIssues := make([]providers.SemanticIssue, 0)
	noticeIssues := make([]providers.SemanticIssue, 0)

	for _, issues := range result.Issues {
		for _, issue := range issues {
			switch issue.Level {
			case "ERROR":
				errorIssues = append(errorIssues, issue)
			case "WARNING":
				warningIssues = append(warningIssues, issue)
			case "NOTICE":
				noticeIssues = append(noticeIssues, issue)
			}
		}
	}

	// Display errors
	if len(errorIssues) > 0 {
		fmt.Print("\n")
		boldRed.Printf("🚨 ERRORS (%d)\n", len(errorIssues))
		for i, issue := range errorIssues {
			r.displayIssue(issue, i+1, red)
			if i < len(errorIssues)-1 {
				fmt.Println()
			}
		}
	}

	// Display warnings
	if len(warningIssues) > 0 {
		fmt.Print("\n")
		boldYellow.Printf("⚠️  WARNINGS (%d)\n", len(warningIssues))
		for i, issue := range warningIssues {
			r.displayIssue(issue, i+1, yellow)
			if i < len(warningIssues)-1 {
				fmt.Println()
			}
		}
	}

	// Display notices
	if len(noticeIssues) > 0 {
		fmt.Print("\n")
		boldBlue.Printf("💡 NOTICE (%d)\n", len(noticeIssues))
		for i, issue := range noticeIssues {
			r.displayIssue(issue, i+1, blue)
			if i < len(noticeIssues)-1 {
				fmt.Println()
			}
		}
	}

	fmt.Print("\n")
	bold.Print("📊 SUMMARY: ")
	red.Printf("%d", len(errorIssues))
	fmt.Print(" errors, ")
	yellow.Printf("%d", len(warningIssues))
	fmt.Print(" warnings, ")
	blue.Printf("%d", len(noticeIssues))
	fmt.Println(" notices")
}

func (r *StdoutReporter) displayIssue(issue providers.SemanticIssue, issueNumber int, issueColor *color.Color) {
	// Main issue message with number
	fmt.Print("   ")
	issueColor.Printf("%d. ", issueNumber)
	bold.Printf("%s\n", issue.Message)

	// Reasoning section with better formatting
	if r.options.ShowAnalysis && issue.Reasoning != "" {
		reasoning := r.wrapText(issue.Reasoning, r.options.TextWidth)
		for _, line := range reasoning {
			fmt.Print("      ")
			foreground.Println(line)
		}
	}

	// Suggestion section with better formatting
	if issue.Suggestion != "" {
		fmt.Print("      ")
		bold.Println("Suggestion:")
		suggestion := r.wrapText(issue.Suggestion, r.options.TextWidth)
		for _, line := range suggestion {
			fmt.Print("      ")
			foreground.Println(line)
		}
	}

	fmt.Println()
}

// wrapText wraps long text to specified width
func (r *StdoutReporter) wrapText(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	currentLine := words[0]

	for _, word := range words[1:] {
		// If adding this word would exceed the width, start a new line
		if len(currentLine)+1+len(word) > width {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}

	// Add the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
