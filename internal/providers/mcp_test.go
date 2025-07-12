package providers

import (
	"context"
	"testing"
	"time"
)

// TestNewMCPClient tests MCP client creation
func TestNewMCPClient(t *testing.T) {
	client := NewMCPClient("localhost", 8080)
	
	if client == nil {
		t.Error("NewMCPClient returned nil")
	}
	
	if client.address != "localhost" {
		t.Errorf("Expected address 'localhost', got '%s'", client.address)
	}
	
	if client.port != 8080 {
		t.Errorf("Expected port 8080, got %d", client.port)
	}
}

// TestMCPClientValidate tests MCP client validation
func TestMCPClientValidate(t *testing.T) {
	tests := []struct {
		name      string
		address   string
		port      int
		expectErr bool
	}{
		{
			name:      "valid config",
			address:   "localhost",
			port:      8080,
			expectErr: false,
		},
		{
			name:      "empty address",
			address:   "",
			port:      8080,
			expectErr: true,
		},
		{
			name:      "invalid port - zero",
			address:   "localhost",
			port:      0,
			expectErr: true,
		},
		{
			name:      "invalid port - negative",
			address:   "localhost",
			port:      -1,
			expectErr: true,
		},
		{
			name:      "invalid port - too high",
			address:   "localhost",
			port:      65536,
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewMCPClient(tt.address, tt.port)
			err := client.Validate()
			
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestMCPClientName tests MCP client name
func TestMCPClientName(t *testing.T) {
	client := NewMCPClient("localhost", 8080)
	
	if client.Name() != "mcp" {
		t.Errorf("Expected name 'mcp', got '%s'", client.Name())
	}
}

// TestMCPClientNotConnected tests MCP client behavior when not connected
func TestMCPClientNotConnected(t *testing.T) {
	client := NewMCPClient("localhost", 8080)
	
	ctx := context.Background()
	req := &Request{
		SystemPrompt: "test system",
		UserPrompt:   "test prompt",
		MaxTokens:    100,
		Timeout:      30 * time.Second,
	}
	
	_, err := client.Complete(ctx, req)
	if err == nil {
		t.Error("Expected error when not connected to MCP server")
	}
	
	if err.Error() != "not connected to MCP server" {
		t.Errorf("Expected 'not connected to MCP server' error, got '%s'", err.Error())
	}
}