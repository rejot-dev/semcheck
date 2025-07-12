package mcp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rejot-dev/semcheck/internal/providers"
)

// MockLLMRequestHandler is a mock implementation of LLMRequestHandler for testing
type MockLLMRequestHandler struct {
	responses map[string]*providers.Response
	errors    map[string]error
}

// NewMockLLMRequestHandler creates a new mock LLM request handler
func NewMockLLMRequestHandler() *MockLLMRequestHandler {
	return &MockLLMRequestHandler{
		responses: make(map[string]*providers.Response),
		errors:    make(map[string]error),
	}
}

// AddResponse adds a mock response for a given user prompt
func (m *MockLLMRequestHandler) AddResponse(userPrompt string, response *providers.Response) {
	m.responses[userPrompt] = response
}

// AddError adds a mock error for a given user prompt
func (m *MockLLMRequestHandler) AddError(userPrompt string, err error) {
	m.errors[userPrompt] = err
}

// HandleLLMRequest handles mock LLM requests
func (m *MockLLMRequestHandler) HandleLLMRequest(ctx context.Context, req *providers.Request) (*providers.Response, error) {
	if err, exists := m.errors[req.UserPrompt]; exists {
		return nil, err
	}
	
	if response, exists := m.responses[req.UserPrompt]; exists {
		return response, nil
	}
	
	return nil, fmt.Errorf("no mock response configured for prompt: %s", req.UserPrompt)
}

// TestNewServer tests server creation
func TestNewServer(t *testing.T) {
	handler := NewMockLLMRequestHandler()
	server := NewServer("localhost", 8080, handler)
	
	if server == nil {
		t.Error("NewServer returned nil")
	}
	
	if server.address != "localhost" {
		t.Errorf("Expected address 'localhost', got '%s'", server.address)
	}
	
	if server.port != 8080 {
		t.Errorf("Expected port 8080, got %d", server.port)
	}
	
	if server.handler != handler {
		t.Error("Handler not set correctly")
	}
}

// TestServerStartStop tests server start and stop
func TestServerStartStop(t *testing.T) {
	handler := NewMockLLMRequestHandler()
	server := NewServer("localhost", 0, handler) // Use port 0 to get any available port
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Test start
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	
	if !server.IsRunning() {
		t.Error("Server should be running after start")
	}
	
	// Test stop
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
	
	if server.IsRunning() {
		t.Error("Server should not be running after stop")
	}
}

// TestServerDoubleStart tests that starting an already running server returns error
func TestServerDoubleStart(t *testing.T) {
	handler := NewMockLLMRequestHandler()
	server := NewServer("localhost", 0, handler)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start server
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()
	
	// Try to start again
	err = server.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running server")
	}
}

// TestDirectLLMRequestHandler tests the direct LLM request handler
func TestDirectLLMRequestHandler(t *testing.T) {
	// Create mock provider client
	mockClient := &MockClient{
		responses: make(map[string]*providers.Response),
		errors:    make(map[string]error),
	}
	
	// Add mock response
	mockResponse := &providers.Response{
		Usage: providers.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		Issues: []providers.SemanticIssue{
			{
				Level:      "ERROR",
				Message:    "Test issue",
				Confidence: 0.9,
			},
		},
	}
	mockClient.AddResponse("test prompt", mockResponse)
	
	// Create handler
	handler := NewDirectLLMRequestHandler(mockClient)
	
	// Test request
	ctx := context.Background()
	req := &providers.Request{
		SystemPrompt: "test system",
		UserPrompt:   "test prompt",
		MaxTokens:    100,
		Timeout:      30 * time.Second,
	}
	
	response, err := handler.HandleLLMRequest(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if response == nil {
		t.Error("Expected response, got nil")
	}
	
	if len(response.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(response.Issues))
	}
	
	if response.Issues[0].Message != "Test issue" {
		t.Errorf("Expected message 'Test issue', got '%s'", response.Issues[0].Message)
	}
}

// MockClient is a mock implementation of providers.Client for testing
type MockClient struct {
	responses map[string]*providers.Response
	errors    map[string]error
}

// AddResponse adds a mock response for a given user prompt
func (m *MockClient) AddResponse(userPrompt string, response *providers.Response) {
	m.responses[userPrompt] = response
}

// AddError adds a mock error for a given user prompt
func (m *MockClient) AddError(userPrompt string, err error) {
	m.errors[userPrompt] = err
}

// Complete implements providers.Client interface
func (m *MockClient) Complete(ctx context.Context, req *providers.Request) (*providers.Response, error) {
	if err, exists := m.errors[req.UserPrompt]; exists {
		return nil, err
	}
	
	if response, exists := m.responses[req.UserPrompt]; exists {
		return response, nil
	}
	
	return nil, fmt.Errorf("no mock response configured for prompt: %s", req.UserPrompt)
}

// Name implements providers.Client interface
func (m *MockClient) Name() string {
	return "mock"
}

// Validate implements providers.Client interface
func (m *MockClient) Validate() error {
	return nil
}