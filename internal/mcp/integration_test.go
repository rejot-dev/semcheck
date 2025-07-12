package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/rejot-dev/semcheck/internal/config"
)

// TestMCPServerToolsResourcesIntegration tests MCP server Tools/Resources integration
func TestMCPServerToolsResourcesIntegration(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Version: "1.0",
		MCP: &config.MCPConfig{
			Enabled: true,
			Address: "localhost",
			Port:    0, // Use port 0 to get any available port
		},
		Rules: []config.Rule{
			{
				Name:        "integration-test-rule",
				Description: "Integration test rule",
				Enabled:     true,
				FailOn:      "error",
			},
		},
	}

	// Create handler and server
	handler := NewToolsResourcesHandler(cfg, "/tmp")
	server := NewServer("localhost", 0, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Get the actual port the server is listening on
	serverPort := server.ln.Addr().(*net.TCPAddr).Port

	// Test tools/list
	t.Run("ToolsList", func(t *testing.T) {
		conn, err := net.Dial("tcp", "localhost:"+fmt.Sprintf("%d", serverPort))
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer conn.Close()

		encoder := json.NewEncoder(conn)
		decoder := json.NewDecoder(conn)

		request := MCPRequest{
			ID:     "test-tools-list",
			Method: "tools/list",
			Params: map[string]interface{}{},
		}

		err = encoder.Encode(request)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		var response MCPResponse
		err = decoder.Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Error != nil {
			t.Fatalf("Unexpected error in response: %v", response.Error)
		}

		if response.Result == nil {
			t.Fatal("Expected result in response")
		}
	})

	// Test resources/list
	t.Run("ResourcesList", func(t *testing.T) {
		conn, err := net.Dial("tcp", "localhost:"+fmt.Sprintf("%d", serverPort))
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer conn.Close()

		encoder := json.NewEncoder(conn)
		decoder := json.NewDecoder(conn)

		request := MCPRequest{
			ID:     "test-resources-list",
			Method: "resources/list",
			Params: map[string]interface{}{},
		}

		err = encoder.Encode(request)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		var response MCPResponse
		err = decoder.Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Error != nil {
			t.Fatalf("Unexpected error in response: %v", response.Error)
		}

		if response.Result == nil {
			t.Fatal("Expected result in response")
		}
	})

	// Test tools/call
	t.Run("ToolsCall", func(t *testing.T) {
		conn, err := net.Dial("tcp", "localhost:"+fmt.Sprintf("%d", serverPort))
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer conn.Close()

		encoder := json.NewEncoder(conn)
		decoder := json.NewDecoder(conn)

		request := MCPRequest{
			ID:     "test-tools-call",
			Method: "tools/call",
			Params: map[string]interface{}{
				"name":      "list_rules",
				"arguments": map[string]interface{}{},
			},
		}

		err = encoder.Encode(request)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		var response MCPResponse
		err = decoder.Decode(&response)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Error != nil {
			t.Fatalf("Unexpected error in response: %v", response.Error)
		}

		if response.Result == nil {
			t.Fatal("Expected result in response")
		}
	})
}