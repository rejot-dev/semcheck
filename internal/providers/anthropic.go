package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicClient implements the Client interface for Anthropic API
type AnthropicClient[R any] struct {
	client      *anthropic.Client
	model       string
	temperature float64
	maxTokens   int
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient[R any](config *Config) (Client[R], error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Anthropic provider")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}

	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	client := anthropic.NewClient(opts...)

	return &AnthropicClient[R]{
		client:      &client,
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}, nil
}

// Name returns the provider name
func (c *AnthropicClient[R]) Name() string {
	return "anthropic"
}

// Validate checks if the client configuration is valid
func (c *AnthropicClient[R]) Validate() error {
	if c.client == nil {
		return fmt.Errorf("client is not initialized")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// Complete sends a completion request to Anthropic API
func (c *AnthropicClient[R]) Complete(ctx context.Context, req *Request) (*R, Usage, error) {
	if err := c.Validate(); err != nil {
		return nil, Usage{}, fmt.Errorf("client validation failed: %w", err)
	}

	temperature := c.temperature

	// Prefill response to enforce correct JSON output, to prevent markdown formatting of JSON response.
	// https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/prefill-claudes-response#example-structured-data-extraction-with-prefilling
	prefill := "{ \"issues\": ["

	// Send request
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Temperature: anthropic.Float(temperature),
		Model:       anthropic.Model(c.model),
		System: []anthropic.TextBlockParam{{
			Text: req.SystemPrompt,
		},
		},
		Messages: []anthropic.MessageParam{{
			Content: []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: req.UserPrompt},
			}},
			Role: anthropic.MessageParamRoleUser,
		},
			{
				Content: []anthropic.ContentBlockParamUnion{{
					OfText: &anthropic.TextBlockParam{Text: prefill},
				}},
				Role: anthropic.MessageParamRoleAssistant,
			}},
		MaxTokens: int64(c.maxTokens),
	})
	if err != nil {
		return nil, Usage{}, fmt.Errorf("anthropic API request failed: %w", err)
	}

	// Extract text content from response
	var responseText = prefill
	for _, content := range resp.Content {
		if content.Type == "text" {
			responseText += content.Text
		}
	}

	// Parse JSON response directly into type R
	var result R
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, Usage{}, fmt.Errorf("failed to parse AI response: %w, value: %s", err, responseText)
	}

	// Create usage information
	usage := Usage{
		InputTokens:  int(resp.Usage.InputTokens),
		OutputTokens: int(resp.Usage.OutputTokens),
		TotalTokens:  int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
	}

	return &result, usage, nil
}
