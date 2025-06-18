package processor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rejot.dev/semcheck/internal/config"
)

type FileType int

const (
	FileTypeIgnored FileType = iota
	FileTypeSpec
	FileTypeImpl
)

type IgnoreReason int

const (
	IgnoreReasonNone IgnoreReason = iota
	IgnoreReasonGitignore
	IgnoreReasonExcludedByRule
	IgnoreReasonNoRuleMatch
)

func (r IgnoreReason) String() string {
	switch r {
	case IgnoreReasonGitignore:
		return "gitignore"
	case IgnoreReasonExcludedByRule:
		return "excluded by rule"
	case IgnoreReasonNoRuleMatch:
		return "no rule match"
	default:
		return "unknown"
	}
}

type MatcherResult struct {
	Path         string
	Type         FileType
	MatchedRules []string
	RelatedFiles []string
	IgnoreReason IgnoreReason // Only set when Type == FileTypeIgnored
}

type Matcher struct {
	config         *config.Config
	gitignoreRules []string
	workingDir     string
}

func NewMatcher(cfg *config.Config, workingDir string) (*Matcher, error) {
	m := &Matcher{
		config:     cfg,
		workingDir: workingDir,
	}

	if err := m.loadGitignore(); err != nil {
		return nil, fmt.Errorf("failed to load gitignore: %w", err)
	}

	return m, nil
}

func (m *Matcher) loadGitignore() error {
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

func (m *Matcher) MatchFiles(inputFiles []string) ([]MatcherResult, error) {
	var results []MatcherResult

	for _, file := range inputFiles {
		matched := m.matchFile(file)
		results = append(results, matched)
	}

	// Find related files for each matched file
	for i := range results {
		if results[i].Type != FileTypeIgnored {
			related, err := m.findRelatedFiles(results[i])
			if err != nil {
				return nil, fmt.Errorf("failed to find related files for %s: %w", results[i].Path, err)
			}
			results[i].RelatedFiles = related
		}
	}

	return results, nil
}

func (m *Matcher) matchFile(filePath string) MatcherResult {
	matched := MatcherResult{
		Path: filePath,
		Type: FileTypeIgnored,
	}

	// Check if file should be ignored by gitignore
	if m.isIgnoredByGitignore(filePath) {
		matched.IgnoreReason = IgnoreReasonGitignore
		return matched
	}

	// Check against rules
	var ruleExcluded bool
	for _, rule := range m.config.Rules {
		if !rule.Enabled {
			continue
		}

		// Check if file matches rule's exclude patterns
		if m.matchesPatterns(filePath, rule.Files.Exclude) {
			ruleExcluded = true
			continue
		}

		// Check if file matches rule's include patterns
		if m.matchesPatterns(filePath, rule.Files.Include) {
			matched.Type = FileTypeImpl
			matched.MatchedRules = append(matched.MatchedRules, rule.Name)
			matched.IgnoreReason = IgnoreReasonNone // Clear ignore reason since it matched
		}

		// Check if file matches any spec patterns
		for _, spec := range rule.Specs {
			if m.matchesPattern(filePath, spec.Path) {
				matched.Type = FileTypeSpec
				matched.MatchedRules = append(matched.MatchedRules, rule.Name)
				matched.IgnoreReason = IgnoreReasonNone // Clear ignore reason since it matched
			}
		}
	}

	// Set ignore reason if file was ignored
	if matched.Type == FileTypeIgnored {
		if ruleExcluded {
			matched.IgnoreReason = IgnoreReasonExcludedByRule
		} else {
			matched.IgnoreReason = IgnoreReasonNoRuleMatch
		}
	}

	return matched
}

func (m *Matcher) isIgnoredByGitignore(filePath string) bool {
	for _, pattern := range m.gitignoreRules {
		if m.matchesPattern(filePath, pattern) {
			return true
		}
	}
	return false
}

func (m *Matcher) matchesPatterns(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if m.matchesPattern(filePath, pattern) {
			return true
		}
	}
	return false
}

func (m *Matcher) matchesPattern(filePath, pattern string) bool {
	// Handle patterns that start with ./
	const relativePrefixLen = len("./")
	if strings.HasPrefix(pattern, "./") {
		pattern = pattern[relativePrefixLen:]
	}

	matched, err := filepath.Match(pattern, filePath)
	if err != nil {
		return false
	}
	if matched {
		return true
	}

	// Handle glob patterns with **
	if strings.Contains(pattern, "**") {
		return m.matchesGlobPattern(filePath, pattern)
	}

	// Handle directory-based patterns
	if strings.Contains(pattern, "/") {
		return m.matchesPathPattern(filePath, pattern)
	}

	return false
}

func (m *Matcher) matchesGlobPattern(filePath, pattern string) bool {
	// Simple glob matching for ** patterns
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		// Check if file matches suffix directly (for root level files)
		matched, _ := filepath.Match(suffix, filePath)
		return matched ||
			strings.HasSuffix(filePath, suffix) ||
			strings.Contains(filePath, "/"+suffix)
	}

	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		return strings.HasPrefix(filePath, prefix+"/") ||
			strings.Contains(filePath, "/"+prefix+"/") ||
			strings.HasPrefix(filePath, prefix)
	}

	// For patterns with ** in the middle
	if strings.Contains(pattern, "**/") {
		parts := strings.Split(pattern, "**/")
		if len(parts) == 2 {
			return strings.HasPrefix(filePath, parts[0]) && strings.HasSuffix(filePath, parts[1])
		}
	}

	return false
}

func (m *Matcher) matchesPathPattern(filePath, pattern string) bool {
	// Handle patterns with directory separators
	matched, err := filepath.Match(pattern, filePath)
	if err != nil {
		return false
	}
	if matched {
		return true
	}

	// For simple directory patterns like "temp/"
	if strings.HasSuffix(pattern, "/") {
		dirName := strings.TrimSuffix(pattern, "/")
		return strings.HasPrefix(filePath, pattern) ||
			strings.Contains(filePath, "/"+pattern) ||
			strings.HasPrefix(filePath, dirName+"/")
	}

	return false
}

func (m *Matcher) findRelatedFiles(file MatcherResult) ([]string, error) {
	var related []string

	for _, rule := range m.config.Rules {
		// Skip if this file doesn't match this rule
		found := false
		for _, matchedRule := range file.MatchedRules {
			if matchedRule == rule.Name {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		if file.Type == FileTypeImpl {
			// Find associated spec files
			for _, spec := range rule.Specs {
				specFiles, err := m.expandGlobPattern(spec.Path)
				if err != nil {
					return nil, fmt.Errorf("failed to expand spec pattern %s: %w", spec.Path, err)
				}
				related = append(related, specFiles...)
			}
		} else if file.Type == FileTypeSpec {
			// Find associated implementation files
			implFiles, err := m.findImplementationFiles(rule)
			if err != nil {
				return nil, fmt.Errorf("failed to find implementation files: %w", err)
			}
			related = append(related, implFiles...)
		}
	}

	// Remove duplicates and the file itself
	return m.deduplicate(related, file.Path), nil
}

func (m *Matcher) expandGlobPattern(pattern string) ([]string, error) {
	var searchPattern string

	const relativePrefixLen = len("./")
	if strings.HasPrefix(pattern, "./") {
		searchPattern = filepath.Join(m.workingDir, pattern[relativePrefixLen:])
	} else if !filepath.IsAbs(pattern) {
		searchPattern = filepath.Join(m.workingDir, pattern)
	} else {
		searchPattern = pattern
	}

	matches, err := filepath.Glob(searchPattern)
	if err != nil {
		return nil, err
	}

	// Convert back to relative paths
	var result []string
	for _, match := range matches {
		rel, err := filepath.Rel(m.workingDir, match)
		if err == nil {
			result = append(result, rel)
		} else {
			result = append(result, match)
		}
	}

	return result, nil
}

func (m *Matcher) findImplementationFiles(rule config.Rule) ([]string, error) {
	var files []string

	err := filepath.WalkDir(m.workingDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(m.workingDir, path)
		if err != nil {
			return err
		}

		// Check if file matches this rule's patterns
		if m.matchesPatterns(relPath, rule.Files.Exclude) {
			return nil
		}

		if m.matchesPatterns(relPath, rule.Files.Include) {
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

func (m *Matcher) deduplicate(items []string, exclude string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range items {
		if item != exclude && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func DisplayMatchResults(matchedResults []MatcherResult) {
	fmt.Println("\n--- File Matching Results ---")

	var specFiles, implFiles, ignoredFiles []MatcherResult

	for _, file := range matchedResults {
		switch file.Type {
		case FileTypeSpec:
			specFiles = append(specFiles, file)
		case FileTypeImpl:
			implFiles = append(implFiles, file)
		case FileTypeIgnored:
			ignoredFiles = append(ignoredFiles, file)
		}
	}

	if len(specFiles) > 0 {
		fmt.Printf("\n📋 Specification Files (%d):\n", len(specFiles))
		for _, file := range specFiles {
			fmt.Printf("  • %s", file.Path)
			if len(file.MatchedRules) > 0 {
				fmt.Printf(" [rules: %v]", file.MatchedRules)
			}
			fmt.Println()
			if len(file.RelatedFiles) > 0 {
				fmt.Printf("    → Related implementations: %v\n", file.RelatedFiles)
			}
		}
	}

	if len(implFiles) > 0 {
		fmt.Printf("\n⚙️  Implementation Files (%d):\n", len(implFiles))
		for _, file := range implFiles {
			fmt.Printf("  • %s", file.Path)
			if len(file.MatchedRules) > 0 {
				fmt.Printf(" [rules: %v]", file.MatchedRules)
			}
			fmt.Println()
			if len(file.RelatedFiles) > 0 {
				fmt.Printf("    → Related specifications: %v\n", file.RelatedFiles)
			}
		}
	}

	if len(ignoredFiles) > 0 {
		fmt.Printf("\n🚫 Ignored Files (%d):\n", len(ignoredFiles))

		// Group by ignore reason
		reasonGroups := make(map[string][]MatcherResult)
		for _, file := range ignoredFiles {
			reason := file.IgnoreReason.String()
			reasonGroups[reason] = append(reasonGroups[reason], file)
		}

		for reason, files := range reasonGroups {
			fmt.Printf("  [%s]\n", reason)
			for _, file := range files {
				fmt.Printf("    • %s\n", file.Path)
			}
		}
	}

	fmt.Printf("\nSummary: %d specs, %d implementations, %d ignored\n",
		len(specFiles), len(implFiles), len(ignoredFiles))
}
