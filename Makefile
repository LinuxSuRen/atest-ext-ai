# Makefile for atest-ext-ai plugin

.PHONY: all build test clean install proto help

# Build configuration
BINARY_NAME=atest-store-ai
BUILD_DIR=bin
GO_VERSION=1.22
MAIN_PACKAGE=./cmd/atest-store-ai

# Go build flags
LDFLAGS=-ldflags "-s -w"
TAGS=

# Default target
all: clean proto build test

# Build the plugin binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: clean proto
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Multi-platform build completed"

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Tests completed. Coverage report: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./test/integration/...

# Generate protobuf files (if needed in future)
proto:
	@echo "Checking protobuf files..."
	# Currently using main project's protobuf files via go.mod dependency
	@echo "Protobuf files are imported from main project dependency"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean -cache

# Install the plugin binary to /usr/local/bin (requires sudo)
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation completed"

# Uninstall the plugin binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstallation completed"

# Run the plugin locally for development
dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	@AI_PLUGIN_SOCKET_PATH="/tmp/atest-store-ai-dev.sock" \
	 AI_PROVIDER="local" \
	 OLLAMA_ENDPOINT="http://localhost:11434" \
	 AI_MODEL="codellama" \
	 go run $(MAIN_PACKAGE)

# Format Go code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Run security check
security:
	@echo "Running security check..."
	@gosec ./...

# Generate documentation
docs:
	@echo "Generating documentation..."
	@go doc -all ./... > docs/api.md

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t atest-ext-ai:latest .

# Docker run
docker-run:
	@echo "Running Docker container..."
	@docker run --rm -v /tmp:/tmp atest-ext-ai:latest

# Check Go modules
mod-verify:
	@echo "Verifying Go modules..."
	@go mod verify

# Benchmark tests
benchmark:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the plugin binary"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  test          - Run tests with coverage"
	@echo "  test-integration - Run integration tests"
	@echo "  proto         - Check protobuf files"
	@echo "  deps          - Install dependencies"
	@echo "  update-deps   - Update dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to /usr/local/bin"
	@echo "  uninstall     - Uninstall binary"
	@echo "  dev           - Run in development mode"
	@echo "  fmt           - Format Go code"
	@echo "  lint          - Run linter"
	@echo "  security      - Run security check"
	@echo "  docs          - Generate documentation"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  mod-verify    - Verify Go modules"
	@echo "  benchmark     - Run benchmark tests"
	@echo "  help          - Show this help"

# Development workflow targets
dev-setup: deps fmt lint test build
	@echo "Development environment setup completed"

ci: clean deps fmt lint security test build
	@echo "CI pipeline completed"