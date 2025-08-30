# AI Plugin Makefile

.PHONY: all build build-frontend build-backend clean dev test install-deps help

# Default target
all: build

# Build everything
build: build-frontend build-backend

# Build frontend (Vue.js) and copy assets
build-frontend:
	@echo "Building frontend..."
	cd ui && npm run build
	@echo "Copying frontend assets to Go embed directory..."
	mkdir -p internal/server/assets
	cp ui/dist/index.html internal/server/assets/
	@# Copy main JS file (find the index-*.js file)
	find ui/dist/assets -name "index-*.js" -exec cp {} internal/server/assets/index.js \;
	@# Copy main CSS file (find the index-*.css file)
	find ui/dist/assets -name "index-*.css" -exec cp {} internal/server/assets/style.css \;
	@echo "Frontend build and asset copy complete"

# Build backend (Go) with embedded assets
build-backend: build-frontend
	@echo "Building backend with embedded assets..."
	go build -tags embed -o bin/ai-plugin-server ./cmd/server
	@echo "Backend build complete"

# Development mode - start both frontend and backend
dev:
	@echo "Starting development servers..."
	@echo "Frontend will be available at http://localhost:5173"
	@echo "Backend will be available at http://localhost:8080"
	make dev-frontend & make dev-backend

# Start frontend development server
dev-frontend:
	cd ui && npm run dev

# Start backend development server
dev-backend:
	./bin/ai-plugin-server

# Install dependencies
install-deps:
	@echo "Installing Go dependencies..."
	go mod tidy
	@echo "Installing frontend dependencies..."
	cd ui && npm install

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf ui/dist/
	rm -rf ui/node_modules/.vite/

# Run tests
test:
	@echo "Running Go tests..."
	go test ./...
	@echo "Running frontend tests..."
	cd ui && npm run test

# Format code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Formatting frontend code..."
	cd ui && npm run format

# Lint code
lint:
	@echo "Linting Go code..."
	golangci-lint run
	@echo "Linting frontend code..."
	cd ui && npm run lint

# Build production version
build-prod: install-deps build-backend
	@echo "Production build complete"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Help
help:
	@echo "Available targets:"
	@echo "  all           - Build everything (default)"
	@echo "  build         - Build frontend and backend"
	@echo "  build-frontend - Build frontend only"
	@echo "  build-backend  - Build backend only"
	@echo "  build-prod    - Build production binary with embedded assets"
	@echo "  dev           - Start development servers"
	@echo "  dev-frontend  - Start frontend development server"
	@echo "  dev-backend   - Start backend development server"
	@echo "  install-deps  - Install all dependencies"
	@echo "  install-tools - Install development tools"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  help          - Show this help"