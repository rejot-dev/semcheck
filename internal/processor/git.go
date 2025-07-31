package processor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

var (
	cmdStagedFiles = []string{"git", "diff", "--name-only", "--cached", "--diff-filter=ACMR"}
)

func GetStagedFiles(workingDir string) []string {
	cmd := exec.Command(cmdStagedFiles[0], cmdStagedFiles[1:]...)
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}
	}

	return files
}

func LoadGitignore(workingDir string) ([]string, error) {
	gitignorePath := filepath.Join(workingDir, ".gitignore")
	return loadIgnoreFile(gitignorePath)
}

func LoadSemignore(workingDir string) ([]string, error) {
	semignorePath := filepath.Join(workingDir, ".semignore")
	return loadIgnoreFile(semignorePath)
}

func loadIgnoreFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open ignore file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Warn("failed to close file: %v\n", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	var ignoreRules []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ignoreRules = append(ignoreRules, line)
	}

	return ignoreRules, scanner.Err()
}
