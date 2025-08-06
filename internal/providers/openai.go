package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIClient implements the Client interface for OpenAI API
type OpenAIClient[R any] struct {
	client      *openai.Client
	model       string
	temperature float64
	maxTokens   int
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient[R any](config *Config) (Client[R], error) {
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

	return &OpenAIClient[R]{
		client:      &client,
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}, nil
}

// Name returns the provider name
func (c *OpenAIClient[R]) Name() string {
	return string(ProviderOpenAI)
}

// Validate checks if the client configuration is valid
func (c *OpenAIClient[R]) Validate() error {
	if c.client == nil {
		return fmt.Errorf("client is not initialized")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

func generateSchema[T any]() any {
	// Structured Outputs uses a subset of JSON schema
	// These flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// Complete sends a completion request to OpenAI API
func (c *OpenAIClient[R]) Complete(ctx context.Context, req *Request) (*R, Usage, error) {
	if err := c.Validate(); err != nil {
		return nil, Usage{}, fmt.Errorf("client validation failed: %w", err)
	}

	temperature := c.temperature

	// Generate schema for structured output
	schema := generateSchema[R]()

	// Create structured output response format following the example pattern
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "semantic_analysis",
		Description: openai.String("Semantic analysis results"),
		Schema:      schema,
		Strict:      openai.Bool(false),
	}

	// Create chat completion request with structured output
	chatReq := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(req.UserPrompt),
			openai.SystemMessage(req.SystemPrompt),
		},
		Model:               openai.ChatModel(c.model),
		MaxCompletionTokens: openai.Int(int64(c.maxTokens)),
		Temperature:         openai.Float(temperature),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
	}

	// Send request
	resp, err := c.client.Chat.Completions.New(ctx, chatReq)
	if err != nil {
		return nil, Usage{}, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, Usage{}, fmt.Errorf("no choices in response")
	}

	// Parse structured JSON response directly into type R
	var result R
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, Usage{}, fmt.Errorf("failed to parse structured response: %w", err)
	}

	// Create usage information
	usage := Usage{
		InputTokens:  int(resp.Usage.PromptTokens),
		OutputTokens: int(resp.Usage.CompletionTokens),
		TotalTokens:  int(resp.Usage.TotalTokens),
	}

	return &result, usage, nil
}
