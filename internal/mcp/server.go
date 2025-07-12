package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/rejot-dev/semcheck/internal/providers"
)

// Server represents an MCP server that accepts TCP connections
type Server struct {
	address string
	port    int
	handler LLMRequestHandler
	mu      sync.RWMutex
	running bool
	ln      net.Listener
}

// LLMRequestHandler defines the callback interface for handling LLM requests
type LLMRequestHandler interface {
	HandleLLMRequest(ctx context.Context, req *providers.Request) (*providers.Response, error)
}

// MCPRequest represents a request received via MCP protocol
type MCPRequest struct {
	ID         string                 `json:"id"`
	Method     string                 `json:"method"`
	Params     map[string]interface{} `json:"params"`
	SystemPrompt string               `json:"system_prompt,omitempty"`
	UserPrompt   string               `json:"user_prompt,omitempty"`
	MaxTokens    int                  `json:"max_tokens,omitempty"`
	Timeout      int                  `json:"timeout,omitempty"`
}

// MCPResponse represents a response sent via MCP protocol
type MCPResponse struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error in MCP protocol
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewServer creates a new MCP server
func NewServer(address string, port int, handler LLMRequestHandler) *Server {
	return &Server{
		address: address,
		port:    port,
		handler: handler,
	}
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	addr := fmt.Sprintf("%s:%d", s.address, s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.ln = ln
	s.running = true

	fmt.Printf("MCP server started on %s\n", addr)

	go s.acceptConnections(ctx)

	return nil
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.ln != nil {
		err := s.ln.Close()
		s.ln = nil
		return err
	}

	return nil
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// acceptConnections accepts incoming TCP connections
func (s *Server) acceptConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				s.mu.RLock()
				if !s.running {
					s.mu.RUnlock()
					return
				}
				s.mu.RUnlock()
				fmt.Printf("Error accepting connection: %v\n", err)
				continue
			}

			go s.handleConnection(ctx, conn)
		}
	}
}

// handleConnection handles an individual TCP connection
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	fmt.Printf("New MCP connection from %s\n", conn.RemoteAddr())

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var req MCPRequest
			if err := decoder.Decode(&req); err != nil {
				fmt.Printf("Error decoding request: %v\n", err)
				return
			}

			response := s.processRequest(ctx, &req)
			if err := encoder.Encode(response); err != nil {
				fmt.Printf("Error encoding response: %v\n", err)
				return
			}
		}
	}
}

// processRequest processes an MCP request
func (s *Server) processRequest(ctx context.Context, req *MCPRequest) *MCPResponse {
	response := &MCPResponse{
		ID: req.ID,
	}

	switch req.Method {
	case "llm_request":
		result, err := s.handleLLMRequest(ctx, req)
		if err != nil {
			response.Error = &MCPError{
				Code:    500,
				Message: err.Error(),
			}
		} else {
			response.Result = result
		}
	default:
		response.Error = &MCPError{
			Code:    400,
			Message: fmt.Sprintf("unknown method: %s", req.Method),
		}
	}

	return response
}

// handleLLMRequest handles LLM requests via the callback handler
func (s *Server) handleLLMRequest(ctx context.Context, req *MCPRequest) (interface{}, error) {
	// Convert MCP request to providers.Request
	providerReq := &providers.Request{
		SystemPrompt: req.SystemPrompt,
		UserPrompt:   req.UserPrompt,
		MaxTokens:    req.MaxTokens,
		Timeout:      time.Duration(req.Timeout) * time.Second,
	}

	// Set defaults if not provided
	if providerReq.MaxTokens == 0 {
		providerReq.MaxTokens = 3000
	}
	if providerReq.Timeout == 0 {
		providerReq.Timeout = 30 * time.Second
	}

	// Call the handler
	response, err := s.handler.HandleLLMRequest(ctx, providerReq)
	if err != nil {
		return nil, err
	}

	return response, nil
}