package providers

import (
	"context"
	"fmt"
	"testing"

	"github.com/rejot-dev/semcheck/internal/config"
)

// mockClient implements the Client interface for testing
type mockClient struct {
	name     string
	response *IssueResponse
	usage    Usage
	err      error
	valid    bool
}

func (m *mockClient) Complete(ctx context.Context, req *Request) (*IssueResponse, Usage, error) {
	if err := m.Validate(); err != nil {
		return nil, Usage{}, err
	}
	if m.err != nil {
		return nil, Usage{}, m.err
	}
	return m.response, m.usage, nil
}

func (m *mockClient) Name() string {
	return m.name
}

func (m *mockClient) Validate() error {
	if !m.valid {
		return fmt.Errorf("mock client is invalid")
	}
	return nil
}

func TestClientInterface(t *testing.T) {
	tests := []struct {
		name       string
		client     Client[IssueResponse]
		request    *Request
		wantError  bool
		wantResult string
	}{
		{
			name: "successful completion",
			client: &mockClient{
				name: "mock",
				response: &IssueResponse{
					Issues: []SemanticIssue{},
				},
				usage: Usage{
					InputTokens:  10,
					OutputTokens: 5,
					TotalTokens:  15,
				},
				valid: true,
			},
			request: &Request{
				SystemPrompt: "You are a helpful assistant",
				UserPrompt:   "test prompt",
			},
			wantError:  false,
			wantResult: "test response",
		},
		{
			name: "client error",
			client: &mockClient{
				name:  "mock",
				err:   fmt.Errorf("API error"),
				valid: true,
			},
			request: &Request{
				UserPrompt: "test prompt",
			},
			wantError: true,
		},
		{
			name: "invalid client",
			client: &mockClient{
				name:  "mock",
				valid: false,
			},
			request: &Request{
				UserPrompt: "test prompt",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Test name
			if got := tt.client.Name(); got != tt.client.(*mockClient).name {
				t.Errorf("Name() = %v, want %v", got, tt.client.(*mockClient).name)
			}

			// Test completion (which includes validation)
			resp, usage, err := tt.client.Complete(ctx, tt.request)
			if tt.wantError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.wantError && resp != nil && len(resp.Issues) != 0 {
				// For successful completion, we expect empty issues array
				t.Errorf("Complete() returned unexpected issues: %v", resp.Issues)
			}
			if !tt.wantError && usage.TotalTokens != 15 {
				t.Errorf("Expected usage.TotalTokens = 15, got %d", usage.TotalTokens)
			}
		})
	}
}

func TestRequest(t *testing.T) {
	req := &Request{
		UserPrompt: "test prompt",
	}

	if req.UserPrompt != "test prompt" {
		t.Errorf("expected prompt 'test prompt', got %s", req.UserPrompt)
	}
}

func TestIssueResponse(t *testing.T) {
	resp := &IssueResponse{
		Issues: []SemanticIssue{
			{
				Level:      "ERROR",
				Message:    "test issue",
				Reasoning:  "test reasoning",
				Suggestion: "fix this",
			},
		},
	}

	usage := Usage{
		InputTokens:  20,
		OutputTokens: 30,
		TotalTokens:  50,
	}

	if usage.TotalTokens != 50 {
		t.Errorf("expected total tokens 50, got %d", usage.TotalTokens)
	}
	if len(resp.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(resp.Issues))
	}
	if resp.Issues[0].Level != "ERROR" {
		t.Errorf("expected issue level 'ERROR', got %s", resp.Issues[0].Level)
	}
}

func TestAnthropicClient(t *testing.T) {
	// Test that AnthropicClient implements the Client interface
	config := &Config{
		Provider: "anthropic",
		Model:    "claude-3-sonnet-20240229",
		APIKey:   "test-key",
	}

	client, err := NewAnthropicClient[IssueResponse](config)
	if err != nil {
		t.Fatalf("Failed to create Anthropic client: %v", err)
	}

	// Test client methods
	if client.Name() != "anthropic" {
		t.Errorf("Expected name 'anthropic', got %s", client.Name())
	}

	// Test validation
	if err := client.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test invalid config
	invalidClient := &AnthropicClient[IssueResponse]{model: ""}
	if err := invalidClient.Validate(); err == nil {
		t.Error("Expected validation to fail for empty model")
	}
}

func TestCreateAIClientAnthropic(t *testing.T) {
	// Test CreateAIClient with Anthropic provider
	temperature := 0.1
	cfg := &config.Config{
		Provider:    "anthropic",
		Model:       "claude-3-sonnet-20240229",
		APIKey:      "test-key",
		Temperature: &temperature,
	}

	client, err := CreateAIClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create Anthropic client via factory: %v", err)
	}

	if client.Name() != "anthropic" {
		t.Errorf("Expected client name 'anthropic', got %s", client.Name())
	}

	// Test invalid provider
	invalidTemp := 0.1
	invalidCfg := &config.Config{
		Provider:    "unsupported",
		Model:       "test",
		APIKey:      "test-key",
		Temperature: &invalidTemp,
	}

	_, err = CreateAIClient(invalidCfg)
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}
}

func TestGeminiClient(t *testing.T) {
	// Test that GeminiClient implements the Client interface
	config := &Config{
		Provider: ProviderGemini,
		Model:    "gemini-2.5-pro",
		APIKey:   "test-key",
	}

	client, err := NewGeminiClient[IssueResponse](config)
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}

	// Test client methods
	if client.Name() != "gemini" {
		t.Errorf("Expected name 'gemini', got %s", client.Name())
	}

	// Test validation
	if err := client.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test invalid config
	invalidClient := &GeminiClient[IssueResponse]{model: ""}
	if err := invalidClient.Validate(); err == nil {
		t.Error("Expected validation to fail for empty model")
	}
}
