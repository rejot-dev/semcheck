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
	ProviderGemini    Provider = "gemini"
)

func ToProvider(provider string) (Provider, error) {
	switch provider {
	case "openai":
		return ProviderOpenAI, nil
	case "anthropic":
		return ProviderAnthropic, nil
	case "gemini":
		return ProviderGemini, nil
	default:
		return "", fmt.Errorf("invalid provider: %s", provider)
	}
}

func GetAllProviders() []Provider {
	return []Provider{ProviderOpenAI, ProviderAnthropic, ProviderGemini}
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
	case ProviderGemini:
		return ProviderDefaults{
			Model:     "gemini-2.5-flash",
			ApiKeyVar: "GOOGLE_API_KEY",
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
	Reasoning  string  `json:"reasoning" jsonschema_description:"Reasoning why the found issue has it's severity level"`
	Level      string  `json:"level" jsonschema_description:"Severity level of the issue"`
	Message    string  `json:"message" jsonschema_description:"Description of the issue"`
	Confidence float64 `json:"confidence" jsonschema_description:"Confidence level of the issue (0.0-1.0)"`
	Suggestion string  `json:"suggestion" jsonschema_description:"Suggestion for fixing the issue (optional)"`
}

// Config holds common configuration for AI providers
type Config struct {
	Provider    Provider
	Model       string
	APIKey      string
	BaseURL     string
	Timeout     time.Duration
	Temperature float64
}

func CreateAIClient(cfg *config.Config) (Client, error) {
	// Convert config to provider config

	provider, providerErr := ToProvider(cfg.Provider)
	if providerErr != nil {
		return nil, fmt.Errorf("invalid provider: %s", cfg.Provider)
	}

	providerConfig := &Config{
		Provider:    provider,
		Model:       cfg.Model,
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Timeout:     time.Duration(cfg.Timeout) * time.Second,
		Temperature: *cfg.Temperature,
	}

	var client Client
	var err error

	switch provider {
	case ProviderOpenAI:
		client, err = NewOpenAIClient(providerConfig)
	case ProviderAnthropic:
		client, err = NewAnthropicClient(providerConfig)
	case ProviderGemini:
		client, err = NewGeminiClient(providerConfig)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}
