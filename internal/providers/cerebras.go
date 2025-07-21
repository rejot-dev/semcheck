package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CerebrasClient implements the Client interface for Cerebras API
type CerebrasClient struct {
	httpClient  *http.Client
	apiKey      string
	model       string
	baseURL     string
	temperature float64
	maxTokens   int
}

// NewCerebrasClient creates a new Cerebras client
func NewCerebrasClient(config *Config) (*CerebrasClient, error) {
	baseURL := "https://api.cerebras.ai/v1"
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}

	return &CerebrasClient{
		httpClient:  &http.Client{},
		apiKey:      config.APIKey,
		model:       config.Model,
		baseURL:     baseURL,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}, nil
}

// Name returns the provider name
func (c *CerebrasClient) Name() string {
	return string(ProviderCerebras)
}

// Validate checks if the client configuration is valid
func (c *CerebrasClient) Validate() error {
	if c.httpClient == nil {
		return fmt.Errorf("HTTP client is not initialized")
	}
	if c.apiKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// CerebrasMessage represents a message in the chat completion request
type CerebrasMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CerebrasJsonSchema struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema any    `json:"schema"`
}

type CerebrasResponseFormat struct {
	Type       string             `json:"type"`
	JsonSchema CerebrasJsonSchema `json:"json_schema"`
}

// CerebrasRequest represents the request payload for Cerebras API
type CerebrasRequest struct {
	Model          string                  `json:"model"`
	Messages       []CerebrasMessage       `json:"messages"`
	Temperature    *float64                `json:"temperature,omitempty"`
	MaxTokens      *int                    `json:"max_completion_tokens,omitempty"`
	ResponseFormat *CerebrasResponseFormat `json:"response_format,omitempty"`
}

// CerebrasChoice represents a choice in the API response
type CerebrasChoice struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

// CerebrasUsage represents token usage in the API response
type CerebrasUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CerebrasResponse represents the response from Cerebras API
type CerebrasResponse struct {
	ID      string           `json:"id"`
	Choices []CerebrasChoice `json:"choices"`
	Usage   CerebrasUsage    `json:"usage"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
}

// Complete sends a completion request to Cerebras API
func (c *CerebrasClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("client validation failed: %w", err)
	}

	// Prepare messages for the API request
	messages := []CerebrasMessage{
		{Role: "system", Content: req.SystemPrompt},
		{Role: "user", Content: req.UserPrompt},
	}

	// Generate schema for structured output
	schema := generateSchema[StructuredResponse]()

	// Create request payload
	payload := CerebrasRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: &c.temperature,
		MaxTokens:   &c.maxTokens,
		ResponseFormat: &CerebrasResponseFormat{
			Type: "json_schema",
			JsonSchema: CerebrasJsonSchema{
				Name:   "semantic_analysis",
				Strict: true,
				Schema: schema,
			},
		},
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("cerebras API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cerebras API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var cerebrasResp CerebrasResponse
	if err := json.Unmarshal(body, &cerebrasResp); err != nil {
		return nil, fmt.Errorf("failed to parse Cerebras response: %w", err)
	}

	if len(cerebrasResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	// Parse the response content as JSON to extract semantic issues
	var structuredResp StructuredResponse
	if err := json.Unmarshal([]byte(cerebrasResp.Choices[0].Message.Content), &structuredResp); err != nil {
		return nil, fmt.Errorf("failed to parse structured response: %w", err)
	}

	// Convert to our response format
	response := &Response{
		Usage: Usage{
			PromptTokens:     cerebrasResp.Usage.PromptTokens,
			CompletionTokens: cerebrasResp.Usage.CompletionTokens,
			TotalTokens:      cerebrasResp.Usage.TotalTokens,
		},
		Issues: structuredResp.Issues,
	}

	return response, nil
}
