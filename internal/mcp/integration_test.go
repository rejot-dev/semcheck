package mcp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/rejot-dev/semcheck/internal/providers"
)

// TestMCPServerClientIntegration tests MCP server and client integration
func TestMCPServerClientIntegration(t *testing.T) {
	// Create mock handler
	handler := NewMockLLMRequestHandler()
	
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
				Message:    "Integration test issue",
				Confidence: 0.9,
			},
		},
	}
	handler.AddResponse("integration test", mockResponse)
	
	// Create and start server
	server := NewServer("localhost", 0, handler) // Use port 0 to get any available port
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()
	
	// Get the actual port the server is listening on
	serverPort := server.ln.Addr().(*net.TCPAddr).Port
	
	// Create and connect client
	client := providers.NewMCPClient("localhost", serverPort)
	err = client.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to MCP server: %v", err)
	}
	defer client.Disconnect()
	
	// Test request
	req := &providers.Request{
		SystemPrompt: "integration test system",
		UserPrompt:   "integration test",
		MaxTokens:    100,
		Timeout:      30 * time.Second,
	}
	
	response, err := client.Complete(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if response == nil {
		t.Error("Expected response, got nil")
	}
	
	if len(response.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(response.Issues))
	}
	
	if response.Issues[0].Message != "Integration test issue" {
		t.Errorf("Expected message 'Integration test issue', got '%s'", response.Issues[0].Message)
	}
}