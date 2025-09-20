# Makefile for atest-ext-ai plugin

.PHONY: all build test clean install proto help deploy release

# Build configuration
BINARY_NAME=atest-ext-ai
BUILD_DIR=bin
GO_VERSION=1.22
MAIN_PACKAGE=./cmd/atest-ext-ai
DOCKER_REGISTRY=ghcr.io
DOCKER_IMAGE=$(DOCKER_REGISTRY)/linuxsuren/atest-ext-ai
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go build flags with version info
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildDate=$(BUILD_DATE)"
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
	@AI_PLUGIN_SOCKET_PATH="/tmp/atest-ext-ai-dev.sock" \
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

# ==============================================================================
# DEPLOYMENT TARGETS
# ==============================================================================

# Build and tag Docker images
docker-build-multi:
	@echo "Building multi-platform Docker images..."
	@docker buildx create --use --name atest-builder 2>/dev/null || true
	@docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		--push .

# Build Docker image for local testing
docker-build-local:
	@echo "Building local Docker image..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest .

# Push Docker images to registry
docker-push:
	@echo "Pushing Docker images to registry..."
	@docker push $(DOCKER_IMAGE):$(VERSION)
	@docker push $(DOCKER_IMAGE):latest

# Start development environment with Docker Compose
dev-up:
	@echo "Starting development environment..."
	@docker-compose -f docker-compose.dev.yml up -d
	@echo "Development environment started"
	@echo "Plugin available at: unix:///tmp/atest-ext-ai.sock"
	@echo "Ollama available at: http://localhost:11434"
	@echo "Adminer available at: http://localhost:8081"

# Stop development environment
dev-down:
	@echo "Stopping development environment..."
	@docker-compose -f docker-compose.dev.yml down -v

# Start production environment with Docker Compose
prod-up:
	@echo "Starting production environment..."
	@docker-compose up -d
	@echo "Production environment started"

# Stop production environment
prod-down:
	@echo "Stopping production environment..."
	@docker-compose down

# Deploy to Kubernetes (development)
k8s-deploy-dev:
	@echo "Deploying to Kubernetes (development)..."
	@kubectl create namespace atest-system --dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -f k8s/configmap.yaml
	@kubectl apply -f k8s/secret.yaml
	@kubectl apply -f k8s/rbac.yaml
	@kubectl apply -f k8s/pvc.yaml
	@kubectl apply -f k8s/deployment.yaml
	@kubectl apply -f k8s/service.yaml
	@echo "Waiting for deployment to be ready..."
	@kubectl wait --for=condition=available --timeout=300s deployment/atest-ai-plugin -n atest-system
	@echo "Deployment completed"

# Deploy to Kubernetes (production)
k8s-deploy-prod:
	@echo "Deploying to Kubernetes (production)..."
	@kubectl create namespace atest-system --dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -f k8s/configmap.yaml
	@kubectl apply -f k8s/secret.yaml
	@kubectl apply -f k8s/rbac.yaml
	@kubectl apply -f k8s/pvc.yaml
	@kubectl apply -f k8s/deployment.yaml
	@kubectl apply -f k8s/service.yaml
	@kubectl apply -f k8s/hpa.yaml
	@kubectl apply -f k8s/ingress.yaml
	@echo "Waiting for deployment to be ready..."
	@kubectl wait --for=condition=available --timeout=600s deployment/atest-ai-plugin -n atest-system
	@echo "Production deployment completed"

# Remove from Kubernetes
k8s-remove:
	@echo "Removing from Kubernetes..."
	@kubectl delete -f k8s/ --ignore-not-found=true
	@echo "Kubernetes resources removed"

# Update Kubernetes deployment with new image
k8s-update:
	@echo "Updating Kubernetes deployment..."
	@kubectl set image deployment/atest-ai-plugin atest-ai-plugin=$(DOCKER_IMAGE):$(VERSION) -n atest-system
	@kubectl rollout status deployment/atest-ai-plugin -n atest-system
	@echo "Deployment updated"

# ==============================================================================
# RELEASE TARGETS
# ==============================================================================

# Create release archives
release-archives:
	@echo "Creating release archives..."
	@mkdir -p dist
	@for file in $(BUILD_DIR)/$(BINARY_NAME)-*; do \
		if [[ "$$file" == *".exe" ]]; then \
			zip -j "dist/$$(basename $$file .exe).zip" "$$file"; \
		else \
			tar -czf "dist/$$(basename $$file).tar.gz" -C $(BUILD_DIR) "$$(basename $$file)"; \
		fi; \
	done
	@echo "Release archives created in dist/"

# Generate checksums
release-checksums:
	@echo "Generating checksums..."
	@cd dist && find . -name "*.tar.gz" -o -name "*.zip" | xargs sha256sum > SHA256SUMS
	@cd dist && find . -name "*.tar.gz" -o -name "*.zip" | xargs md5sum > MD5SUMS
	@echo "Checksums generated"

# Create GitHub release (requires gh CLI)
github-release:
	@echo "Creating GitHub release $(VERSION)..."
	@gh release create $(VERSION) dist/* \
		--title "Release $(VERSION)" \
		--notes-file CHANGELOG.md \
		--draft=false \
		--prerelease=false
	@echo "GitHub release created"

# Complete release build
release: clean deps build-all release-archives release-checksums docker-build-multi
	@echo "Release $(VERSION) completed"
	@echo "Binaries available in dist/"
	@echo "Docker images pushed to $(DOCKER_IMAGE):$(VERSION)"

# ==============================================================================
# MAINTENANCE TARGETS
# ==============================================================================

# Update all dependencies
update-all: update-deps
	@echo "Updating Docker base images..."
	@docker pull golang:1.22-alpine
	@docker pull alpine:3.19
	@echo "All dependencies updated"

# Security audit
audit:
	@echo "Running security audit..."
	@go list -json -m all | nancy sleuth
	@docker scout cves $(DOCKER_IMAGE):latest 2>/dev/null || echo "Docker Scout not available"
	@echo "Security audit completed"

# Performance profiling
profile:
	@echo "Running performance profiling..."
	@go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof ./...
	@echo "Profile files generated: cpu.prof, mem.prof"
	@echo "Analyze with: go tool pprof cpu.prof"

# Load testing
load-test:
	@echo "Running load tests..."
	@if command -v hey >/dev/null 2>&1; then \
		hey -n 1000 -c 10 -m POST -H "Content-Type: application/json" \
			-d '{"type":"ai","natural_language":"Show all users","database_type":"mysql"}' \
			http://localhost:8080/api/v1/data/query; \
	else \
		echo "Install 'hey' for load testing: go install github.com/rakyll/hey@latest"; \
	fi

# Health check all services
health-check:
	@echo "Checking service health..."
	@curl -f http://localhost:9090/health || echo "❌ Plugin health check failed"
	@curl -f http://localhost:11434/api/tags || echo "❌ Ollama health check failed"
	@test -S /tmp/atest-ext-ai.sock && echo "✅ Unix socket available" || echo "❌ Unix socket not available"

# Backup configuration and data
backup:
	@echo "Creating backup..."
	@mkdir -p backups
	@tar -czf backups/config-$(shell date +%Y%m%d-%H%M%S).tar.gz config/ docs/
	@echo "Backup created in backups/"

# ==============================================================================
# MONITORING TARGETS
# ==============================================================================

# Show current metrics
metrics:
	@echo "Current metrics:"
	@curl -s http://localhost:9090/metrics | grep -E "(atest_ai_|go_|process_)" | head -20

# Show logs
logs:
	@echo "Recent logs:"
	@if command -v journalctl >/dev/null 2>&1; then \
		journalctl -u atest-ai-plugin -n 20 --no-pager; \
	else \
		docker-compose logs --tail=20 atest-ai-plugin 2>/dev/null || echo "No logs available"; \
	fi

# Monitor resource usage
monitor:
	@echo "Resource usage:"
	@ps aux | grep atest-ext-ai | grep -v grep || echo "Process not running"
	@df -h /tmp /var/log 2>/dev/null || true
	@free -h || echo "Memory info not available"

# ==============================================================================
# HELP TARGET
# ==============================================================================

# Show help
help:
	@echo "atest-ext-ai Makefile"
	@echo ""
	@echo "Build Targets:"
	@echo "  build              - Build the plugin binary"
	@echo "  build-all          - Build for multiple platforms"
	@echo "  clean              - Clean build artifacts"
	@echo ""
	@echo "Development Targets:"
	@echo "  dev                - Run in development mode"
	@echo "  dev-up             - Start development environment (Docker Compose)"
	@echo "  dev-down           - Stop development environment"
	@echo "  dev-setup          - Setup development environment"
	@echo ""
	@echo "Testing Targets:"
	@echo "  test               - Run tests with coverage"
	@echo "  test-integration   - Run integration tests"
	@echo "  benchmark          - Run benchmark tests"
	@echo "  load-test          - Run load tests (requires hey)"
	@echo ""
	@echo "Quality Targets:"
	@echo "  fmt                - Format Go code"
	@echo "  lint               - Run linter"
	@echo "  security           - Run security check"
	@echo "  audit              - Run security audit"
	@echo ""
	@echo "Docker Targets:"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-build-local - Build local Docker image"
	@echo "  docker-build-multi - Build multi-platform images and push"
	@echo "  docker-push        - Push Docker images"
	@echo ""
	@echo "Kubernetes Targets:"
	@echo "  k8s-deploy-dev     - Deploy to Kubernetes (development)"
	@echo "  k8s-deploy-prod    - Deploy to Kubernetes (production)"
	@echo "  k8s-update         - Update Kubernetes deployment"
	@echo "  k8s-remove         - Remove from Kubernetes"
	@echo ""
	@echo "Release Targets:"
	@echo "  release            - Create complete release"
	@echo "  release-archives   - Create release archives"
	@echo "  release-checksums  - Generate checksums"
	@echo "  github-release     - Create GitHub release (requires gh CLI)"
	@echo ""
	@echo "Production Targets:"
	@echo "  prod-up            - Start production environment"
	@echo "  prod-down          - Stop production environment"
	@echo "  install            - Install binary to /usr/local/bin"
	@echo "  uninstall          - Uninstall binary"
	@echo ""
	@echo "Maintenance Targets:"
	@echo "  deps               - Install dependencies"
	@echo "  update-deps        - Update dependencies"
	@echo "  update-all         - Update all dependencies including Docker images"
	@echo "  backup             - Backup configuration and data"
	@echo ""
	@echo "Monitoring Targets:"
	@echo "  health-check       - Check service health"
	@echo "  metrics            - Show current metrics"
	@echo "  logs               - Show recent logs"
	@echo "  monitor            - Monitor resource usage"
	@echo "  profile            - Run performance profiling"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  GIT_COMMIT=$(GIT_COMMIT)"
	@echo "  BUILD_DATE=$(BUILD_DATE)"
	@echo "  DOCKER_IMAGE=$(DOCKER_IMAGE)"

# Development workflow targets
dev-setup: deps fmt lint test build
	@echo "Development environment setup completed"

ci: clean deps fmt lint security test build
	@echo "CI pipeline completed"

# Production readiness check
production-check: ci audit health-check
	@echo "Production readiness check completed"