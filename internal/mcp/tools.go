package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rejot-dev/semcheck/internal/config"
	"github.com/rejot-dev/semcheck/internal/processor"
)

// Tool represents an MCP tool that can be called by external clients
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// Resource represents an MCP resource that can be accessed by external clients
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ToolCallRequest represents a tool call request from an external client
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResponse represents a tool call response
type ToolCallResponse struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content returned by a tool
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ResourceReadRequest represents a resource read request
type ResourceReadRequest struct {
	URI string `json:"uri"`
}

// ResourceReadResponse represents a resource read response
type ResourceReadResponse struct {
	Contents []Content `json:"contents"`
}

// ToolsResourcesHandler handles MCP tools and resources
type ToolsResourcesHandler struct {
	config     *config.Config
	workingDir string
}

// NewToolsResourcesHandler creates a new tools/resources handler
func NewToolsResourcesHandler(cfg *config.Config, workingDir string) *ToolsResourcesHandler {
	return &ToolsResourcesHandler{
		config:     cfg,
		workingDir: workingDir,
	}
}

// ListTools returns the list of available tools
func (h *ToolsResourcesHandler) ListTools() []Tool {
	return []Tool{
		{
			Name:        "analyze_code",
			Description: "Analyze code files against specifications for semantic issues",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"files": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of file paths to analyze",
					},
					"specs": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of specification file paths to check against",
					},
					"rule_name": map[string]interface{}{
						"type":        "string",
						"description": "Optional specific rule name to apply",
					},
				},
				"required": []string{"files"},
			},
		},
		{
			Name:        "list_rules",
			Description: "List all available semantic checking rules",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_rule_details",
			Description: "Get detailed information about a specific rule",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"rule_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the rule to get details for",
					},
				},
				"required": []string{"rule_name"},
			},
		},
		{
			Name:        "match_files",
			Description: "Match files against configured rules and return matching pairs",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"files": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of file paths to match",
					},
				},
				"required": []string{"files"},
			},
		},
	}
}

// ListResources returns the list of available resources
func (h *ToolsResourcesHandler) ListResources() []Resource {
	resources := []Resource{
		{
			URI:         "config://semcheck.yaml",
			Name:        "Semcheck Configuration",
			Description: "Current semcheck configuration",
			MimeType:    "application/yaml",
		},
	}

	// Add specification files as resources
	for _, rule := range h.config.Rules {
		for _, spec := range rule.Specs {
			resources = append(resources, Resource{
				URI:         fmt.Sprintf("spec://%s", spec.Path),
				Name:        fmt.Sprintf("Specification: %s", filepath.Base(spec.Path)),
				Description: fmt.Sprintf("Specification file for rule %s", rule.Name),
				MimeType:    "text/markdown",
			})
		}
	}

	return resources
}

// CallTool executes a tool call
func (h *ToolsResourcesHandler) CallTool(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
	switch req.Name {
	case "analyze_code":
		return h.analyzeCode(ctx, req.Arguments)
	case "list_rules":
		return h.listRules(ctx, req.Arguments)
	case "get_rule_details":
		return h.getRuleDetails(ctx, req.Arguments)
	case "match_files":
		return h.matchFiles(ctx, req.Arguments)
	default:
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Unknown tool: %s", req.Name),
			}},
			IsError: true,
		}, nil
	}
}

// ReadResource reads a resource
func (h *ToolsResourcesHandler) ReadResource(ctx context.Context, req *ResourceReadRequest) (*ResourceReadResponse, error) {
	if strings.HasPrefix(req.URI, "config://") {
		return h.readConfig(ctx, req.URI)
	} else if strings.HasPrefix(req.URI, "spec://") {
		return h.readSpec(ctx, req.URI)
	} else {
		return &ResourceReadResponse{
			Contents: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Unknown resource URI: %s", req.URI),
			}},
		}, fmt.Errorf("unknown resource URI: %s", req.URI)
	}
}

// analyzeCode performs semantic analysis on code files
func (h *ToolsResourcesHandler) analyzeCode(ctx context.Context, args map[string]interface{}) (*ToolCallResponse, error) {
	files, ok := args["files"].([]interface{})
	if !ok {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: "Invalid 'files' parameter: must be an array of strings",
			}},
			IsError: true,
		}, nil
	}

	// Convert interface{} slice to string slice
	fileList := make([]string, len(files))
	for i, f := range files {
		if str, ok := f.(string); ok {
			fileList[i] = str
		} else {
			return &ToolCallResponse{
				Content: []Content{{
					Type: "text",
					Text: fmt.Sprintf("Invalid file path at index %d: must be a string", i),
				}},
				IsError: true,
			}, nil
		}
	}

	// Create file matcher
	matcher, err := processor.NewMatcher(h.config, h.workingDir)
	if err != nil {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error creating file matcher: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Match files against rules
	matchedResults, err := matcher.MatchFiles(fileList)
	if err != nil {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error matching files: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format results as JSON for the external client to process
	resultData, err := json.MarshalIndent(matchedResults, "", "  ")
	if err != nil {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting results: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolCallResponse{
		Content: []Content{{
			Type: "text",
			Text: string(resultData),
		}},
	}, nil
}

// listRules returns all available rules
func (h *ToolsResourcesHandler) listRules(ctx context.Context, args map[string]interface{}) (*ToolCallResponse, error) {
	rules := make([]map[string]interface{}, 0, len(h.config.Rules))
	for _, rule := range h.config.Rules {
		rules = append(rules, map[string]interface{}{
			"name":        rule.Name,
			"description": rule.Description,
			"enabled":     rule.Enabled,
			"fail_on":     rule.FailOn,
		})
	}

	resultData, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting rules: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolCallResponse{
		Content: []Content{{
			Type: "text",
			Text: string(resultData),
		}},
	}, nil
}

// getRuleDetails returns details for a specific rule
func (h *ToolsResourcesHandler) getRuleDetails(ctx context.Context, args map[string]interface{}) (*ToolCallResponse, error) {
	ruleName, ok := args["rule_name"].(string)
	if !ok {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: "Invalid 'rule_name' parameter: must be a string",
			}},
			IsError: true,
		}, nil
	}

	for _, rule := range h.config.Rules {
		if rule.Name == ruleName {
			resultData, err := json.MarshalIndent(rule, "", "  ")
			if err != nil {
				return &ToolCallResponse{
					Content: []Content{{
						Type: "text",
						Text: fmt.Sprintf("Error formatting rule details: %v", err),
					}},
					IsError: true,
				}, nil
			}

			return &ToolCallResponse{
				Content: []Content{{
					Type: "text",
					Text: string(resultData),
				}},
			}, nil
		}
	}

	return &ToolCallResponse{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Rule not found: %s", ruleName),
		}},
		IsError: true,
	}, nil
}

// matchFiles matches files against rules
func (h *ToolsResourcesHandler) matchFiles(ctx context.Context, args map[string]interface{}) (*ToolCallResponse, error) {
	files, ok := args["files"].([]interface{})
	if !ok {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: "Invalid 'files' parameter: must be an array of strings",
			}},
			IsError: true,
		}, nil
	}

	// Convert interface{} slice to string slice
	fileList := make([]string, len(files))
	for i, f := range files {
		if str, ok := f.(string); ok {
			fileList[i] = str
		} else {
			return &ToolCallResponse{
				Content: []Content{{
					Type: "text",
					Text: fmt.Sprintf("Invalid file path at index %d: must be a string", i),
				}},
				IsError: true,
			}, nil
		}
	}

	// Create file matcher
	matcher, err := processor.NewMatcher(h.config, h.workingDir)
	if err != nil {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error creating file matcher: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Match files
	matchedResults, err := matcher.MatchFiles(fileList)
	if err != nil {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error matching files: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format results
	resultData, err := json.MarshalIndent(matchedResults, "", "  ")
	if err != nil {
		return &ToolCallResponse{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting match results: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolCallResponse{
		Content: []Content{{
			Type: "text",
			Text: string(resultData),
		}},
	}, nil
}

// readConfig reads the configuration
func (h *ToolsResourcesHandler) readConfig(ctx context.Context, uri string) (*ResourceReadResponse, error) {
	configData, err := json.MarshalIndent(h.config, "", "  ")
	if err != nil {
		return &ResourceReadResponse{
			Contents: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error formatting config: %v", err),
			}},
		}, fmt.Errorf("error formatting config: %w", err)
	}

	return &ResourceReadResponse{
		Contents: []Content{{
			Type: "text",
			Text: string(configData),
		}},
	}, nil
}

// readSpec reads a specification file
func (h *ToolsResourcesHandler) readSpec(ctx context.Context, uri string) (*ResourceReadResponse, error) {
	// Extract file path from URI
	specPath := strings.TrimPrefix(uri, "spec://")
	
	// Make path absolute relative to working directory
	if !filepath.IsAbs(specPath) {
		specPath = filepath.Join(h.workingDir, specPath)
	}

	// Read the specification file
	content, err := os.ReadFile(specPath)
	if err != nil {
		return &ResourceReadResponse{
			Contents: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error reading specification file: %v", err),
			}},
		}, fmt.Errorf("error reading specification file: %w", err)
	}

	return &ResourceReadResponse{
		Contents: []Content{{
			Type: "text",
			Text: string(content),
		}},
	}, nil
}