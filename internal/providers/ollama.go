package providers

import (
	"context"
	"encoding/json"
	"fmt"

	ollama "github.com/prathyushnallamothu/ollamago"
)

type OllamaClient[R any] struct {
	client      *ollama.Client
	model       string
	temperature float64
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient[R any](config *Config) (Client[R], error) {
	client := ollama.NewClient(
		ollama.WithBaseURL(config.BaseURL),
	)

	return &OllamaClient[R]{
		client:      client,
		model:       config.Model,
		temperature: config.Temperature,
	}, nil
}

// Name returns the provider name
func (c *OllamaClient[R]) Name() string {
	return string(ProviderOllama)
}

// Validate checks if the client configuration is valid
func (c *OllamaClient[R]) Validate() error {
	if c.client == nil {
		return fmt.Errorf("client is not initialized")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// Complete sends a completion request to Ollama API
func (c *OllamaClient[R]) Complete(ctx context.Context, req *Request) (*R, Usage, error) {
	if err := c.Validate(); err != nil {
		return nil, Usage{}, fmt.Errorf("client validation failed: %w", err)
	}

	// Create the generate request
	generateReq := &ollama.GenerateRequest{
		Model:  c.model,
		Prompt: req.UserPrompt,
		System: req.SystemPrompt,
		Options: &ollama.Options{
			Temperature: &c.temperature,
		},
		Format: "json", // Request JSON format for structured output
		Stream: false,  // We want the complete response
	}

	// Send request
	resp, err := c.client.Generate(ctx, *generateReq)
	if err != nil {
		return nil, Usage{}, fmt.Errorf("ollama API request failed: %w", err)
	}

	// Parse the JSON response directly into type R
	var result R
	if err := json.Unmarshal([]byte(resp.Response), &result); err != nil {
		return nil, Usage{}, fmt.Errorf("failed to parse response as JSON: %w", err)
	}

	// Convert token usage information
	usage := Usage{
		PromptTokens:     resp.PromptEvalCount,
		CompletionTokens: resp.EvalCount,
		TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
	}

	return &result, usage, nil
}
