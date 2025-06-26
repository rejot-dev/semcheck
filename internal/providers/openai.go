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
type OpenAIClient struct {
	client      *openai.Client
	model       string
	temperature float64
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
		client:      &client,
		model:       config.Model,
		temperature: config.Temperature,
	}, nil
}

// Name returns the provider name
func (c *OpenAIClient) Name() string {
	return string(ProviderOpenAI)
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

func generateSchema[T any]() interface{} {
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

type StructuredResponse struct {
	Issues []SemanticIssue `json:"issues" jsonschema_description:"List of issues found"`
}

var StructuredResponseSchema = generateSchema[StructuredResponse]()

// Complete sends a completion request to OpenAI API
func (c *OpenAIClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("client validation failed: %w", err)
	}

	// Set defaults
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 3000
	}

	temperature := c.temperature

	// Generate schema for structured output
	schema := generateSchema[StructuredResponse]()

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
		MaxCompletionTokens: openai.Int(int64(req.MaxTokens)),
		Temperature:         openai.Float(temperature),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
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

	// Parse structured JSON response
	var structuredResp StructuredResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &structuredResp); err != nil {
		return nil, fmt.Errorf("failed to parse structured response: %w", err)
	}

	// Convert to our response format
	response := &Response{
		Usage: Usage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
		Issues: structuredResp.Issues,
	}

	return response, nil
}
