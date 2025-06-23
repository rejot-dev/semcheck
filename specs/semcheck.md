# Semcheck Configuration Specification

## Overview

The `semcheck.yaml` file configures the semcheck tool for semantic checking of code implementations against specifications.

## Configuration Schema

### Root Level Configuration

```yaml
# Configuration schema version (REQUIRED)
# Specifies the version of the configuration format being used
# Currently only "1.0" is supported
# This ensures forward compatibility as the tool evolves
version: "1.0"

# AI Provider Configuration (REQUIRED)
# Specifies which AI service to use for semantic analysis
# Supported values:
#   - "openai": Use OpenAI's GPT models (requires API key)
#   - "anthropic": Use Anthropic's Claude models (requires API key)
#   - "local": Use a local LLM endpoint (e.g., Ollama, vLLM)
provider: "openai"

# Model name to use for analysis (REQUIRED)
# The specific model varies by provider:
# OpenAI: "gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"
# Anthropic: "claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"
# Local: depends on your local setup, e.g., "llama2", "codellama"
model: "gpt-4"

# API authentication key (REQUIRED for cloud providers)
# Best practice: Use environment variables for security
# Examples:
#   - "${OPENAI_API_KEY}" (environment variable substitution)
#   - "${ANTHROPIC_API_KEY}" (for Anthropic)
#   - Direct string (NOT recommended for production)
api_key: "${OPENAI_API_KEY}"

# Custom base URL for API requests (OPTIONAL)
# Use cases:
#   - Local LLM servers (e.g., "http://localhost:11434" for Ollama)
#   - Corporate proxy endpoints
#   - Alternative API gateways
# Default: Provider's standard endpoint
base_url: "https://api.openai.com/v1"

# Request timeout in seconds (OPTIONAL)
# How long to wait for AI model responses before timing out
# Considerations:
#   - Larger codebases may need longer timeouts
#   - Complex specifications may require more processing time
#   - Network latency affects response time
# Default: 30 seconds
# Range: 10-300 seconds recommended
timeout: 30

# Maximum retry attempts for failed requests (OPTIONAL)
# Number of times to retry failed API calls due to:
#   - Network issues
#   - Temporary API unavailability
#   - Rate limiting (with exponential backoff)
# Default: 3
# Range: 0-10 recommended
max_retries: 3

# Exit behavior on detected issues (OPTIONAL)
# Controls whether semcheck exits with non-zero code when issues are found
# Usage:
#   - true: Fail CI/CD pipelines on violations (recommended for enforcement)
#   - false: Report issues but don't fail builds (useful for gradual adoption)
# Default: true
fail_on_issues: true

# Checking rules configuration (REQUIRED)
# Array of rules that define what to check and how
rules:
  - # Rule identifier (REQUIRED)
    # Used for logging, reporting, and selective rule execution
    # Must be unique within the configuration file
    # Convention: use kebab-case descriptive names
    name: "api-specification-compliance"

    # Human-readable description (REQUIRED)
    # Explains what this rule checks and why it matters
    # Used in reports and error messages
    description: "Ensures API implementation matches OpenAPI specification"

    # Whether this rule is active (OPTIONAL)
    # Allows temporarily disabling rules without removing them
    # Default: true
    enabled: true

    # File selection patterns (REQUIRED)
    files:
      # Files to include in analysis (REQUIRED)
      # Supports glob patterns with ** for recursive matching
      # Examples of common patterns:
      #   - "src/**/*.go": All Go files in src directory tree
      #   - "lib/**/*.{js,ts}": JavaScript/TypeScript files in lib
      #   - "**/*.py": All Python files in project
      include:
        - "src/**/*.go"
        - "internal/**/*.go"
        - "pkg/**/*.go"

      # Files to exclude from analysis (OPTIONAL)
      # Useful for excluding:
      #   - Test files that may not follow same patterns
      #   - Generated code that shouldn't be manually modified
      #   - Third-party dependencies
      #   - Legacy code not yet brought into compliance
      exclude:
        - "**/*_test.go"
        - "**/*_generated.go"
        - "vendor/**"
        - "third_party/**"
        - "legacy/**"

    # Specification files to check against (REQUIRED)
    # At least one specification must be provided
    specs:
      - # Path to specification file (REQUIRED)
        # Can be relative to config file or absolute
        # Supports various documentation formats
        path: "specs/api.md"

    # Additional context for AI analysis (OPTIONAL)
    # Custom instructions to guide the semantic checking process
    # Use cases:
    #   - Emphasize specific architectural patterns
    #   - Highlight security requirements
    #   - Focus on performance considerations
    #   - Specify coding standards beyond what's in specs
    prompt: |
      When analyzing the code, pay special attention to:
      1. Error handling patterns and consistency
      2. Security considerations (input validation, authentication)
      3. Performance implications of implementation choices
      4. Adherence to established architectural patterns
      5. Code maintainability and readability

      Consider both what is implemented and what might be missing
      compared to the specifications.

    # Severity level for issues found by this rule (OPTIONAL)
    # Controls how violations are reported and handled
    # Values: "error", "warning", "info"
    # Default: "error"
    severity: "error"

    # Custom confidence threshold (OPTIONAL)
    # AI confidence level required to report an issue (0.0-1.0)
    # Higher values reduce false positives but may miss subtle issues
    # Default: 0.8
    confidence_threshold: 0.8

  # Example of a second rule with different focus
  - name: "security-standards"
    description: "Verify implementation follows security best practices"
    enabled: true
    files:
      include:
        - "src/auth/**/*.go"
        - "src/security/**/*.go"
      exclude:
        - "**/*_test.go"
    specs:
      - path: "docs/security-requirements.md"

    prompt: |
      Focus specifically on security vulnerabilities:
      - SQL injection prevention
      - XSS protection
      - Authentication and authorization
      - Input sanitization
      - Secure data handling
    severity: "error"
    confidence_threshold: 0.9
```

## Environment Variable Support

The configuration supports environment variable substitution using `${VARIABLE_NAME}` syntax. This feature provides several benefits:

### Usage Examples

```yaml
# Standard environment variable patterns
api_key: "${OPENAI_API_KEY}"
api_key: "${ANTHROPIC_API_KEY}"
base_url: "${CUSTOM_LLM_ENDPOINT}"
```

## Validation Rules

### Required Fields

- `version`: Must be "1.0" (case-sensitive)
- `provider`: Must be "openai", "anthropic", or "local"
- `model`: Must be a valid model name for the selected provider
- `api_key`: Required for "openai" and "anthropic" providers
- `rules`: Must contain at least one rule object
- `rules[].name`: Must be unique within the configuration
- `rules[].files.include`: Must contain at least one pattern
- `rules[].specs`: Must contain at least one specification

### Optional Field Defaults

- `timeout`: 30 seconds
- `max_retries`: 3 attempts
- `fail_on_issues`: true
- `rules[].enabled`: true
- `rules[].severity`: "error"
- `rules[].confidence_threshold`: 0.8

### File Pattern Validation

- Exclude patterns are applied after include patterns
- Patterns support standard glob syntax: `*`, `**`, `?`, `[abc]`, `{a,b,c}`

### Specification File Validation

- All specified paths must exist and be readable
- File types must match their actual format
