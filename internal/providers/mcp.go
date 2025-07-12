package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
)

// MCPClient represents an MCP client that connects to an MCP server
type MCPClient struct {
	address string
	port    int
	conn    net.Conn
	encoder *json.Encoder
	decoder *json.Decoder
	reqID   int64
}

// MCPRequest represents a request sent via MCP protocol
type MCPRequest struct {
	ID           string `json:"id"`
	Method       string `json:"method"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	UserPrompt   string `json:"user_prompt,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	Timeout      int    `json:"timeout,omitempty"`
}

// MCPResponse represents a response received via MCP protocol
type MCPResponse struct {
	ID     string    `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError `json:"error,omitempty"`
}

// MCPError represents an error in MCP protocol
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewMCPClient creates a new MCP client
func NewMCPClient(address string, port int) *MCPClient {
	return &MCPClient{
		address: address,
		port:    port,
	}
}

// Connect establishes a connection to the MCP server
func (c *MCPClient) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.address, c.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to MCP server at %s: %w", addr, err)
	}

	c.conn = conn
	c.encoder = json.NewEncoder(conn)
	c.decoder = json.NewDecoder(conn)

	return nil
}

// Disconnect closes the connection to the MCP server
func (c *MCPClient) Disconnect() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.encoder = nil
		c.decoder = nil
		return err
	}
	return nil
}

// Complete sends a completion request to the MCP server
func (c *MCPClient) Complete(ctx context.Context, req *Request) (*Response, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	// Generate unique request ID
	reqID := fmt.Sprintf("%d", atomic.AddInt64(&c.reqID, 1))

	// Create MCP request
	mcpReq := &MCPRequest{
		ID:           reqID,
		Method:       "llm_request",
		SystemPrompt: req.SystemPrompt,
		UserPrompt:   req.UserPrompt,
		MaxTokens:    req.MaxTokens,
		Timeout:      int(req.Timeout.Seconds()),
	}

	// Send request
	if err := c.encoder.Encode(mcpReq); err != nil {
		return nil, fmt.Errorf("failed to send MCP request: %w", err)
	}

	// Receive response
	var mcpResp MCPResponse
	if err := c.decoder.Decode(&mcpResp); err != nil {
		return nil, fmt.Errorf("failed to receive MCP response: %w", err)
	}

	// Check for errors
	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP server error: %s", mcpResp.Error.Message)
	}

	// Parse result as providers.Response
	resultBytes, err := json.Marshal(mcpResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP result: %w", err)
	}

	var providerResp Response
	if err := json.Unmarshal(resultBytes, &providerResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider response: %w", err)
	}

	return &providerResp, nil
}

// Name returns the name of the client
func (c *MCPClient) Name() string {
	return "mcp"
}

// Validate validates the client configuration
func (c *MCPClient) Validate() error {
	if c.address == "" {
		return fmt.Errorf("MCP address is required")
	}
	if c.port <= 0 || c.port > 65535 {
		return fmt.Errorf("MCP port must be between 1 and 65535")
	}
	return nil
}