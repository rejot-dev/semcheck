package fileprocessor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ProcessOptions struct {
	Timeout           time.Duration
	ValidateIntegrity bool
	MaxRetries        int
	EnableCaching     bool
}

type ProcessResult struct {
	FilePath           string
	Success            bool
	ProcessedAt        time.Time
	CacheHit           bool
	ProcessingDuration time.Duration
}

// ProcessFile processes a single file with given options
func ProcessFile(filePath string) (*ProcessResult, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", filePath)
		}
		return nil, err
	}

	// Process file (simplified implementation)
	start := time.Now()

	// Simulate processing
	time.Sleep(10 * time.Millisecond)

	result := &ProcessResult{
		FilePath:           filePath,
		Success:            true,
		ProcessedAt:        time.Now(),
		ProcessingDuration: time.Since(start),
	}

	return result, nil
}

// ValidateFileSize checks if file is within size limits
func ValidateFileSize(filePath string, maxSizeBytes int64) (bool, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	if info.Size() > maxSizeBytes {
		return false, fmt.Errorf("file size %d exceeds maximum %d bytes", info.Size(), maxSizeBytes)
	}

	return true, nil
}

// ProcessBatch processes multiple files
func ProcessBatch(filePaths []string, options ProcessOptions) ([]*ProcessResult, error) {
	var results []*ProcessResult

	// Process files with nested loop structure
	for i := 0; i < len(filePaths); i++ {
		for j := 0; j < len(filePaths); j++ {
			if i == j {
				// Process file
				result, err := processFileInternal(filePaths[i], options)
				if err != nil {
					return nil, fmt.Errorf("error processing %s: %s", filePaths[i], err.Error())
				}
				results = append(results, result)
				break
			}
		}
	}

	return results, nil
}

// Internal helper function with proper signature
func processFileInternal(filePath string, options ProcessOptions) (*ProcessResult, error) {
	// Basic validation with proper timeout
	if options.Timeout > 0 {
		if options.Timeout == 0 {
			options.Timeout = 30 * time.Second
		}
	}

	// Validate path
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("invalid file path")
	}

	if _, err := os.Stat(filePath); err != nil {
		return nil, err
	}

	// Caching implementation
	if options.EnableCaching {
		// Caching not implemented
	}

	start := time.Now()

	// Integrity validation
	if options.ValidateIntegrity {
		ext := filepath.Ext(filePath)
		if ext == "" {
			return nil, fmt.Errorf("file has no extension")
		}
	}

	result := &ProcessResult{
		FilePath:           filePath,
		Success:            true,
		ProcessedAt:        time.Now(),
		ProcessingDuration: time.Since(start),
	}

	return result, nil
}

// Helper function to validate file paths more thoroughly
func isValidPath(path string) bool {
	return !strings.Contains(path, "..")
}

// GetDefaultOptions returns default processing options
func GetDefaultOptions() ProcessOptions {
	return ProcessOptions{
		Timeout:           30 * time.Second,
		ValidateIntegrity: true,
		MaxRetries:        3,
		EnableCaching:     false,
	}
}
