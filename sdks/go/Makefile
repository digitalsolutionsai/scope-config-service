# Go SDK Makefile
# This Makefile provides targets for building, testing, and linting the SDK.

.PHONY: all build test test-unit test-integration lint fmt clean help

# Default target
all: lint build test-unit

# Build the SDK
build:
	@echo "Building SDK..."
	go build ./...

# Run all tests (unit + integration if server is available)
test:
	@echo "Running all tests..."
	go test -v ./...

# Run unit tests only (no external dependencies)
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

# Run integration tests with testcontainers (requires Docker)
test-integration:
	@echo "Running integration tests with testcontainers..."
	@echo "Note: This requires Docker to be running"
	go test -v -tags=integration -timeout=10m ./tests/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Running go vet instead..."; \
		go vet ./...; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f coverage.out coverage.html
	go clean ./...

# Install development dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Install golangci-lint
install-lint:
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  make build            - Build the SDK"
	@echo "  make test             - Run all tests"
	@echo "  make test-unit        - Run unit tests only"
	@echo "  make test-integration - Run integration tests with testcontainers (requires Docker)"
	@echo "  make test-coverage    - Run tests with coverage report"
	@echo "  make lint             - Run linter"
	@echo "  make fmt              - Format code"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make deps             - Install dependencies"
	@echo "  make install-lint     - Install golangci-lint"
	@echo "  make help             - Show this help"
