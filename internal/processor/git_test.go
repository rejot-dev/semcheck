package processor

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGit(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()

	if err != nil {
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	t.Run("test staging files", func(t *testing.T) {
		files := GetStagedFiles(tmpDir)
		if len(files) != 0 {
			t.Errorf("expected no staged files, got %v", files)
		}

		err = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		cmd = exec.Command("git", "add", "test.txt")
		cmd.Dir = tmpDir
		err = cmd.Run()

		if err != nil {
			t.Fatalf("Failed to stage file: %v", err)
		}

		files = GetStagedFiles(tmpDir)
		if len(files) != 1 {
			t.Errorf("expected 1 staged file, got %v", files)
		}
		if files[0] != "test.txt" {
			t.Errorf("expected file name 'test.txt', got %s", files[0])
		}
	})

	t.Run("test gitignore file loading", func(t *testing.T) {
		files, err := LoadGitignore(tmpDir)

		if err != nil {
			t.Fatalf("Failed to load gitignore file: %v", err)
		}

		if files != nil {
			t.Fatalf("expected no gitignore files, got %v", files)
		}

		err = os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("ola"), 0644)
		if err != nil {
			t.Fatalf("Failed to create gitignore file: %v", err)
		}

		files, err = LoadGitignore(tmpDir)
		if err != nil {
			t.Fatalf("Failed to load gitignore file: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("expected 1 gitignore file, got %v", files)
		}
		if files[0] != "ola" {
			t.Errorf("expected file name 'ola', got %s", files[0])
		}

	})
}
