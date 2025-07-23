# Semcheck Implementation Plan

## Overview

Semcheck is a Go 1.24-based tool for semantic checking implementations and specifications are consistent using AI language models.

Semcheck is meant to be run as part of merge or pull request CI pipelines.

- On changes to specifications, check all matching implementations
- On changes to implementations, check all specifications

A semcheck.yaml config file determines the mappings between implementation and spec.

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

IMPORTANT: When creating features using code, consider adding tests for those features!

Sign off on phases once completed.

### Phase 1: Core Infrastructure

- [x] Initialize Go module with Go 1.24
- [x] Implement configuration parser for `semcheck.yaml`
- [x] Create CLI argument parsing
- [x] Set up project structure and dependencies

### Phase 2: Initial AI Integration

- [x] Create unified AI client interface
- [x] Implement OpenAI API client (skip the other providers for now)

### Phase 3: Implement file matching

- [x] Read .gitignore file for exclude list
- [x] Match input files to rules and assigned file type: "spec" file, "impl" file or an "ignored" file
- [x] For implementation files, find the associated specification files based on the rules
- [x] For specification files, find the associated implementation files based on the rules

### Phase 4: Semantic Analysis

- [x] Implement spec-to-implementation comparison
- [x] Create reporting system

### Phase 5: Expansion

- [x] Implement Anthropic client
- [x] Implement Local LLM client

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
    └── correct.yaml
```

## Error Handling

- Graceful handling of network failures
- Clear error messages for configuration issues
- Appropriate exit codes for CI/CD integration

## Performance Considerations

- Concurrent processing of multiple files
- Efficient API request batching
- Configurable timeout settings
