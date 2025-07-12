# MCP (Model Context Protocol) Server Documentation

## Overview

Semcheck supports MCP (Model Context Protocol) server functionality, which allows you to run semcheck as a TCP server that accepts LLM requests and processes them through callback handlers. This enables integration with external systems that can make semantic analysis requests via the MCP protocol.

## Configuration

### MCP Server Configuration

To enable MCP mode, add the following configuration to your `semcheck.yaml` file:

```yaml
version: "1.0"
provider: ollama  # or any other provider (openai, anthropic, etc.)
model: llama3.2   # model to use for LLM requests
timeout: 30
fail_on_issues: true

# MCP Configuration
mcp:
  enabled: true
  address: localhost  # Address to bind the server to
  port: 8080         # Port to listen on

rules:
  - name: example-rule
    description: Example rule for semantic analysis
    enabled: true
    files:
      include:
        - "*.go"
      exclude:
        - "*_test.go"
    specs:
      - path: "specs/example.md"
    fail_on: "error"
```

### MCP Client Configuration

To use semcheck as an MCP client (connecting to an external MCP server):

```yaml
version: "1.0"
provider: mcp      # Use MCP provider
model: mcp-client  # Model identifier for MCP client
timeout: 30
fail_on_issues: true

# MCP Configuration - connects to external MCP server
mcp:
  enabled: true
  address: localhost  # Address of the MCP server
  port: 8080         # Port of the MCP server

rules:
  - name: example-rule
    description: Example rule for semantic analysis
    enabled: true
    files:
      include:
        - "*.go"
      exclude:
        - "*_test.go"
    specs:
      - path: "specs/example.md"
    fail_on: "error"
```

## Usage

### Running MCP Server

To start semcheck in MCP server mode:

```bash
semcheck -config semcheck-mcp.yaml -mcp-server
```

This will:
1. Load the configuration
2. Create an underlying AI provider client (e.g., OpenAI, Anthropic, Ollama)
3. Start a TCP server that accepts MCP protocol requests
4. Handle incoming LLM requests by forwarding them to the underlying provider
5. Return structured responses via the MCP protocol

### Running MCP Client

To use semcheck as an MCP client:

```bash
semcheck -config semcheck-mcp-client.yaml
```

This will:
1. Connect to the configured MCP server
2. Perform semantic analysis using the MCP protocol
3. Send LLM requests to the MCP server instead of directly to AI providers

## MCP Protocol

### Request Format

MCP requests are JSON objects sent over TCP connections:

```json
{
  "id": "unique-request-id",
  "method": "llm_request",
  "system_prompt": "System prompt for the LLM",
  "user_prompt": "User prompt containing the analysis request",
  "max_tokens": 3000,
  "timeout": 30
}
```

### Response Format

MCP responses are JSON objects:

```json
{
  "id": "unique-request-id",
  "result": {
    "usage": {
      "prompt_tokens": 100,
      "completion_tokens": 200,
      "total_tokens": 300
    },
    "issues": [
      {
        "reasoning": "Explanation of the issue",
        "level": "ERROR",
        "message": "Description of the semantic issue",
        "confidence": 0.9,
        "suggestion": "How to fix the issue",
        "file": "path/to/file.go"
      }
    ]
  }
}
```

### Error Format

MCP errors are returned in the response:

```json
{
  "id": "unique-request-id",
  "error": {
    "code": 500,
    "message": "Error description"
  }
}
```

## Architecture

### Components

1. **MCP Server** (`internal/mcp/server.go`):
   - Accepts TCP connections
   - Handles MCP protocol requests
   - Manages concurrent connections
   - Delegates LLM requests to callback handlers

2. **MCP Client** (`internal/providers/mcp.go`):
   - Implements the `providers.Client` interface
   - Connects to MCP servers
   - Sends MCP protocol requests
   - Handles responses and errors

3. **LLM Request Handler** (`internal/mcp/handler.go`):
   - Callback interface for processing LLM requests
   - `DirectLLMRequestHandler`: Forwards requests to provider clients
   - Extensible for custom handler implementations

### Callback System

The MCP server uses a callback system to handle LLM requests:

```go
type LLMRequestHandler interface {
    HandleLLMRequest(ctx context.Context, req *providers.Request) (*providers.Response, error)
}
```

This allows for:
- Custom processing logic
- Request routing
- Response transformation
- Integration with external systems

## Configuration Options

### MCP Section

- `enabled` (bool): Enable/disable MCP functionality
- `address` (string): Server bind address (default: "localhost")
- `port` (int): Server port (default: 8080)

### Provider Integration

When MCP is enabled:
- **Server mode**: Uses the configured provider (openai, anthropic, etc.) as the backend
- **Client mode**: Uses `provider: mcp` to connect to an external MCP server

## Testing

### Unit Tests

Run MCP unit tests:

```bash
go test ./internal/mcp/...
go test ./internal/providers/...
```

### Integration Tests

The integration tests verify end-to-end MCP functionality:

```bash
go test ./internal/mcp/... -v
```

### Manual Testing

1. Start MCP server:
   ```bash
   semcheck -config semcheck-mcp.yaml -mcp-server
   ```

2. In another terminal, test with MCP client:
   ```bash
   semcheck -config semcheck-mcp-client.yaml main.go
   ```

## Use Cases

### 1. Distributed Semantic Analysis

Run semcheck as a service that multiple clients can connect to:

```bash
# Server (with expensive GPU/API access)
semcheck -config server-config.yaml -mcp-server

# Clients (lightweight, connect to server)
semcheck -config client-config.yaml file1.go
semcheck -config client-config.yaml file2.go
```

### 2. Integration with External Systems

Create custom MCP clients that integrate with:
- CI/CD systems
- Code review tools
- IDE plugins
- Automated testing frameworks

### 3. Load Balancing

Use MCP to distribute LLM requests across multiple providers or instances:

```go
// Custom handler that implements load balancing
type LoadBalancingHandler struct {
    providers []providers.Client
    current   int
}

func (h *LoadBalancingHandler) HandleLLMRequest(ctx context.Context, req *providers.Request) (*providers.Response, error) {
    // Round-robin load balancing
    provider := h.providers[h.current]
    h.current = (h.current + 1) % len(h.providers)
    return provider.Complete(ctx, req)
}
```

## Error Handling

The MCP implementation includes comprehensive error handling:

- **Connection errors**: Automatic reconnection for clients
- **Protocol errors**: Structured error responses
- **Provider errors**: Proper error propagation
- **Timeout handling**: Configurable timeouts for requests

## Security Considerations

- **Network Security**: MCP servers bind to localhost by default
- **Authentication**: No built-in authentication (add firewall rules or proxy)
- **Rate Limiting**: No built-in rate limiting (implement in custom handlers)
- **Input Validation**: All inputs are validated before processing

## Troubleshooting

### Common Issues

1. **Connection Refused**:
   - Ensure MCP server is running
   - Check address and port configuration
   - Verify firewall settings

2. **Protocol Errors**:
   - Verify JSON format of requests
   - Check required fields in MCP requests
   - Ensure compatible versions

3. **Provider Errors**:
   - Check underlying provider configuration
   - Verify API keys and endpoints
   - Review provider-specific error messages

### Debug Mode

Enable debug logging by setting environment variables:

```bash
export DEBUG=1
semcheck -config semcheck-mcp.yaml -mcp-server
```

## Future Enhancements

Potential future improvements:

1. **Authentication**: Add token-based authentication
2. **TLS Support**: Encrypt MCP connections
3. **Metrics**: Add prometheus metrics for monitoring
4. **Discovery**: Add service discovery for MCP servers
5. **Streaming**: Support streaming responses for large analyses