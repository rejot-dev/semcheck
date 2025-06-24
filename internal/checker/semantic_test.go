package checker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rejot.dev/semcheck/internal/config"
	"rejot.dev/semcheck/internal/processor"
	"rejot.dev/semcheck/internal/providers"
)

// Mock client for testing
type mockClient struct {
	responses          map[string]string
	structuredResponse []providers.SemanticIssue
}

func (m *mockClient) Name() string {
	return "mock"
}

func (m *mockClient) Complete(ctx context.Context, req *providers.Request) (*providers.Response, error) {
	resp := &providers.Response{
		Usage: providers.Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
		Issues: m.structuredResponse,
	}

	return resp, nil
}

func (m *mockClient) Validate() error {
	return nil
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
	specPath2 := filepath.Join(tmpDir, "spec2.md")
	implPath := filepath.Join(tmpDir, "impl.go")

	err := os.WriteFile(specPath, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create spec file: %v", err)
	}

	err = os.WriteFile(specPath2, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create spec 2 file: %v", err)
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
		responses:          make(map[string]string),
		structuredResponse: []providers.SemanticIssue{}, // No issues found
	}

	checker := NewSemanticChecker(cfg, client, tmpDir)

	matchedFiles := []processor.MatcherResult{
		{
			Path:         "spec.md",
			Type:         processor.FileTypeSpec,
			MatchedRules: []string{"test-rule"},
			Counterparts: []string{"impl.go"},
		},
		{
			Path:         "spec2.md",
			Type:         processor.FileTypeSpec,
			MatchedRules: []string{"test-rule"},
			Counterparts: []string{"impl.go"},
		},
		{
			Path:         "impl.go",
			Type:         processor.FileTypeImpl,
			MatchedRules: []string{"test-rule"},
			Counterparts: []string{"spec.md", "spec2.md"},
		},
	}

	ctx := context.Background()
	result, err := checker.CheckFiles(ctx, matchedFiles)
	if err != nil {
		t.Fatalf("CheckFiles failed: %v", err)
	}

	if result.Processed != 2 {
		t.Errorf("Expected 2 processed, got %d", result.Processed)
	}

	if result.Passed != 2 {
		t.Errorf("Expected 2 passed, got %d", result.Passed)
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

	userPrompt := checker.buildUserPrompt(rule, "spec.md", "spec content", []string{"impl.go"}, []string{"impl content"})

	if !strings.Contains(userPrompt, "Check for proper error handling") {
		t.Error("User prompt should contain custom rule instructions")
	}

	if !strings.Contains(userPrompt, "spec.md") {
		t.Error("User prompt should contain spec file name")
	}

	if !strings.Contains(userPrompt, "impl.go") {
		t.Error("User prompt should contain impl file name")
	}

	if !strings.Contains(userPrompt, "spec content") {
		t.Error("User prompt should contain spec content")
	}

	if !strings.Contains(userPrompt, "impl content") {
		t.Error("User prompt should contain impl content")
	}

	if !strings.Contains(SystemPrompt, "SEVERITY LEVEL GUIDELINES") {
		t.Error("System prompt should contain severity guidelines")
	}
}
