package checker

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rejot-dev/semcheck/internal/color"
	"github.com/rejot-dev/semcheck/internal/providers"
)

var (
	boldCyan = lipgloss.NewStyle().
			Bold(true).
			Foreground(color.Cyan)

	muted = lipgloss.NewStyle().
		Foreground(color.DarkGray)

	foreground = lipgloss.NewStyle().
			Foreground(color.LightGray)

	boldGreen = lipgloss.NewStyle().
			Bold(true).
			Foreground(color.DarkGreen)

	boldRed = lipgloss.NewStyle().
		Bold(true).
		Foreground(color.DarkRed)

	red = lipgloss.NewStyle().
		Foreground(color.Red)

	boldYellow = lipgloss.NewStyle().
			Bold(true).
			Foreground(color.Orange)

	yellow = lipgloss.NewStyle().
		Foreground(color.Yellow)

	boldBlue = lipgloss.NewStyle().
			Bold(true).
			Foreground(color.DarkBlue)

	blue = lipgloss.NewStyle().
		Foreground(color.Blue)

	bold = lipgloss.NewStyle().
		Bold(true)
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
	fmt.Println(boldCyan.Render("🔍 SEMANTIC ANALYSIS RESULTS"))

	if result.Processed == 0 {
		fmt.Println(muted.Render("Nothing found to analyze."))
		return
	}

	totalIssues := 0
	for _, issues := range result.Issues {
		totalIssues += len(issues)
	}

	if totalIssues == 0 {
		fmt.Print("\n")
		fmt.Println(boldGreen.Render("🎉 No issues found! All implementations match their specifications."))
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
		fmt.Println(boldRed.Render(fmt.Sprintf("🚨 ERRORS (%d)", len(errorIssues))))
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
		fmt.Println(boldYellow.Render(fmt.Sprintf("⚠️  WARNINGS (%d)", len(warningIssues))))
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
		fmt.Println(boldBlue.Render(fmt.Sprintf("💡 NOTICE (%d)", len(noticeIssues))))
		for i, issue := range noticeIssues {
			r.displayIssue(issue, i+1, blue)
			if i < len(noticeIssues)-1 {
				fmt.Println()
			}
		}
	}

	fmt.Print("\n")
	summary := bold.Render("📊 SUMMARY: ") +
		red.Render(fmt.Sprintf("%d", len(errorIssues))) +
		" errors, " +
		yellow.Render(fmt.Sprintf("%d", len(warningIssues))) +
		" warnings, " +
		blue.Render(fmt.Sprintf("%d", len(noticeIssues))) +
		" notices"
	fmt.Println(summary)
}

func (r *StdoutReporter) displayIssue(issue providers.SemanticIssue, issueNumber int, issueColor lipgloss.Style) {
	// Main issue message with number
	numberText := issueColor.Render(fmt.Sprintf("%d. ", issueNumber))
	messageText := bold.Render(issue.Message)
	fmt.Printf("   %s%s\n", numberText, messageText)

	// Reasoning section with better formatting
	if r.options.ShowAnalysis && issue.Reasoning != "" {
		reasoning := r.wrapText(issue.Reasoning, r.options.TextWidth)
		for _, line := range reasoning {
			fmt.Printf("      %s\n", foreground.Render(line))
		}
	}

	// Suggestion section with better formatting
	if issue.Suggestion != "" {
		fmt.Printf("      %s\n", bold.Render("Suggestion:"))
		suggestion := r.wrapText(issue.Suggestion, r.options.TextWidth)
		for _, line := range suggestion {
			fmt.Printf("      %s\n", foreground.Render(line))
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
