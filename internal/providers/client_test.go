package providers

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockClient implements the Client interface for testing
type mockClient struct {
	name     string
	response *Response
	err      error
	valid    bool
}

func (m *mockClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
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
		client     Client
		request    *Request
		wantError  bool
		wantResult string
	}{
		{
			name: "successful completion",
			client: &mockClient{
				name: "mock",
				response: &Response{
					Usage: Usage{
						PromptTokens:     10,
						CompletionTokens: 5,
						TotalTokens:      15,
					},
					Issues: []SemanticIssue{},
				},
				valid: true,
			},
			request: &Request{
				Prompt:      "test prompt",
				MaxTokens:   100,
				Temperature: 0.1,
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
				Prompt: "test prompt",
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
				Prompt: "test prompt",
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
			resp, err := tt.client.Complete(ctx, tt.request)
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
		})
	}
}

func TestRequest(t *testing.T) {
	req := &Request{
		Prompt:      "test prompt",
		MaxTokens:   500,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
	}

	if req.Prompt != "test prompt" {
		t.Errorf("expected prompt 'test prompt', got %s", req.Prompt)
	}
	if req.MaxTokens != 500 {
		t.Errorf("expected MaxTokens 500, got %d", req.MaxTokens)
	}
	if req.Temperature != 0.7 {
		t.Errorf("expected Temperature 0.7, got %f", req.Temperature)
	}
	if req.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", req.Timeout)
	}
}

func TestResponse(t *testing.T) {
	resp := &Response{
		Usage: Usage{
			PromptTokens:     20,
			CompletionTokens: 30,
			TotalTokens:      50,
		},
		Issues: []SemanticIssue{
			{
				Level:      "ERROR",
				Message:    "test issue",
				Reasoning:  "test reasoning",
				Confidence: 0.9,
				Suggestion: "fix this",
			},
		},
	}

	if resp.Usage.TotalTokens != 50 {
		t.Errorf("expected total tokens 50, got %d", resp.Usage.TotalTokens)
	}
	if len(resp.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(resp.Issues))
	}
	if resp.Issues[0].Level != "ERROR" {
		t.Errorf("expected issue level 'ERROR', got %s", resp.Issues[0].Level)
	}
}
