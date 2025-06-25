# Justfile for semcheck project

# Default recipe (runs when just is called without arguments)
default:
    @just --list

# Run semcheck against itself on all files
# uses find because glob patterns are handled differently per shell used
dogfood: build
    ./semcheck $(find . -type f -name "*" | grep -v ".git")

pre-commit: check build build-eval
    ./semcheck $(git diff --name-only --cached)

# Install pre-commit hook
install-pre-commit:
    @echo "Installing pre-commit hook..."
    @mkdir -p .git/hooks
    @echo '#!/bin/sh' > .git/hooks/pre-commit
    @echo 'just pre-commit' >> .git/hooks/pre-commit
    @echo 'exit $?' >> .git/hooks/pre-commit
    @chmod +x .git/hooks/pre-commit
    @echo "✅ Pre-commit hook installed successfully!"
    @echo "The hook will run 'just pre-commit' before each commit."

# Build the semcheck binary
build:
    go build -o semcheck .

build-eval:
    go build -o semcheck-eval ./cmd/eval

# Run tests for all packages
test:
    go test ./...

# Run tests with verbose output
test-verbose:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -coverprofile=coverage.out.tmp ./...
    cat coverage.out.tmp | grep -v "evals/cases" > coverage.out
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
    rm -f semcheck semcheck-eval
    rm -f coverage.out coverage.html coverage.out.tmp

# Format code
fmt:
    go fmt ./...

# Run evaluation suite to test semcheck performance
eval: build-eval
    ./semcheck-eval

# Run all checks (format, test, lint)
check: fmt test

# Help for available recipes
help:
    @just --list
