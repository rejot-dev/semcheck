package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/rejot-dev/semcheck/internal/config"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
)

func ToProvider(provider string) (Provider, error) {
	switch provider {
	case "openai":
		return ProviderOpenAI, nil
	case "anthropic":
		return ProviderAnthropic, nil
	default:
		return "", fmt.Errorf("invalid provider: %s", provider)
	}
}

func GetAllProviders() []Provider {
	return []Provider{ProviderOpenAI, ProviderAnthropic}
}

type ProviderDefaults struct {
	Model     string
	ApiKeyVar string
}

func GetProviderDefaults(provider Provider) ProviderDefaults {
	switch provider {
	case ProviderOpenAI:
		return ProviderDefaults{
			Model:     "gpt-4o",
			ApiKeyVar: "OPENAI_API_KEY",
		}
	case ProviderAnthropic:
		return ProviderDefaults{
			Model:     "claude-sonnet-4-0",
			ApiKeyVar: "ANTHROPIC_API_KEY",
		}
	default:
		return ProviderDefaults{
			Model:     "<unknown>",
			ApiKeyVar: "<unknown>",
		}
	}
}

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
	Provider   Provider
	Model      string
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

func CreateAIClient(cfg *config.Config) (Client, error) {
	// Convert config to provider config

	provider, providerErr := ToProvider(cfg.Provider)
	if providerErr != nil {
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}

	providerConfig := &Config{
		Provider:   provider,
		Model:      cfg.Model,
		APIKey:     cfg.APIKey,
		BaseURL:    cfg.BaseURL,
		Timeout:    time.Duration(cfg.Timeout) * time.Second,
		MaxRetries: cfg.MaxRetries,
	}

	var client Client
	var err error

	switch provider {
	case ProviderOpenAI:
		client, err = NewOpenAIClient(providerConfig)
	case ProviderAnthropic:
		client, err = NewAnthropicClient(providerConfig)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}
