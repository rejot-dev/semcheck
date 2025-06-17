# Semcheck Implementation Plan

## Overview

Semcheck is a Go 1.24-based tool for semantic checking of code implementations against specifications using AI language models. It integrates with pre-commit hooks to validate staged files.

## Architecture

### Core Components

1. **Configuration Parser**

   - Parse `semcheck.yaml` configuration file
   - Validate configuration schema
   - Support for multiple LLM providers (OpenAI, Anthropic, local)

2. **Command Line Interface**

   - Accept list of files as arguments
   - Support common flags (`--config`, `--help`, `--version`)

3. **File Processor**

   - Read and process input files
   - Extract relevant code sections
   - Prepare context for AI analysis

4. **AI Client**

   - Unified interface for multiple LLM providers
   - API key management and authentication
   - Request/response handling with retry logic

5. **Semantic Checker**
   - Compare specifications against implementations
   - Generate detailed reports
   - Return appropriate exit codes
   - Dog food the tool by checking against its own spec in ./specs/semcheck.md

## Implementation Phases

IMPORTANT: When creating features using code, consider adding a few tests for those features!

### Phase 1: Core Infrastructure

- [ ] Initialize Go module with Go 1.24
- [ ] Implement configuration parser for `semcheck.yaml`
- [ ] Create CLI argument parsing
- [ ] Set up project structure and dependencies

### Phase 2: Initial AI Integration

- [ ] Create unified AI client interface
- [ ] Implement OpenAI API client (skip the other providers for now)

### Phase 3: File Processing

- [ ] Implement file reading and parsing
- [ ] Add support for multiple file formats
- [ ] Create context extraction logic

### Phase 4: Semantic Analysis

- [ ] Implement spec-to-implementation comparison
- [ ] Create reporting system
- [ ] Add exit code handling

### Phase 5: Integration & Testing

- [ ] Pre-commit hook integration
- [ ] Comprehensive testing suite
- [ ] Documentation and examples

### Phase 6: Expansion

- [ ] Implement Local LLM client
- [ ] Implement Anthropic client

## Code Conventions

Make sure to factor code in a way that it can be easily be tested.

## Dependencies

Prefer stdlib dependencies over third-party dependencies where possible.

- `flags` for cli argument parsing

## Directory Structure

```
semcheck/
├── main.go
├── go.mod
├── go.sum
├── cmd/
│   └── root.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── providers/
│   │   ├── client.go
│   │   ├── openai.go
│   │   ├── anthropic.go
│   │   └── local.go
│   ├── processor/
│   │   └── file.go
│   └── checker/
│       └── semantic.go
├── specs/
│   └── semcheck.md
└── examples/
    └── semcheck.yaml
```

## Error Handling

- Graceful handling of network failures
- Clear error messages for configuration issues
- Appropriate exit codes for CI/CD integration

## Performance Considerations

- Concurrent processing of multiple files
- Efficient API request batching
- Configurable timeout settings
