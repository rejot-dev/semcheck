package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")
	specPath := filepath.Join(tempDir, "spec.md")

	// Create the spec file that the config references
	err := os.WriteFile(specPath, []byte("# Test Spec"), 0644)
	if err != nil {
		t.Fatalf("failed to write test spec: %v", err)
	}

	validConfig := `version: "1.0"
provider: openai
model: gpt-4
api_key: test-key
timeout: 30
fail_on_issues: true
rules:
  - name: test-rule
    description: Test rule description
    enabled: true
    files:
      include:
        - "*.go"
    specs:
      - path: "` + specPath + `"
    fail_on: "error"
`

	err = os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Test loading valid config
	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify config values
	if config.Version != "1.0" {
		t.Errorf("expected version '1.0', got %s", config.Version)
	}
	if config.Provider != "openai" {
		t.Errorf("expected provider 'openai', got %s", config.Provider)
	}
	if config.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %s", config.Model)
	}
	if config.APIKey != "test-key" {
		t.Errorf("expected api_key 'test-key', got %s", config.APIKey)
	}
	if config.Timeout != 30 {
		t.Errorf("expected timeout 30, got %d", config.Timeout)
	}
	if config.FailOnIssues == nil || !*config.FailOnIssues {
		t.Error("expected fail_on_issues to be true")
	}
	if len(config.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(config.Rules))
	}

	rule := config.Rules[0]
	if rule.Name != "test-rule" {
		t.Errorf("expected rule name 'test-rule', got %s", rule.Name)
	}
	if rule.Description != "Test rule description" {
		t.Errorf("expected rule description 'Test rule description', got %s", rule.Description)
	}
	if !rule.Enabled {
		t.Error("expected rule to be enabled")
	}
	if len(rule.Files.Include) != 1 || rule.Files.Include[0] != "*.go" {
		t.Errorf("expected include pattern '*.go', got %v", rule.Files.Include)
	}
	if len(rule.Specs) != 1 {
		t.Errorf("expected 1 spec, got %d", len(rule.Specs))
	}
	if rule.Specs[0].Path != specPath {
		t.Errorf("expected spec path '%s', got %s", specPath, rule.Specs[0].Path)
	}
	if rule.FailOn != "error" {
		t.Errorf("expected fail_on 'error', got %s", rule.FailOn)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := Load("non-existent-file.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	invalidYAML := `invalid: yaml: content: [unclosed`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestConfig_validate(t *testing.T) {
	// Create temp directory and spec file for tests
	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "spec.md")
	err := os.WriteFile(specPath, []byte("# Test Spec"), 0644)
	if err != nil {
		t.Fatalf("failed to write test spec: %v", err)
	}

	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "valid config",
			config: Config{
				Version:  "1.0",
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
				Rules: []Rule{
					{
						Name:        "test",
						Description: "test rule",
						Files:       FilePattern{Include: []string{"*.go"}},
						Specs:       []Spec{{Path: specPath}},
					},
				},
			},
			wantError: false,
		},
		{
			name: "missing version",
			config: Config{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			wantError: true,
		},
		{
			name: "unsupported version",
			config: Config{
				Version:  "2.0",
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			wantError: true,
		},
		{
			name: "missing provider",
			config: Config{
				Version: "1.0",
				Model:   "gpt-4",
				APIKey:  "test-key",
			},
			wantError: true,
		},
		{
			name: "unsupported provider",
			config: Config{
				Version:  "1.0",
				Provider: "unsupported",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			wantError: true,
		},
		{
			name: "missing model",
			config: Config{
				Version:  "1.0",
				Provider: "openai",
				APIKey:   "test-key",
			},
			wantError: true,
		},
		{
			name: "no rules",
			config: Config{
				Version:  "1.0",
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
				Rules:    []Rule{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_validate_Defaults(t *testing.T) {
	// Create temp directory and spec file for test
	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "spec.md")
	err := os.WriteFile(specPath, []byte("# Test Spec"), 0644)
	if err != nil {
		t.Fatalf("failed to write test spec: %v", err)
	}

	config := Config{
		Version:  "1.0",
		Provider: "openai",
		Model:    "gpt-4",
		APIKey:   "test-key",
		Rules: []Rule{
			{
				Name:        "test",
				Description: "test rule",
				Files:       FilePattern{Include: []string{"*.go"}},
				Specs:       []Spec{{Path: specPath}},
			},
		},
	}

	err = config.validate()
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Check that defaults were set
	if config.Timeout != 30 {
		t.Errorf("expected default timeout 30, got %d", config.Timeout)
	}
	if config.FailOnIssues == nil || !*config.FailOnIssues {
		t.Error("expected default fail_on_issues to be true")
	}
}

func TestConfig_validate_ExplicitFailOnIssuesFalse(t *testing.T) {
	// Create temp directory and spec file for test
	tempDir := t.TempDir()
	specPath := filepath.Join(tempDir, "spec.md")
	err := os.WriteFile(specPath, []byte("# Test Spec"), 0644)
	if err != nil {
		t.Fatalf("failed to write test spec: %v", err)
	}

	explicitFalse := false
	config := Config{
		Version:      "1.0",
		Provider:     "openai",
		Model:        "gpt-4",
		APIKey:       "test-key",
		FailOnIssues: &explicitFalse,
		Rules: []Rule{
			{
				Name:        "test",
				Description: "test rule",
				Files:       FilePattern{Include: []string{"*.go"}},
				Specs:       []Spec{{Path: specPath}},
			},
		},
	}

	err = config.validate()
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Check that explicit false is preserved
	if config.FailOnIssues == nil || *config.FailOnIssues {
		t.Error("expected fail_on_issues to remain false when explicitly set")
	}
}

func TestConfig_validate_URLSpecs(t *testing.T) {
	tests := []struct {
		name      string
		specPath  string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid HTTPS URL",
			specPath:  "https://example.com/spec.md",
			wantError: false,
		},
		{
			name:      "valid HTTP URL",
			specPath:  "http://example.com/spec.md",
			wantError: false,
		},
		{
			name:      "invalid URL format",
			specPath:  "ht!p://invalid-url",
			wantError: true,
			errorMsg:  "invalid URL format",
		},
		{
			name:      "unsupported URL scheme",
			specPath:  "ftp://example.com/spec.md",
			wantError: true,
			errorMsg:  "only HTTP/HTTPS URLs are supported",
		},
		{
			name:      "file URL not supported",
			specPath:  "file:///path/to/file.md",
			wantError: true,
			errorMsg:  "only HTTP/HTTPS URLs are supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Version:  "1.0",
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
				Rules: []Rule{
					{
						Name:        "test",
						Description: "test rule",
						Files:       FilePattern{Include: []string{"*.go"}},
						Specs:       []Spec{{Path: tt.specPath}},
					},
				},
			}

			err := config.validate()
			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
