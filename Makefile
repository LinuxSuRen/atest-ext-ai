# Makefile for atest-ext-ai plugin

.PHONY: all build test clean install install-local help dev docker-build docker-release docker-release-github

# Build configuration
BINARY_NAME=atest-ext-ai
BUILD_DIR=bin
MAIN_PACKAGE=./cmd/atest-ext-ai
DOCKER_IMAGE=atest-ext-ai
DOCKER_REGISTRY ?= ghcr.io/linuxsuren
DOCKER_TAG ?= $(VERSION)
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

# Run tests
test:
	@echo "Running tests..."
	@go test -v -coverprofile=coverage.out ./...
	@echo "Tests completed."

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

# Install the plugin binary to system location
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation completed"

# Install the plugin binary for local development and testing
install-local: build
	@echo "Installing $(BINARY_NAME) for local development..."
	@mkdir -p ~/.config/atest/bin/
	@cp $(BUILD_DIR)/$(BINARY_NAME) ~/.config/atest/bin/
	@chmod +x ~/.config/atest/bin/$(BINARY_NAME)
	@echo "Local installation completed: ~/.config/atest/bin/$(BINARY_NAME)"
	@echo "The plugin will be automatically discovered by the API Testing Server"

# Run development mode
dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	@AI_PROVIDER="local" \
	 OLLAMA_ENDPOINT="http://localhost:11434" \
	 AI_MODEL="gemma3:1b" \
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

# Build and push OCI image to registry
docker-release: docker-build
	@echo "Tagging and pushing Docker image to $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)..."
	@docker tag $(DOCKER_IMAGE):latest $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker tag $(DOCKER_IMAGE):latest $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	@echo "Docker image pushed successfully to $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)"

# Quick release to GitHub Container Registry (uses default registry)
docker-release-github:
	@$(MAKE) docker-release DOCKER_REGISTRY=ghcr.io/linuxsuren

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the plugin binary for local development"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install binary to /usr/local/bin"
	@echo "  install-local - Install binary to ~/.config/atest/bin for local development"
	@echo "  dev          - Run in development mode"
	@echo "  fmt          - Format Go code"
	@echo "  deps         - Install dependencies"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-release - Build and push Docker image to registry (default: ghcr.io/linuxsuren)"
	@echo "  docker-release-github - Build and push Docker image to GitHub Container Registry"
	@echo "  help         - Show this help"