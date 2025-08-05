package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rejot-dev/semcheck/internal/inline"
)

func TestFindAllInlineReferences(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create the specs directory and spec file
	specsDir := filepath.Join(tmpDir, "specs")
	err := os.MkdirAll(specsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create specs directory: %v", err)
	}

	specFile := filepath.Join(specsDir, "api.md")
	err = os.WriteFile(specFile, []byte("# API Spec\nThis is a test spec."), 0644)
	if err != nil {
		t.Fatalf("Failed to create spec file: %v", err)
	}

	// Create test files with inline references
	testFile1 := filepath.Join(tmpDir, "test1.go")
	testContent1 := `package main

// This is a test file
// semcheck:file(./specs/api.md)
func main() {
	// semcheck:rfc(1234)
	println("hello")
}`

	err = os.WriteFile(testFile1, []byte(testContent1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := filepath.Join(tmpDir, "test2.go")
	testContent2 := `package util

// semcheck:url(https://example.com)
func helper() {
}`

	err = os.WriteFile(testFile2, []byte(testContent2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to the test directory so relative paths work
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Call FindAllInlineReferences
	result, err := FindAllInlineReferences(tmpDir)
	if err != nil {
		t.Fatalf("FindAllInlineReferences failed: %v", err)
	}

	// Verify results
	if len(result) != 2 {
		t.Errorf("Expected 2 files with references, got %d", len(result))
	}

	// Check test1.go references
	test1Refs := result[NormalizedPath("test1.go")]
	if len(test1Refs) != 2 {
		t.Errorf("Expected 2 references in test1.go, got %d", len(test1Refs))
	}

	if test1Refs[0].Command != inline.File || len(test1Refs[0].Args) == 0 || test1Refs[0].Args[0] != "./specs/api.md" {
		t.Errorf("Expected file reference to ./specs/api.md, got %v", test1Refs[0])
	}

	if test1Refs[1].Command != inline.RFC || len(test1Refs[1].Args) == 0 || test1Refs[1].Args[0] != "https://datatracker.ietf.org/doc/html/rfc1234" {
		t.Errorf("Expected RFC reference to https://datatracker.ietf.org/doc/html/rfc1234, got %v", test1Refs[1])
	}

	// Check test2.go references
	test2Refs := result[NormalizedPath("test2.go")]
	if len(test2Refs) != 1 {
		t.Errorf("Expected 1 reference in test2.go, got %d", len(test2Refs))
	}

	if test2Refs[0].Command != inline.URL || len(test2Refs[0].Args) == 0 || test2Refs[0].Args[0] != "https://example.com" {
		t.Errorf("Expected URL reference to https://example.com, got %v", test2Refs[0])
	}
}
