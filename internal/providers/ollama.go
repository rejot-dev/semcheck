package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	ollama "github.com/prathyushnallamothu/ollamago"
)

type OllamaClient struct {
	client      *ollama.Client
	model       string
	temperature float64
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(config *Config) (*OllamaClient, error) {
	client := ollama.NewClient(
		ollama.WithBaseURL(config.BaseURL),
		ollama.WithTimeout(config.Timeout*time.Second),
	)

	return &OllamaClient{
		client:      client,
		model:       config.Model,
		temperature: config.Temperature,
	}, nil
}

// Name returns the provider name
func (c *OllamaClient) Name() string {
	return string(ProviderOllama)
}

// Validate checks if the client configuration is valid
func (c *OllamaClient) Validate() error {
	if c.client == nil {
		return fmt.Errorf("client is not initialized")
	}
	if c.model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// Complete sends a completion request to Ollama API
func (c *OllamaClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("client validation failed: %w", err)
	}

	// Apply timeout if specified
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
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
		return nil, fmt.Errorf("ollama API request failed: %w", err)
	}

	// Parse the JSON response to extract semantic issues
	var structuredResp StructuredResponse
	if err := json.Unmarshal([]byte(resp.Response), &structuredResp); err != nil {
		// If JSON parsing fails, try to extract issues from plain text
		// This is a fallback for models that don't follow the exact JSON format
		issues, parseErr := parseTextResponse(resp.Response)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse response as JSON: %w, and failed to parse as text: %w", err, parseErr)
		}
		structuredResp.Issues = issues
	}

	// Convert token usage information
	usage := Usage{
		PromptTokens:     resp.PromptEvalCount,
		CompletionTokens: resp.EvalCount,
		TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
	}

	// Convert to our response format
	response := &Response{
		Usage:  usage,
		Issues: structuredResp.Issues,
	}

	return response, nil
}

// parseTextResponse attempts to parse a plain text response and extract semantic issues
// This is a fallback when the model doesn't return proper JSON
func parseTextResponse(text string) ([]SemanticIssue, error) {
	// This is a simple parser - in a real implementation you might want more sophisticated parsing
	// For now, we'll return an empty list if we can't parse the response
	// The assumption is that most modern models will be able to return JSON when requested

	// Try to find JSON-like content in the response
	lines := strings.Split(text, "\n")
	var issues []SemanticIssue

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for lines that might contain issue information
		if strings.Contains(strings.ToLower(line), "error") ||
			strings.Contains(strings.ToLower(line), "warning") ||
			strings.Contains(strings.ToLower(line), "issue") {

			// Create a basic issue from the text
			issue := SemanticIssue{
				Reasoning:  "Parsed from text response",
				Level:      determineLevel(line),
				Message:    line,
				Confidence: 0.5, // Lower confidence for text parsing
				Suggestion: "",
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}

// determineLevel tries to determine the severity level from text
func determineLevel(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "error") || strings.Contains(lower, "critical") {
		return "error"
	}
	if strings.Contains(lower, "warning") || strings.Contains(lower, "warn") {
		return "warning"
	}
	return "notice"
}
