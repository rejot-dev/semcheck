package mcp

import (
	"context"
	"testing"

	"github.com/rejot-dev/semcheck/internal/config"
)

// TestNewServer tests server creation
func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		MCP: &config.MCPConfig{
			Enabled: true,
			Address: "localhost",
			Port:    8080,
		},
		Rules: []config.Rule{
			{
				Name:        "test-rule",
				Description: "Test rule",
				Enabled:     true,
				FailOn:      "error",
			},
		},
	}

	handler := NewToolsResourcesHandler(cfg, "/tmp")
	server := NewServer("localhost", 8080, handler)

	if server.address != "localhost" {
		t.Errorf("Expected address localhost, got %s", server.address)
	}
	if server.port != 8080 {
		t.Errorf("Expected port 8080, got %d", server.port)
	}
	if server.handler != handler {
		t.Error("Handler not set correctly")
	}
}

// TestServer_StartStop tests server start and stop
func TestServer_StartStop(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		MCP: &config.MCPConfig{
			Enabled: true,
			Address: "localhost",
			Port:    8081, // Use different port for testing
		},
		Rules: []config.Rule{
			{
				Name:        "test-rule",
				Description: "Test rule",
				Enabled:     true,
				FailOn:      "error",
			},
		},
	}

	handler := NewToolsResourcesHandler(cfg, "/tmp")
	server := NewServer("localhost", 8081, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	err := server.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start server: %v", err)
	}

	// Check if server is running
	if !server.IsRunning() {
		t.Error("Server should be running")
	}

	// Stop server
	err = server.Stop()
	if err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Check if server is stopped
	if server.IsRunning() {
		t.Error("Server should be stopped")
	}
}

// TestServer_IsRunning tests server running status
func TestServer_IsRunning(t *testing.T) {
	cfg := &config.Config{
		Version: "1.0",
		MCP: &config.MCPConfig{
			Enabled: true,
			Address: "localhost",
			Port:    8082,
		},
		Rules: []config.Rule{
			{
				Name:        "test-rule",
				Description: "Test rule",
				Enabled:     true,
				FailOn:      "error",
			},
		},
	}

	handler := NewToolsResourcesHandler(cfg, "/tmp")
	server := NewServer("localhost", 8082, handler)

	// Initially should not be running
	if server.IsRunning() {
		t.Error("Server should not be running initially")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	err := server.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start server: %v", err)
	}

	// Should be running now
	if !server.IsRunning() {
		t.Error("Server should be running after start")
	}

	// Stop server
	err = server.Stop()
	if err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Should not be running after stop
	if server.IsRunning() {
		t.Error("Server should not be running after stop")
	}
}
