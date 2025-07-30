package processor

import (
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
	IgnoreReasonIgnore
	IgnoreReasonExcludedByRule
	IgnoreReasonNoRuleMatch
)

func (r IgnoreReason) String() string {
	switch r {
	case IgnoreReasonIgnore:
		return "ignore"
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
	config      *config.Config
	ignoreRules []string
	implFiles   RuleFileMap
	workingDir  string
}

func NewMatcher(cfg *config.Config, workingDir string) (*Matcher, error) {
	m := &Matcher{
		config:     cfg,
		implFiles:  make(RuleFileMap),
		workingDir: workingDir,
	}

	gitignoreRules, err := LoadGitignore(m.workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load gitignore: %w", err)
	}
	semignoreRules, err := LoadSemignore(m.workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load semignore: %w", err)
	}
	m.ignoreRules = append(gitignoreRules, semignoreRules...)
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
			if MatchesPatterns(relPath, rule.Files.Exclude) {
				return nil
			}

			if MatchesPatterns(relPath, rule.Files.Include) {
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

// TODO: GetAllMatcherResults doesn't use MatchFile function, therefore there is some code duplication here
// Returns all implementation and specification files from all rules
func (m *Matcher) GetAllMatcherResults() []MatcherResult {
	var results []MatcherResult
	seen := make(map[NormalizedPath]bool)

	for ruleName, implFiles := range m.implFiles {
		rule := m.findRule(ruleName)
		if rule == nil {
			continue
		}
		for _, spec := range rule.Specs {
			results = append(results, MatcherResult{
				Path:         NormalizedPath(spec.Path),
				Type:         FileTypeSpec,
				RuleName:     rule.Name,
				IgnoreReason: IgnoreReasonNone,
			})
		}

		for _, implFile := range implFiles {
			if !seen[implFile] {
				seen[implFile] = true
				results = append(results, MatcherResult{
					Path:         implFile,
					Type:         FileTypeImpl,
					RuleName:     ruleName,
					IgnoreReason: IgnoreReasonNone,
				})
			}
		}
	}

	inlineSeen := make(map[NormalizedPath]bool)
	inlineSpecs, err := FindAllInlineReferences(m.workingDir)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for path, specs := range inlineSpecs {
		if inlineSeen[path] {
			continue
		}
		inlineSeen[path] = true
		results = append(results, MatcherResult{
			Path:         path,
			Type:         FileTypeImpl,
			RuleName:     "inline-ref",
			IgnoreReason: IgnoreReasonNone,
		})
		for _, spec := range specs {
			normPath := NormalizePath(spec.Args[0])
			if inlineSeen[normPath] {
				continue
			}
			inlineSeen[normPath] = true
			results = append(results, MatcherResult{
				Path:         normPath,
				Type:         FileTypeSpec,
				RuleName:     "inline-ref",
				IgnoreReason: IgnoreReasonNone,
			})
		}
	}

	return results
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

	// Check if file should be ignored
	if MatchesPatterns(filePath, m.ignoreRules) {
		return []MatcherResult{{
			Path:         normalizedPath,
			Type:         FileTypeIgnored,
			IgnoreReason: IgnoreReasonIgnore,
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
		if MatchesPatterns(filePath, rule.Files.Exclude) {
			ruleExcluded = true
			continue
		}

		// Check if file matches rule's include patterns (impl file)
		if MatchesPatterns(filePath, rule.Files.Include) {
			results = append(results, MatcherResult{
				Path:         normalizedPath,
				Type:         FileTypeImpl,
				RuleName:     rule.Name,
				IgnoreReason: IgnoreReasonNone,
			})
		}

		// Check if file matches any spec patterns
		for _, spec := range rule.Specs {
			if MatchesPattern(filePath, spec.Path) {
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

	// Check for inline spec references
	refs := FindInlineReferencesInFile(filePath)

	for _, ref := range refs {
		results = append(results, MatcherResult{
			Path:         normalizedPath,
			Type:         FileTypeImpl,
			RuleName:     "inline-ref",
			IgnoreReason: IgnoreReasonNone,
		})
		results = append(results, MatcherResult{
			Path:         NormalizePath(ref.Args[0]), // TODO(doesn't work for rfc's just yet)
			Type:         FileTypeSpec,
			RuleName:     "inline-ref",
			IgnoreReason: IgnoreReasonNone,
		})
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

func DisplayMatchResults(matchedResults []MatcherResult) {
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
}
