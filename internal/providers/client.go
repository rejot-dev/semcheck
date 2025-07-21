package providers

import (
	"context"
	"fmt"

	"github.com/rejot-dev/semcheck/internal/config"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
	ProviderOllama    Provider = "ollama"
	ProviderCerebras  Provider = "cerebras"
)

func ToProvider(provider string) (Provider, error) {
	switch provider {
	case "openai":
		return ProviderOpenAI, nil
	case "anthropic":
		return ProviderAnthropic, nil
	case "gemini":
		return ProviderGemini, nil
	case "ollama":
		return ProviderOllama, nil
	case "cerebras":
		return ProviderCerebras, nil
	default:
		return "", fmt.Errorf("invalid provider: %s", provider)
	}
}

func GetAllProviders() []Provider {
	return []Provider{ProviderOpenAI, ProviderAnthropic, ProviderGemini, ProviderOllama, ProviderCerebras}
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
	case ProviderOllama:
		return ProviderDefaults{
			Model:     "llama3.2",
			ApiKeyVar: "", // Ollama doesn't require an API key
		}
	case ProviderCerebras:
		return ProviderDefaults{
			Model:     "llama-4-scout-17b-16e-instruct",
			ApiKeyVar: "CEREBRAS_API_KEY",
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
	File       string  `json:"file" jsonschema_description:"The file that the issue is in"`
}

// Config holds common configuration for AI providers
type Config struct {
	Provider    Provider
	Model       string
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
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
		MaxTokens:   cfg.MaxTokens,
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
	case ProviderOllama:
		client, err = NewOllamaClient(providerConfig)
	case ProviderCerebras:
		client, err = NewCerebrasClient(providerConfig)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}
