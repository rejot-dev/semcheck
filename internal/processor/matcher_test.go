package processor

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"rejot.dev/semcheck/internal/config"
)

func TestNewMatcher(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test .gitignore file
	gitignoreContent := `# Test gitignore
*.log
temp/
.DS_Store
`
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .gitignore: %v", err)
	}

	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:    "test-rule",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"**/*.go"},
					Exclude: []string{"**/*_test.go"},
				},
			},
		},
	}

	matcher, err := NewMatcher(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	expectedRules := []string{"*.log", "temp/", ".DS_Store"}
	if !reflect.DeepEqual(matcher.gitignoreRules, expectedRules) {
		t.Errorf("Expected gitignore rules %v, got %v", expectedRules, matcher.gitignoreRules)
	}
}

func TestMatcher_matchesPattern(t *testing.T) {
	matcher := &Matcher{}

	tests := []struct {
		name     string
		filePath string
		pattern  string
		expected bool
	}{
		{
			name:     "simple match",
			filePath: "test.go",
			pattern:  "*.go",
			expected: true,
		},
		{
			name:     "simple no match",
			filePath: "test.py",
			pattern:  "*.go",
			expected: false,
		},
		{
			name:     "glob pattern match",
			filePath: "src/main.go",
			pattern:  "**/*.go",
			expected: true,
		},
		{
			name:     "glob pattern no match",
			filePath: "src/main.py",
			pattern:  "**/*.go",
			expected: false,
		},
		{
			name:     "directory pattern match",
			filePath: "vendor/lib/test.go",
			pattern:  "vendor/**",
			expected: true,
		},
		{
			name:     "directory pattern no match",
			filePath: "src/test.go",
			pattern:  "vendor/**",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.matchesPattern(tt.filePath, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, expected %v", tt.filePath, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestMatcher_isIgnoredByGitignore(t *testing.T) {
	matcher := &Matcher{
		gitignoreRules: []string{"*.log", "temp/", ".DS_Store"},
	}

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "ignored log file",
			filePath: "debug.log",
			expected: true,
		},
		{
			name:     "not ignored go file",
			filePath: "main.go",
			expected: false,
		},
		{
			name:     "ignored temp directory file",
			filePath: "temp/test.txt",
			expected: true,
		},
		{
			name:     "ignored system file",
			filePath: ".DS_Store",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.isIgnoredByGitignore(tt.filePath)
			if result != tt.expected {
				t.Errorf("isIgnoredByGitignore(%q) = %v, expected %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestMatcher_matchFile(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:    "go-files",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"**/*.go"},
					Exclude: []string{"**/*_test.go"},
				},
				Specs: []config.Spec{
					{
						Path: "./specs/*.md",
					},
				},
			},
		},
	}

	matcher := &Matcher{
		config:         cfg,
		gitignoreRules: []string{"*.log"},
	}

	tests := []struct {
		name          string
		filePath      string
		expectedType  FileType
		expectedRules []string
	}{
		{
			name:          "go implementation file",
			filePath:      "src/main.go",
			expectedType:  FileTypeImpl,
			expectedRules: []string{"go-files"},
		},
		{
			name:          "test file excluded",
			filePath:      "src/main_test.go",
			expectedType:  FileTypeIgnored,
			expectedRules: nil,
		},
		{
			name:          "spec file",
			filePath:      "specs/api.md",
			expectedType:  FileTypeSpec,
			expectedRules: []string{"go-files"},
		},
		{
			name:          "ignored log file",
			filePath:      "debug.log",
			expectedType:  FileTypeIgnored,
			expectedRules: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.matchFile(tt.filePath)
			if result.Type != tt.expectedType {
				t.Errorf("matchFile(%q).Type = %v, expected %v", tt.filePath, result.Type, tt.expectedType)
			}
			if !reflect.DeepEqual(result.MatchedRules, tt.expectedRules) {
				t.Errorf("matchFile(%q).MatchedRules = %v, expected %v", tt.filePath, result.MatchedRules, tt.expectedRules)
			}
		})
	}
}

func TestMatcher_deduplicate(t *testing.T) {
	matcher := &Matcher{}

	tests := []struct {
		name     string
		items    []string
		exclude  string
		expected []string
	}{
		{
			name:     "remove duplicates and exclude",
			items:    []string{"a.go", "b.go", "a.go", "c.go", "b.go"},
			exclude:  "c.go",
			expected: []string{"a.go", "b.go"},
		},
		{
			name:     "no duplicates",
			items:    []string{"a.go", "b.go", "c.go"},
			exclude:  "d.go",
			expected: []string{"a.go", "b.go", "c.go"},
		},
		{
			name:     "empty input",
			items:    []string{},
			exclude:  "a.go",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.deduplicate(tt.items, tt.exclude)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("deduplicate(%v, %q) = %v, expected %v", tt.items, tt.exclude, result, tt.expected)
			}
		})
	}
}

func TestMatcher_MatchFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"main.go",
		"main_test.go",
		"specs/api.md",
		"debug.log",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tmpDir, file)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Create .gitignore
	gitignoreContent := "*.log\n"
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:    "go-files",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"**/*.go"},
					Exclude: []string{"**/*_test.go"},
				},
				Specs: []config.Spec{
					{
						Path: "./specs/*.md",
					},
				},
			},
		},
	}

	matcher, err := NewMatcher(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewMatcher failed: %v", err)
	}

	inputFiles := []string{"main.go", "main_test.go", "specs/api.md", "debug.log"}
	results, err := matcher.MatchFiles(inputFiles)
	if err != nil {
		t.Fatalf("MatchFiles failed: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("Expected 4 results, got %d", len(results))
	}

	// Verify results
	expectedTypes := map[string]FileType{
		"main.go":      FileTypeImpl,
		"main_test.go": FileTypeIgnored,
		"specs/api.md": FileTypeSpec,
		"debug.log":    FileTypeIgnored,
	}

	for _, result := range results {
		expectedType, exists := expectedTypes[result.Path]
		if !exists {
			t.Errorf("Unexpected file in results: %s", result.Path)
			continue
		}
		if result.Type != expectedType {
			t.Errorf("File %s: expected type %v, got %v", result.Path, expectedType, result.Type)
		}
	}
}
