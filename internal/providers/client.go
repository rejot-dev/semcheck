package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/rejot-dev/semcheck/internal/config"
)

// Response represents the response from an AI provider
type Response struct {
	Usage  Usage
	Issues []SemanticIssue
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Request represents a request to an AI provider
type Request struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
	Temperature  float64
	Timeout      time.Duration
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
	Reasoning  string
	Level      string
	Message    string
	Confidence float64
	Suggestion string
	LineNumber int
}

// Config holds common configuration for AI providers
type Config struct {
	Provider   string
	Model      string
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

func CreateAIClient(cfg *config.Config) (Client, error) {
	// Convert config to provider config
	providerConfig := &Config{
		Provider:   cfg.Provider,
		Model:      cfg.Model,
		APIKey:     cfg.APIKey,
		BaseURL:    cfg.BaseURL,
		Timeout:    time.Duration(cfg.Timeout) * time.Second,
		MaxRetries: cfg.MaxRetries,
	}

	var client Client
	var err error

	switch cfg.Provider {
	case "openai":
		client, err = NewOpenAIClient(providerConfig)
	case "anthropic":
		client, err = NewAnthropicClient(providerConfig)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}
