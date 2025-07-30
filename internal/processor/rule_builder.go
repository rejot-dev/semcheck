package processor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rejot-dev/semcheck/internal/inline"
)

var IgnoredPaths = []string{"**/.git",
	"**/.svn",
	"**/.hg",
	"**/.jj",
	"**/CVS",
	"**/.DS_Store",
	"**/Thumbs.db",
	"**/.classpath",
	"**/.settings"}

func FindAllInlineReferences(workingDir string) (map[NormalizedPath][]inline.InlineReference, error) {
	// Traverse all files in the working directory, read them, and collect inline spec references
	// Skip files in the tree that are ignored by Gitignore rules, or mentioned in IgnorePaths
	gitIgnoreRules, err := LoadGitignore(workingDir)
	if err != nil {
		return nil, err
	}

	semIgnoreRules, err := LoadSemignore(workingDir)
	if err != nil {
		return nil, err
	}

	allIgnoredPatterns := append(gitIgnoreRules, IgnoredPaths...)
	allIgnoredPatterns = append(allIgnoredPatterns, semIgnoreRules...)

	allReferences := make(map[NormalizedPath][]inline.InlineReference)

	err = filepath.WalkDir(workingDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(workingDir, path)
		if err != nil {
			return err
		}

		// Check if file should be ignored
		if MatchesPatterns(relPath, allIgnoredPatterns) {
			return nil // Skip this file
		}

		refs := FindInlineReferencesInFile(path)

		// Store references grouped by file path
		if len(refs) > 0 {
			normalizedPath := NormalizePath(relPath)
			allReferences[normalizedPath] = refs
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to traverse directory: %w", err)
	}

	return allReferences, nil
}

func FindInlineReferencesInFile(path string) []inline.InlineReference {
	content, err := os.ReadFile(path)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read file, won't check for inline spec references: %s\n", path)
		return nil
	}

	// Parse file for inline references
	refs, inlineErrors := inline.FindReferences(string(content))

	for _, parseError := range inlineErrors {
		// Log warnings for argument errors only, ignore the rest
		if parseError.Err != inline.ErrorInvalidCommand {
			fmt.Fprintf(os.Stderr, "Warning: failed to process inline reference in %s: %s\n", path, parseError.Format())
		}
	}
	return refs
}
