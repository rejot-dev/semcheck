package providers

import (
	"context"
	"time"
)

// Response represents the response from an AI provider
type Response struct {
	Usage  Usage           `json:"usage"`
	Issues []SemanticIssue `json:"issues,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Request represents a request to an AI provider
type Request struct {
	Prompt      string        `json:"prompt"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
}

// Client defines the interface for AI providers
type Client interface {
	// Complete sends a completion request to the AI provider
	Complete(ctx context.Context, req *Request) (*Response, error)

	// Name returns the name of the provider
	Name() string

	// Validate checks if the client configuration is valid
	Validate() error
}

// SemanticIssue represents a single issue found during semantic analysis
type SemanticIssue struct {
	Level      string  `json:"level" jsonschema_description:"Severity level of the issue"`
	Message    string  `json:"message" jsonschema_description:"Description of the issue"`
	Confidence float64 `json:"confidence" jsonschema_description:"Confidence level of the issue (0.0-1.0)"`
	Suggestion string  `json:"suggestion" jsonschema_description:"Suggestion for fixing the issue (optional)"`
	LineNumber int     `json:"line_number,omitempty" jsonschema_description:"Line number of the issue (optional)"`
}

// Config holds common configuration for AI providers
type Config struct {
	Provider   string        `yaml:"provider"`
	Model      string        `yaml:"model"`
	APIKey     string        `yaml:"api_key"`
	BaseURL    string        `yaml:"base_url,omitempty"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int           `yaml:"max_retries"`
}
