package mcp

import (
	"context"
	"fmt"

	"github.com/rejot-dev/semcheck/internal/providers"
)

// DirectLLMRequestHandler is an implementation that directly uses a provider client
type DirectLLMRequestHandler struct {
	client providers.Client
}

// NewDirectLLMRequestHandler creates a new direct LLM request handler
func NewDirectLLMRequestHandler(client providers.Client) *DirectLLMRequestHandler {
	return &DirectLLMRequestHandler{
		client: client,
	}
}

// HandleLLMRequest handles LLM requests by calling the provider client directly
func (h *DirectLLMRequestHandler) HandleLLMRequest(ctx context.Context, req *providers.Request) (*providers.Response, error) {
	if h.client == nil {
		return nil, fmt.Errorf("provider client not configured")
	}

	// Call the provider client directly
	return h.client.Complete(ctx, req)
}