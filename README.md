# Semcheck

A Go-based tool for semantic checking of code implementations against specifications using large language models.

## Overview

Semcheck validates that your code implementations match their specifications by leveraging large language models. It integrates seamlessly with pre-commit hooks to validate staged files and ensures your code adheres to documented requirements.


## Features

- Non-intrusive: don't have to change existing code or specification files
- BYOM: Bring Your Own Model, support for OpenAI, Anthropic, Gemini and Ollama (local)
- Remote specification definitions (for example: `https://www.rfc-editor.org/rfc/rfc7946.txt`)
- Simple onboarding using `semcheck -init`

## Installation

### Prerequisites

- Go 1.24 or later
- [Just](https://github.com/casey/just) (optional, for development)

### Install

```bash
go install github.com/rejot-dev/semcheck@latest
```

## Configuration

Semcheck needs a configuration file to function, one can be generated using the `-init` flag.

```bash
semcheck -init
```

This creates (by default) a `semcheck.yaml` configuration file, edit this file further to fit your needs.

```yaml
version: "1.0"
provider: openai  # Options: openai, anthropic, gemini, ollama
model: gpt-4.1
api_key: ${OPENAI_API_KEY}
base_url: http://localhost:11434  # Optional: for ollama or custom endpoints
timeout: 30
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
      # URL specifications are also supported
      - path: "https://example.com/spec.md"
    fail-on: "error"
    confidence_threshold: 0.8
```

## Usage

### Basic Usage

```bash
# Init config file
semcheck -init

# Pass either implementation or specification files, semcheck will figure out which rules to check based on the files you pass here
semcheck spec.md spec2.md impl.go

# Run semcheck on your staged files
semcheck -pre-commit

# Use custom config file
semcheck -config my-config.yaml file1.go

# Double dash syntax for flags is also accepted
semcheck --config my-config.yaml

# Show help
semcheck -help
```

### Development

This project includes a [Justfile](./Justfile) for starting common development tasks.

```bash
# Show available commands
just
```

### Running Tests

```bash
just test
just test-coverage
```

### Check self

Semcheck has its own semcheck configuration, use the `dogfood` task in the Justfile

```bash
just dogfood
```

## Ideal Situation

![The Office meme: 'Corporate needs you to find the difference between these pictures' showing 'specification' and 'implementation', with semcheck saying 'they are the same picture'](./assets/office-meme.webp)
