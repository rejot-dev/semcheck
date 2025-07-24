package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"google.golang.org/genai"
)

// GeminiClient implements the Client interface for Google Gemini API
type GeminiClient[R any] struct {
	client      *genai.Client
	model       string
	temperature float64
	maxTokens   int
}

func generateSchemaForGemini[T any]() any {
	// Generate JSON schema using the same approach as OpenAI
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient[R any](config *Config) (Client[R], error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Gemini provider")
	}

	// TODO: why is there another context is created here?
	ctx := context.Background()

	// Create client with Gemini API backend
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  config.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiClient[R]{
		client:      client,
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}, nil
}

// Name returns the provider name
func (c *GeminiClient[R]) Name() string {
	return string(ProviderGemini)
}

// Validate checks if the client configuration is valid
func (c *GeminiClient[R]) Validate() error {
	if c.client == nil {
		return fmt.Errorf("client is not initialized")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// Complete sends a completion request to Gemini API
func (c *GeminiClient[R]) Complete(ctx context.Context, req *Request) (*R, Usage, error) {
	if err := c.Validate(); err != nil {
		return nil, Usage{}, fmt.Errorf("client validation failed: %w", err)
	}

	// Generate schema for structured output using the same approach as OpenAI
	schema := generateSchemaForGemini[R]()
	temperature := float32(c.temperature)

	// Create generation config with structured output
	genConfig := &genai.GenerateContentConfig{
		MaxOutputTokens:    int32(c.maxTokens),
		Temperature:        &temperature,
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: schema,
		SystemInstruction:  &genai.Content{Parts: []*genai.Part{{Text: req.SystemPrompt}}},
	}

	// Send request to Gemini using the simpler genai.Text helper
	geminiResult, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(req.UserPrompt), genConfig)
	if err != nil {
		return nil, Usage{}, fmt.Errorf("gemini API request failed: %w", err)
	}

	// Extract structured response using result.Text() which handles the JSON parsing
	responseText := geminiResult.Text()
	if responseText == "" {
		return nil, Usage{}, fmt.Errorf("empty response from Gemini")
	}

	// Parse JSON response directly into type R
	var parsedResult R
	if err := json.Unmarshal([]byte(responseText), &parsedResult); err != nil {
		return nil, Usage{}, fmt.Errorf("failed to parse AI response: %w, value: %s", err, responseText)
	}

	// Create usage information
	usage := Usage{
		// Gemini usage information may not be as detailed as other providers
		// We'll set basic values based on what's available
		PromptTokens:     0, // Not always available in Gemini response
		CompletionTokens: 0, // Not always available in Gemini response
		TotalTokens:      0, // Not always available in Gemini response
	}

	// If usage metadata is available, use it
	if geminiResult.UsageMetadata != nil {
		usage = Usage{
			PromptTokens:     int(geminiResult.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(geminiResult.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(geminiResult.UsageMetadata.TotalTokenCount),
		}
	}

	return &parsedResult, usage, nil
}
