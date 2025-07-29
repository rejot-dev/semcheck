package processor

import (
	"path/filepath"
	"strings"
)

// MatchesPatterns checks if a file path matches any of the given patterns
func MatchesPatterns(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if MatchesPattern(filePath, pattern) {
			return true
		}
	}
	return false
}

// MatchesPattern checks if a file path matches a single pattern
func MatchesPattern(filePath, pattern string) bool {
	// Normalize both file path and pattern by removing ./ prefix
	const relativePrefixLen = len("./")
	if strings.HasPrefix(pattern, "./") {
		pattern = pattern[relativePrefixLen:]
	}
	if strings.HasPrefix(filePath, "./") {
		filePath = filePath[relativePrefixLen:]
	}

	// Handle exact matches
	if pattern == filePath {
		return true
	}

	// Handle simple glob patterns
	matched, err := filepath.Match(pattern, filePath)
	if err == nil && matched {
		return true
	}

	// Handle glob patterns with **
	if strings.Contains(pattern, "**") {
		return matchesGlobPattern(filePath, pattern)
	}

	// Handle directory-based patterns
	if strings.Contains(pattern, "/") {
		return matchesPathPattern(filePath, pattern)
	}

	return false
}

// matchesGlobPattern handles ** glob patterns
func matchesGlobPattern(filePath, pattern string) bool {
	// Handle patterns like **/.git, **/.DS_Store
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		// Check if file matches suffix directly (for root level files)
		matched, _ := filepath.Match(suffix, filePath)
		return matched ||
			strings.HasSuffix(filePath, suffix) ||
			strings.Contains(filePath, "/"+suffix)
	}

	// Handle patterns like .git/**, temp/**
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

// matchesPathPattern handles directory-based patterns
func matchesPathPattern(filePath, pattern string) bool {
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
