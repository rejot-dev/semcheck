package checker

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"rejot.dev/semcheck/internal/config"
	"rejot.dev/semcheck/internal/processor"
	"rejot.dev/semcheck/internal/providers"
)

// Mock client for testing
type mockClient struct {
	responses map[string]string
}

func (m *mockClient) Name() string {
	return "mock"
}

func (m *mockClient) Complete(ctx context.Context, req *providers.Request) (*providers.Response, error) {
	response, exists := m.responses[req.Prompt]
	if !exists {
		response = "NO_ISSUES_FOUND"
	}

	return &providers.Response{
		Content: response,
		Usage: providers.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}, nil
}

func (m *mockClient) Validate() error {
	return nil
}

func TestSemanticChecker_parseIssueBlock(t *testing.T) {
	rule := &config.Rule{
		Name:                "test-rule",
		ConfidenceThreshold: 0.7,
	}

	checker := &SemanticChecker{}

	tests := []struct {
		name     string
		block    string
		expected *Issue
	}{
		{
			name: "valid error issue",
			block: `ISSUE: ERROR
MESSAGE: Function signature does not match specification
CONFIDENCE: 0.9
SUGGESTION: Update function to match the specified interface`,
			expected: &Issue{
				Level:      IssueLevelError,
				Message:    "Function signature does not match specification",
				File:       "test.go",
				Rule:       "test-rule",
				Confidence: 0.9,
				Suggestion: "Update function to match the specified interface",
			},
		},
		{
			name: "low confidence issue filtered out",
			block: `ISSUE: WARNING
MESSAGE: Minor inconsistency found
CONFIDENCE: 0.5
SUGGESTION: Consider updating`,
			expected: nil, // Below threshold
		},
		{
			name: "missing required fields",
			block: `ISSUE: ERROR
CONFIDENCE: 0.8`,
			expected: nil, // No message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.parseIssueBlock(tt.block, rule, "test.go")
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseIssueBlock() = %+v, expected %+v", result, tt.expected)
			}
		})
	}
}

func TestSemanticChecker_parseAIResponse(t *testing.T) {
	rule := &config.Rule{
		Name:                "test-rule",
		ConfidenceThreshold: 0.7,
	}

	checker := &SemanticChecker{}

	tests := []struct {
		name     string
		response string
		expected []Issue
	}{
		{
			name:     "no issues found",
			response: "NO_ISSUES_FOUND",
			expected: nil,
		},
		{
			name: "single issue",
			response: `ISSUE: ERROR
MESSAGE: Test issue
CONFIDENCE: 0.8
SUGGESTION: Fix it
---`,
			expected: []Issue{
				{
					Level:      IssueLevelError,
					Message:    "Test issue",
					File:       "test.go",
					Rule:       "test-rule",
					Confidence: 0.8,
					Suggestion: "Fix it",
				},
			},
		},
		{
			name: "multiple issues",
			response: `ISSUE: WARNING
MESSAGE: First issue
CONFIDENCE: 0.9
SUGGESTION: Fix first
---
ISSUE: ERROR
MESSAGE: Second issue
CONFIDENCE: 0.8
SUGGESTION: Fix second
---`,
			expected: []Issue{
				{
					Level:      IssueLevelWarning,
					Message:    "First issue",
					File:       "test.go",
					Rule:       "test-rule",
					Confidence: 0.9,
					Suggestion: "Fix first",
				},
				{
					Level:      IssueLevelError,
					Message:    "Second issue",
					File:       "test.go",
					Rule:       "test-rule",
					Confidence: 0.8,
					Suggestion: "Fix second",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.parseAIResponse(tt.response, rule, "test.go")
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseAIResponse() = %+v, expected %+v", result, tt.expected)
			}
		})
	}
}

func TestSemanticChecker_groupFilesByRules(t *testing.T) {
	checker := &SemanticChecker{}

	matchedFiles := []processor.MatcherResult{
		{
			Path:         "spec.md",
			Type:         processor.FileTypeSpec,
			MatchedRules: []string{"rule1", "rule2"},
		},
		{
			Path:         "impl.go",
			Type:         processor.FileTypeImpl,
			MatchedRules: []string{"rule1"},
		},
		{
			Path:         "ignored.log",
			Type:         processor.FileTypeIgnored,
			MatchedRules: []string{"rule1"},
		},
		{
			Path:         "other.go",
			Type:         processor.FileTypeImpl,
			MatchedRules: []string{"rule2"},
		},
	}

	result := checker.groupFilesByRules(matchedFiles)

	expected := map[string]*ruleFileGroup{
		"rule1": {
			specificationFiles:  []string{"spec.md"},
			implementationFiles: []string{"impl.go"},
		},
		"rule2": {
			specificationFiles:  []string{"spec.md"},
			implementationFiles: []string{"other.go"},
		},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("groupFilesByRules() = %+v, expected %+v", result, expected)
	}
}

func TestSemanticChecker_CheckFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	specContent := "# Test Specification\nThis function should return the sum of two numbers."
	implContent := `package main
func Add(a, b int) int {
	return a + b
}`

	specPath := filepath.Join(tmpDir, "spec.md")
	implPath := filepath.Join(tmpDir, "impl.go")

	err := os.WriteFile(specPath, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create spec file: %v", err)
	}

	err = os.WriteFile(implPath, []byte(implContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create impl file: %v", err)
	}

	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:                "test-rule",
				ConfidenceThreshold: 0.7,
			},
		},
	}

	client := &mockClient{
		responses: make(map[string]string),
	}

	checker := NewSemanticChecker(cfg, client, tmpDir)

	matchedFiles := []processor.MatcherResult{
		{
			Path:         "spec.md",
			Type:         processor.FileTypeSpec,
			MatchedRules: []string{"test-rule"},
		},
		{
			Path:         "impl.go",
			Type:         processor.FileTypeImpl,
			MatchedRules: []string{"test-rule"},
		},
	}

	ctx := context.Background()
	result, err := checker.CheckFiles(ctx, matchedFiles)
	if err != nil {
		t.Fatalf("CheckFiles failed: %v", err)
	}

	if result.Processed != 1 {
		t.Errorf("Expected 1 processed, got %d", result.Processed)
	}

	if result.Passed != 1 {
		t.Errorf("Expected 1 passed, got %d", result.Passed)
	}

	if result.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", result.Failed)
	}
}

func TestSemanticChecker_buildComparisonPrompt(t *testing.T) {
	checker := &SemanticChecker{}

	rule := &config.Rule{
		Name:   "test-rule",
		Prompt: "Check for proper error handling",
	}

	prompt := checker.buildComparisonPrompt(rule, "spec.md", "spec content", "impl.go", "impl content")

	if !strings.Contains(prompt, "Check for proper error handling") {
		t.Error("Prompt should contain custom rule instructions")
	}

	if !strings.Contains(prompt, "SPECIFICATION FILE: spec.md") {
		t.Error("Prompt should contain spec file name")
	}

	if !strings.Contains(prompt, "IMPLEMENTATION FILE: impl.go") {
		t.Error("Prompt should contain impl file name")
	}

	if !strings.Contains(prompt, "spec content") {
		t.Error("Prompt should contain spec content")
	}

	if !strings.Contains(prompt, "impl content") {
		t.Error("Prompt should contain impl content")
	}
}

func TestIssueLevel_String(t *testing.T) {
	tests := []struct {
		level    IssueLevel
		expected string
	}{
		{IssueLevelInfo, "info"},
		{IssueLevelWarning, "warning"},
		{IssueLevelError, "error"},
		{IssueLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("IssueLevel.String() = %v, expected %v", got, tt.expected)
			}
		})
	}
}