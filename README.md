# Semcheck

A Go-based tool for semantic checking of code implementations against specifications using large language models.

## Overview

Semcheck validates that your code implementations match their specifications by leveraging large language models. It integrates seamlessly with pre-commit hooks to validate staged files and ensures your code adheres to documented requirements.

## Installation

### Prerequisites

- Go 1.24 or later
- [Just](https://github.com/casey/just) (optional, for development)

### Building from Source

```bash
git clone git@github.com:rejot-dev/semcheck.git
cd semcheck
go build -o semcheck .
```

## Configuration

Create a `semcheck.yaml` configuration file:

```yaml
version: "1.0"
provider: openai
model: gpt-4
api_key: ${OPENAI_API_KEY}
timeout: 30
max_retries: 3
fail_on_issues: true

rules:
  - name: function-spec-compliance
    description: Check if functions match their specifications
    enabled: true
    files:
      include:
        - "**/*.go"
      exclude:
        - "*_test.go"
    specs:
      - path: "docs/api.md"
        type: "markdown"
    severity: "error"
    confidence_threshold: 0.8
```

## Usage

### Basic Usage

```bash
# Check specific files
./semcheck file1.go file2.go

# Use custom config
./semcheck -config my-config.yaml file1.go

# Show help
./semcheck -help
```

### Development

This project includes a [Justfile](./Justfile) for starting common development tasks.

```bash
# Show available commands
just
```

### Project Structure

```
semcheck/
├── cmd/           # CLI command implementation
├── internal/
│   ├── config/    # Configuration parsing and validation
│   ├── providers/ # AI provider implementations
│   ├── processor/ # File processing logic
│   └── checker/   # Semantic checking logic
├── examples/      # Example configurations
├── specs/         # Project specifications
└── Justfile       # Development task runner
```

### Running Tests

```bash
just test
just test-coverage
```

## Ideal Situation

![The Office meme: 'Corporate needs you to find the difference between these pictures' showing 'specification' and 'implementation', with semcheck saying 'they are the same picture'](./assets/office-meme.webp)
