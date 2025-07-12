package mcp

import (
	"context"
	"testing"

	"github.com/rejot-dev/semcheck/internal/config"
)

func TestToolsResourcesHandler(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{
		Version:  "1.0",
		Provider: "ollama",
		Model:    "llama3.2",
		Timeout:  30,
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
				Files: config.FilePattern{
					Include: []string{"*.go"},
					Exclude: []string{"*_test.go"},
				},
				Specs: []config.Spec{
					{Path: "specs/test.md"},
				},
				FailOn: "error",
			},
		},
	}

	handler := NewToolsResourcesHandler(cfg, "/tmp")

	t.Run("ListTools", func(t *testing.T) {
		tools := handler.ListTools()
		if len(tools) == 0 {
			t.Error("Expected at least one tool")
		}

		// Check if required tools are present
		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name] = true
		}

		expectedTools := []string{"analyze_code", "list_rules", "get_rule_details", "match_files"}
		for _, expected := range expectedTools {
			if !toolNames[expected] {
				t.Errorf("Expected tool %s not found", expected)
			}
		}
	})

	t.Run("ListResources", func(t *testing.T) {
		resources := handler.ListResources()
		if len(resources) == 0 {
			t.Error("Expected at least one resource")
		}

		// Check if config resource is present
		found := false
		for _, resource := range resources {
			if resource.URI == "config://semcheck.yaml" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected config resource not found")
		}
	})

	t.Run("CallTool_ListRules", func(t *testing.T) {
		req := &ToolCallRequest{
			Name:      "list_rules",
			Arguments: map[string]interface{}{},
		}

		resp, err := handler.CallTool(context.Background(), req)
		if err != nil {
			t.Errorf("Error calling list_rules tool: %v", err)
		}

		if resp.IsError {
			t.Error("Expected successful response but got error")
		}

		if len(resp.Content) == 0 {
			t.Error("Expected content in response")
		}
	})

	t.Run("CallTool_GetRuleDetails", func(t *testing.T) {
		req := &ToolCallRequest{
			Name: "get_rule_details",
			Arguments: map[string]interface{}{
				"rule_name": "test-rule",
			},
		}

		resp, err := handler.CallTool(context.Background(), req)
		if err != nil {
			t.Errorf("Error calling get_rule_details tool: %v", err)
		}

		if resp.IsError {
			t.Error("Expected successful response but got error")
		}

		if len(resp.Content) == 0 {
			t.Error("Expected content in response")
		}
	})

	t.Run("CallTool_UnknownTool", func(t *testing.T) {
		req := &ToolCallRequest{
			Name:      "unknown_tool",
			Arguments: map[string]interface{}{},
		}

		resp, err := handler.CallTool(context.Background(), req)
		if err != nil {
			t.Errorf("Error calling unknown tool: %v", err)
		}

		if !resp.IsError {
			t.Error("Expected error response for unknown tool")
		}
	})

	t.Run("ReadResource_Config", func(t *testing.T) {
		req := &ResourceReadRequest{
			URI: "config://semcheck.yaml",
		}

		resp, err := handler.ReadResource(context.Background(), req)
		if err != nil {
			t.Errorf("Error reading config resource: %v", err)
		}

		if len(resp.Contents) == 0 {
			t.Error("Expected content in response")
		}
	})

	t.Run("ReadResource_UnknownResource", func(t *testing.T) {
		req := &ResourceReadRequest{
			URI: "unknown://resource",
		}

		_, err := handler.ReadResource(context.Background(), req)
		if err == nil {
			t.Error("Expected error for unknown resource")
		}
	})
}