package checker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/processor"
	"github.com/rejot-dev/semcheck/internal/providers"
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
				Name: "test-rule",
				Files: config.FilePattern{
					Include: []string{"**/*.go"},
				},
				Specs: []config.Spec{
					{Path: "spec.md"},
					{Path: "spec2.md"},
				},
			},
		},
	}

	client := &mockClient{
		responses:          make(map[string]string),
		structuredResponse: []providers.SemanticIssue{}, // No issues found
	}

	checker := NewSemanticChecker(cfg, client, tmpDir)

	// Create a matcher for the test
	matcher, err := processor.NewMatcher(cfg, tmpDir)
	if err != nil {
		t.Fatalf("Failed to create matcher: %v", err)
	}

	matchedFiles := []processor.MatcherResult{
		{
			Path:     "spec.md",
			Type:     processor.FileTypeSpec,
			RuleName: "test-rule",
		},
		{
			Path:     "spec2.md",
			Type:     processor.FileTypeSpec,
			RuleName: "test-rule",
		},
		{
			Path:     "impl.go",
			Type:     processor.FileTypeImpl,
			RuleName: "test-rule",
		},
	}

	ctx := context.Background()
	result, err := checker.CheckFiles(ctx, matchedFiles, matcher)
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
