# Makefile for atest-ext-ai plugin

.PHONY: all build test clean install help dev

# Build configuration
BINARY_NAME=atest-ext-ai
BUILD_DIR=bin
MAIN_PACKAGE=./cmd/atest-ext-ai
DOCKER_IMAGE=atest-ext-ai
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Default target
all: clean build test

# Build the plugin binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for release (multiple platforms)
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@echo "Multi-platform build completed"

# Run tests
test:
	@echo "Running tests..."
	@go test -v -coverprofile=coverage.out ./...
	@echo "Tests completed."

# Run integration tests (disabled - integration tests removed during simplification)
test-integration:
	@echo "Integration tests have been removed during architecture simplification"
	@echo "Use 'make test' to run unit tests instead"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out
	@go clean -cache

# Install the plugin binary
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation completed"

# Run development mode
dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	@AI_PLUGIN_SOCKET_PATH="/tmp/atest-ext-ai.sock" \
	 AI_PROVIDER="local" \
	 OLLAMA_ENDPOINT="http://localhost:11434" \
	 AI_MODEL="codellama" \
	 LOG_LEVEL="debug" \
	 go run $(MAIN_PACKAGE)

# Format Go code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Verify Go modules
mod-verify:
	@echo "Verifying Go modules..."
	@go mod verify

# Run benchmark tests
benchmark:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./...

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE):latest .

# Docker development environment (disabled - docker-compose.dev.yml removed during simplification)
dev-up:
	@echo "Development Docker environment has been removed during architecture simplification"
	@echo "Use 'make dev' to run the plugin directly instead"

dev-down:
	@echo "Development Docker environment has been removed during architecture simplification"
	@echo "Use Ctrl+C to stop the plugin running with 'make dev'"

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the plugin binary"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  test         - Run tests"
	@echo "  test-integration - Run integration tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install binary to /usr/local/bin"
	@echo "  dev          - Run in development mode"
	@echo "  fmt          - Format Go code"
	@echo "  deps         - Install dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  dev-up       - Start development environment"
	@echo "  dev-down     - Stop development environment"
	@echo "  help         - Show this help"