package providers

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient implements the Client interface for OpenAI API
type OpenAIClient struct {
	client *openai.Client
	model  string
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config *Config) (*OpenAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for OpenAI provider")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(config.APIKey),
	}

	if config.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.BaseURL))
	}

	client := openai.NewClient(opts...)

	return &OpenAIClient{
		client: &client,
		model:  config.Model,
	}, nil
}

// Name returns the provider name
func (c *OpenAIClient) Name() string {
	return "openai"
}

// Validate checks if the client configuration is valid
func (c *OpenAIClient) Validate() error {
	if c.client == nil {
		return fmt.Errorf("client is not initialized")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// Complete sends a completion request to OpenAI API
func (c *OpenAIClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("client validation failed: %w", err)
	}

	// Set defaults
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1000
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.1
	}

	// Create chat completion request
	chatReq := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(req.Prompt),
		},
		Model:       openai.ChatModel(c.model),
		MaxTokens:   openai.Int(int64(maxTokens)),
		Temperature: openai.Float(temperature),
	}

	// Apply timeout if specified
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	// Send request
	resp, err := c.client.Chat.Completions.New(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Convert to our response format
	response := &Response{
		Content: resp.Choices[0].Message.Content,
		Usage: Usage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
		Confidence: 1.0, // OpenAI doesn't provide confidence scores
	}

	return response, nil
}
