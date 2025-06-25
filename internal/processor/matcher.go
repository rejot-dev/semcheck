package processor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejot-dev/semcheck/internal/config"
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
	Path         NormalizedPath
	Type         FileType
	RuleName     string
	IgnoreReason IgnoreReason
}

type NormalizedPath string

func NormalizedPathsToStrings(paths []NormalizedPath) []string {
	result := make([]string, len(paths))
	for i, path := range paths {
		result[i] = string(path)
	}
	return result
}

func NormalizePath(path string) NormalizedPath {
	if strings.HasPrefix(path, "./") {
		return NormalizedPath(path[2:])
	}
	return NormalizedPath(path)
}

// mapping rule names to a list of files
type RuleFileMap map[string][]NormalizedPath

type Matcher struct {
	config         *config.Config
	gitignoreRules []string
	implFiles      RuleFileMap
	workingDir     string
}

func NewMatcher(cfg *config.Config, workingDir string) (*Matcher, error) {
	m := &Matcher{
		config:     cfg,
		implFiles:  make(RuleFileMap),
		workingDir: workingDir,
	}

	if err := m.loadGitignore(); err != nil {
		return nil, fmt.Errorf("failed to load gitignore: %w", err)
	}

	if err := m.resolveImplFiles(); err != nil {
		return nil, fmt.Errorf("failed to resolve implementation files: %w", err)
	}

	return m, nil
}

func (m *Matcher) resolveImplFiles() error {
	// find all impl files in the current working directory by rule
	for _, rule := range m.config.Rules {
		if !rule.Enabled {
			continue
		}
		var files []NormalizedPath

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
				files = append(files, NormalizePath(relPath))
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to resolve impl files: %w", err)
		}

		m.implFiles[rule.Name] = files
	}

	return nil
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

// Should this really be a method on the matcher?
func (m *Matcher) GetRuleImplFiles(ruleName string) []NormalizedPath {
	return m.implFiles[ruleName]
}

// Should this really be a method on the matcher?
func (m *Matcher) GetRuleSpecFiles(ruleName string) []NormalizedPath {
	rule := m.findRule(ruleName)
	if rule == nil {
		return nil
	}
	specFiles := make([]NormalizedPath, len(rule.Specs))
	for i, spec := range rule.Specs {
		specFiles[i] = NormalizePath(spec.Path)
	}
	return specFiles
}

// findRule finds a rule by name
func (m *Matcher) findRule(name string) *config.Rule {

	for i := range m.config.Rules {
		if m.config.Rules[i].Name == name {
			return &m.config.Rules[i]
		}
	}
	return nil
}

func (m *Matcher) MatchFiles(inputFiles []string) ([]MatcherResult, error) {
	var results []MatcherResult
	seen := make(map[NormalizedPath]bool)

	for _, file := range inputFiles {
		fileResults := m.matchFile(file)
		for _, result := range fileResults {
			if !seen[result.Path] {
				seen[result.Path] = true
				results = append(results, result)
			}
		}
	}

	return results, nil
}

func (m *Matcher) matchFile(filePath string) []MatcherResult {
	normalizedPath := NormalizePath(filePath)

	// Check if file should be ignored by gitignore
	if m.matchesPatterns(filePath, m.gitignoreRules) {
		return []MatcherResult{{
			Path:         normalizedPath,
			Type:         FileTypeIgnored,
			IgnoreReason: IgnoreReasonGitignore,
		}}
	}

	var results []MatcherResult
	var ruleExcluded bool

	// Check against rules
	for _, rule := range m.config.Rules {
		if !rule.Enabled {
			continue
		}

		// Check if file matches rule's exclude patterns
		if m.matchesPatterns(filePath, rule.Files.Exclude) {
			ruleExcluded = true
			continue
		}

		// Check if file matches rule's include patterns (impl file)
		if m.matchesPatterns(filePath, rule.Files.Include) {
			results = append(results, MatcherResult{
				Path:         normalizedPath,
				Type:         FileTypeImpl,
				RuleName:     rule.Name,
				IgnoreReason: IgnoreReasonNone,
			})
		}

		// Check if file matches any spec patterns
		for _, spec := range rule.Specs {
			if m.matchesPattern(filePath, spec.Path) {
				results = append(results, MatcherResult{
					Path:         normalizedPath,
					Type:         FileTypeSpec,
					RuleName:     rule.Name,
					IgnoreReason: IgnoreReasonNone,
				})
				break // Only one spec match per rule is needed
			}
		}
	}

	// If no matches found, return ignored result
	if len(results) == 0 {
		ignoreReason := IgnoreReasonNoRuleMatch
		if ruleExcluded {
			ignoreReason = IgnoreReasonExcludedByRule
		}
		return []MatcherResult{{
			Path:         normalizedPath,
			Type:         FileTypeIgnored,
			IgnoreReason: ignoreReason,
		}}
	}

	return results
}

func (m *Matcher) matchesPatterns(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if m.matchesPattern(filePath, pattern) {
			return true
		}
	}
	return false
}

// TODO: this pattern matching for files probably doesn't need to be hand rolled like here.
func (m *Matcher) matchesPattern(filePath, pattern string) bool {
	// Normalize both file path and pattern by removing ./ prefix
	const relativePrefixLen = len("./")
	if strings.HasPrefix(pattern, "./") {
		pattern = pattern[relativePrefixLen:]
	}
	if strings.HasPrefix(filePath, "./") {
		filePath = filePath[relativePrefixLen:]
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
			hasPrefix := strings.HasPrefix(filePath, parts[0])
			// For the suffix, we need to match it as a pattern, not a literal string
			if hasPrefix {
				// Extract the part of the filePath after the prefix
				remaining := filePath[len(parts[0]):]
				// Check if the remaining part matches the suffix pattern
				matched, _ := filepath.Match(parts[1], remaining)
				if matched {
					return true
				}
				// Also check if any subdirectory matches
				if strings.Contains(remaining, "/") {
					pathParts := strings.Split(remaining, "/")
					for i := range pathParts {
						subPath := strings.Join(pathParts[i:], "/")
						if matched, _ := filepath.Match(parts[1], subPath); matched {
							return true
						}
					}
				}
			}
			return false
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
			if file.RuleName != "" {
				fmt.Printf(" [rule: %s]", file.RuleName)
			}
			fmt.Println()
		}
	}

	if len(implFiles) > 0 {
		fmt.Printf("\n⚙️  Implementation Files (%d):\n", len(implFiles))
		for _, file := range implFiles {
			fmt.Printf("  • %s", file.Path)
			if file.RuleName != "" {
				fmt.Printf(" [rule: %s]", file.RuleName)
			}
			fmt.Println()
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
