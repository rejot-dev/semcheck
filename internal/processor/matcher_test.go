package processor

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rejot-dev/semcheck/internal/config"
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
		// Test relative path equivalence
		{
			name:     "relative prefix pattern vs normal file path",
			filePath: "local/file.md",
			pattern:  "./local/file.md",
			expected: true,
		},
		{
			name:     "normal pattern vs relative prefix file path",
			filePath: "./local/file.md",
			pattern:  "local/file.md",
			expected: true,
		},
		{
			name:     "both with relative prefix",
			filePath: "./local/file.md",
			pattern:  "./local/file.md",
			expected: true,
		},
		{
			name:     "relative prefix glob pattern vs normal file path",
			filePath: "local/file.md",
			pattern:  "./local/*.md",
			expected: true,
		},
		{
			name:     "normal glob pattern vs relative prefix file path",
			filePath: "./local/file.md",
			pattern:  "local/*.md",
			expected: true,
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

func TestMatcher_matchesPatterns(t *testing.T) {
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
			result := matcher.matchesPatterns(tt.filePath, matcher.gitignoreRules)
			if result != tt.expected {
				t.Errorf("matchesPatterns(%q, gitignore) = %v, expected %v", tt.filePath, result, tt.expected)
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
		name         string
		filePath     string
		expectedType FileType
		expectedRule string
	}{
		{
			name:         "go implementation file",
			filePath:     "src/main.go",
			expectedType: FileTypeImpl,
			expectedRule: "go-files",
		},
		{
			name:         "test file excluded",
			filePath:     "src/main_test.go",
			expectedType: FileTypeIgnored,
			expectedRule: "",
		},
		{
			name:         "spec file",
			filePath:     "specs/api.md",
			expectedType: FileTypeSpec,
			expectedRule: "go-files",
		},
		{
			name:         "ignored log file",
			filePath:     "debug.log",
			expectedType: FileTypeIgnored,
			expectedRule: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := matcher.matchFile(tt.filePath)
			if len(results) == 0 {
				t.Errorf("matchFile(%q) returned no results", tt.filePath)
				return
			}
			result := results[0] // For these simple tests, expect one result
			if result.Type != tt.expectedType {
				t.Errorf("matchFile(%q).Type = %v, expected %v", tt.filePath, result.Type, tt.expectedType)
			}
			if result.RuleName != tt.expectedRule {
				t.Errorf("matchFile(%q).RuleName = %v, expected %v", tt.filePath, result.RuleName, tt.expectedRule)
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
		expectedType, exists := expectedTypes[string(result.Path)]
		if !exists {
			t.Errorf("Unexpected file in results: %s", result.Path)
			continue
		}
		if result.Type != expectedType {
			t.Errorf("File %s: expected type %v, got %v", result.Path, expectedType, result.Type)
		}
	}
}

func TestMatcher_GetCounterparts(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:    "test-rule",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"**/*.go"},
				},
				Specs: []config.Spec{
					{Path: "spec1.md"},
					{Path: "spec2.md"},
				},
			},
		},
	}

	matcher := &Matcher{
		config: cfg,
		implFiles: RuleFileMap{
			"test-rule": []NormalizedPath{"impl1.go", "impl2.go"},
		},
	}

	// Test getting impl files for rule
	implFiles := matcher.GetRuleImplFiles("test-rule")
	expectedImpl := []NormalizedPath{"impl1.go", "impl2.go"}
	if !reflect.DeepEqual(implFiles, expectedImpl) {
		t.Errorf("GetRuleImplFiles: expected %v, got %v", expectedImpl, implFiles)
	}

	// Test getting spec files for rule
	specFiles := matcher.GetRuleSpecFiles("test-rule")
	expectedSpec := []NormalizedPath{"spec1.md", "spec2.md"}
	if !reflect.DeepEqual(specFiles, expectedSpec) {
		t.Errorf("GetRuleSpecFiles: expected %v, got %v", expectedSpec, specFiles)
	}

	// Test non-existent rule
	nilResult := matcher.GetRuleImplFiles("non-existent")
	if nilResult != nil {
		t.Errorf("GetRuleImplFiles for non-existent rule: expected nil, got %v", nilResult)
	}
}

func TestMatcher_MultipleRulesPerFile(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:    "go-specs",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"**/*.go"},
				},
				Specs: []config.Spec{
					{Path: "docs/*.md"},
				},
			},
			{
				Name:    "markdown-impl",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"**/*.md"},
				},
				Specs: []config.Spec{
					{Path: "specs/*.txt"},
				},
			},
		},
	}

	matcher := &Matcher{
		config: cfg,
	}

	// Test a file that matches as impl in one rule and spec in another
	results := matcher.matchFile("docs/api.md")

	// Should return 2 results - one for each rule with different types
	if len(results) != 2 {
		t.Errorf("Expected 2 results for file matching multiple rules, got %d", len(results))
	}

	// Verify the results have correct types for each rule
	ruleTypes := make(map[string]FileType)
	for _, result := range results {
		ruleTypes[result.RuleName] = result.Type
	}

	// For go-specs rule, docs/*.md should be a spec file
	if ruleTypes["go-specs"] != FileTypeSpec {
		t.Errorf("Expected FileTypeSpec for go-specs rule, got %v", ruleTypes["go-specs"])
	}

	// For markdown-impl rule, *.md should be an impl file
	if ruleTypes["markdown-impl"] != FileTypeImpl {
		t.Errorf("Expected FileTypeImpl for markdown-impl rule, got %v", ruleTypes["markdown-impl"])
	}

	// Verify both results have the same path
	for _, result := range results {
		if string(result.Path) != "docs/api.md" {
			t.Errorf("Expected path 'docs/api.md', got '%s'", result.Path)
		}
	}
}

func TestMatcher_PathNormalizationDuplicates(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:    "test-rule",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"src/**/*.go"}, // Different pattern to avoid overlap
				},
				Specs: []config.Spec{
					{Path: "./docs/*.md"},
				},
			},
		},
	}

	matcher := &Matcher{
		config: cfg,
	}

	// Test both normalized and non-normalized paths via MatchFiles (which should deduplicate)
	inputFiles := []string{
		"docs/api.md",
		"./docs/api.md", // Same file with different path representation
	}

	results, err := matcher.MatchFiles(inputFiles)
	if err != nil {
		t.Fatalf("MatchFiles failed: %v", err)
	}

	// Both inputs should be processed since MatchFiles deduplicates
	if len(results) != 1 {
		t.Errorf("Expected 1 results, got %d", len(results))
		for i, result := range results {
			t.Logf("Result %d: Path=%s, RuleName=%s, Type=%d", i, result.Path, result.RuleName, result.Type)
		}
		return
	}

	// Both results should have the same normalized path
	for _, result := range results {
		if string(result.Path) != "docs/api.md" {
			t.Errorf("Expected normalized path 'docs/api.md', got '%s'", result.Path)
		}
	}
}

func TestMatcher_FileMatchingBothSpecAndImpl(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:    "overlapping-rule",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"**/*.md"}, // This includes docs/api.md as impl
				},
				Specs: []config.Spec{
					{Path: "docs/*.md"}, // This includes docs/api.md as spec
				},
			},
		},
	}

	matcher := &Matcher{
		config: cfg,
	}

	// Test a file that matches both as spec and impl in the same rule
	results := matcher.matchFile("docs/api.md")

	// This SHOULD return 2 results - one as spec, one as impl
	if len(results) != 2 {
		t.Errorf("Expected 2 results (spec and impl), got %d", len(results))
	}

	// Verify we have both types
	types := make(map[FileType]bool)
	for _, result := range results {
		types[result.Type] = true
		if result.RuleName != "overlapping-rule" {
			t.Errorf("Expected rule name 'overlapping-rule', got '%s'", result.RuleName)
		}
		if string(result.Path) != "docs/api.md" {
			t.Errorf("Expected path 'docs/api.md', got '%s'", result.Path)
		}
	}

	if !types[FileTypeSpec] {
		t.Errorf("Expected to find FileTypeSpec result")
	}
	if !types[FileTypeImpl] {
		t.Errorf("Expected to find FileTypeImpl result")
	}
}

func TestMatcher_GetCounterpartsNormalization(t *testing.T) {
	cfg := &config.Config{
		Rules: []config.Rule{
			{
				Name:    "test-rule",
				Enabled: true,
				Files: config.FilePattern{
					Include: []string{"**/*.go"},
				},
				Specs: []config.Spec{
					{Path: "./specs/api.md"}, // Non-normalized path in config
					{Path: "specs/other.md"}, // Already normalized path in config
				},
			},
		},
	}

	matcher := &Matcher{
		config: cfg,
		implFiles: RuleFileMap{
			"test-rule": []NormalizedPath{"internal/api.go", "internal/other.go"},
		},
	}

	// Test getting spec files for rule (should be normalized)
	specFiles := matcher.GetRuleSpecFiles("test-rule")
	expected := []NormalizedPath{"specs/api.md", "specs/other.md"} // Both should be normalized

	if len(specFiles) != len(expected) {
		t.Errorf("Expected %d spec files, got %d", len(expected), len(specFiles))
	}

	for i, expectedPath := range expected {
		if i >= len(specFiles) {
			break
		}
		if specFiles[i] != expectedPath {
			t.Errorf("Expected spec file %s, got %s", expectedPath, specFiles[i])
		}
	}

	// Verify no ./ prefixes in results
	for _, path := range specFiles {
		if len(string(path)) >= 2 && string(path)[:2] == "./" {
			t.Errorf("Found non-normalized path in spec files: %s", path)
		}
	}
}
