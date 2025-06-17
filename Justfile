# Justfile for semcheck project

# Default recipe (runs when just is called without arguments)
default:
    @just --list

# Build the semcheck binary
build:
    go build -o semcheck .

# Run tests for all packages
test:
    go test ./...

# Run tests with verbose output
test-verbose:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
    rm -f semcheck
    rm -f coverage.out coverage.html

# Format code
fmt:
    go fmt ./...

# Run with example config and a test file
demo:
    @just build
    ./semcheck -config examples/correct.yaml main.go

# Run all checks (format, test, lint)
check: fmt test

# Help for available recipes
help:
    @just --list
