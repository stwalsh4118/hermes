# Makefile for Hermes API
# Go development and testing commands

.PHONY: help test lint vet fmt check-fmt coverage clean run install-tools

# Default target
help:
	@echo "Hermes API - Available targets:"
	@echo "  make test          - Run all tests"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make coverage      - Run tests with coverage report"
	@echo "  make lint          - Run golangci-lint"
	@echo "  make vet           - Run go vet"
	@echo "  make fmt           - Format code with gofmt"
	@echo "  make check-fmt     - Check if code is formatted"
	@echo "  make check         - Run all checks (lint, vet, test)"
	@echo "  make run           - Run the server"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install-tools - Install development tools"

# Run all tests
test:
	@echo "Running tests..."
	@go test ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -race ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...
	@echo ""
	@echo "Generating detailed coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Format all Go files
fmt:
	@echo "Formatting Go files..."
	@gofmt -w .
	@echo "Done!"

# Check if code is properly formatted
check-fmt:
	@echo "Checking code formatting..."
	@test -z "$$(gofmt -l . | tee /dev/stderr)"

# Run all quality checks (recommended before submitting for review)
check: check-fmt vet lint test
	@echo ""
	@echo "✓ All checks passed!"

# Run the server
run:
	@echo "Starting Hermes API server..."
	@go run cmd/server/main.go

# Clean build artifacts and test files
clean:
	@echo "Cleaning build artifacts..."
	@rm -f coverage.out
	@rm -f server
	@go clean
	@echo "Done!"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Done! Tools installed:"
	@which golangci-lint

