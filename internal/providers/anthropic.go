package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicClient implements the Client interface for Anthropic API
type AnthropicClient struct {
	client      *anthropic.Client
	model       string
	temperature float64
	maxTokens   int
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(config *Config) (*AnthropicClient, error) {
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

	return &AnthropicClient{
		client:      &client,
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}, nil
}

// Name returns the provider name
func (c *AnthropicClient) Name() string {
	return "anthropic"
}

// Validate checks if the client configuration is valid
func (c *AnthropicClient) Validate() error {
	if c.client == nil {
		return fmt.Errorf("client is not initialized")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// Complete sends a completion request to Anthropic API
func (c *AnthropicClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("client validation failed: %w", err)
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
		return nil, fmt.Errorf("anthropic API request failed: %w", err)
	}

	// Extract text content from response
	var responseText = prefill
	for _, content := range resp.Content {
		if content.Type == "text" {
			responseText += content.Text
		}
	}

	// Parse JSON response into our structured format
	var structuredResp StructuredResponse
	if err := json.Unmarshal([]byte(responseText), &structuredResp); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w, value: %s", err, responseText)
	}

	// Convert to our response format
	response := &Response{
		Usage: Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
		Issues: structuredResp.Issues,
	}

	return response, nil
}
