package processor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	cmdStagedFiles = []string{"git", "diff", "--name-only", "--cached", "--diff-filter=ACMR"}
)

func (m *Matcher) GetStagedFiles() []string {
	cmd := exec.Command(cmdStagedFiles[0], cmdStagedFiles[1:]...)
	cmd.Dir = m.workingDir

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

func (m *Matcher) LoadGitignore() error {
	gitignorePath := filepath.Join(m.workingDir, ".gitignore")

	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m.gitignoreRules = append(m.gitignoreRules, line)
	}

	return scanner.Err()
}
