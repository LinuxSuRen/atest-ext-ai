.SHELLFLAGS := -o pipefail -c
SHELL := /bin/bash

.DEFAULT_GOAL := default

BINARY_NAME ?= atest-ext-ai
BUILD_DIR ?= bin
MAIN_PACKAGE ?= ./cmd/atest-ext-ai
DOCKER_IMAGE ?= atest-ext-ai
DOCKER_REGISTRY ?= ghcr.io/linuxsuren
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -s -w -X main.version=$(VERSION)
BUILD_BIN := $(BUILD_DIR)/$(BINARY_NAME)

.PHONY: default build-frontend build test test-watch deps clean install install-local dev fmt lint lint-check vet verify check benchmark docker-build docker-release docker-release-github coverage integration-test help

default: clean build test ## Clean, build and test

build-frontend: ## Build frontend assets (Vue 3 + TypeScript)
	@[ -d frontend/node_modules ] || (cd frontend && npm install)
	cd frontend && npm run build

build: $(BUILD_BIN) ## Build the plugin binary

$(BUILD_BIN):
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_BIN) $(MAIN_PACKAGE)

test: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

test-watch: ## Run tests in watch mode
	while true; do \
	  go test -v ./...; \
	  inotifywait -r -e modify,create,delete --exclude '\.git' .; \
	done

deps: ## Install and verify dependencies
	go mod tidy
	go mod download
	go mod verify

clean: ## Clean build artifacts and caches
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
	go clean -cache -testcache -modcache

install: build ## Install to system location (/usr/local/bin)
	sudo cp $(BUILD_BIN) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)

install-local: build-frontend build ## Install to local development directory (~/.config/atest/bin)
	mkdir -p ~/.config/atest/bin/
	cp $(BUILD_BIN) ~/.config/atest/bin/
	chmod +x ~/.config/atest/bin/$(BINARY_NAME)
	@echo "Local installation completed - ~/.config/atest/bin/$(BINARY_NAME)"

dev: ## Run in development mode with debug logging
	LOG_LEVEL=debug go run $(MAIN_PACKAGE)

fmt: ## Format Go code
	go fmt ./...
	gofmt -s -w .

lint: ## Run golangci-lint
	golangci-lint run --fix

lint-check: ## Run golangci-lint without fixes
	golangci-lint run

vet: ## Run go vet
	go vet ./...

verify: ## Format, lint, vet and test Go code
	$(MAKE) fmt
	$(MAKE) lint-check
	$(MAKE) vet
	$(MAKE) test

check: ## Run all checks (fmt, vet, lint, test)
	$(MAKE) fmt
	$(MAKE) vet
	$(MAKE) lint-check
	$(MAKE) test

benchmark: ## Run benchmark tests
	go test -bench=. -benchmem -run=^$$ ./...

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):latest .

docker-release: docker-build ## Build and push Docker image to registry
	docker tag $(DOCKER_IMAGE):latest $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)
	docker tag $(DOCKER_IMAGE):latest $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	@echo "Pushed to $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(VERSION)"

docker-release-github: ## Quick release to GitHub Container Registry
	$(MAKE) docker-release DOCKER_REGISTRY=ghcr.io/linuxsuren

coverage: ## Generate and view coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated - coverage.html"

integration-test: install-local ## Run integration tests with plugin
	@echo "Starting plugin..."
	@~/.config/atest/bin/$(BINARY_NAME) & \
	PLUGIN_PID=$$!; \
	sleep 2; \
	if [ -S /tmp/atest-ext-ai.sock ]; then \
	  echo "✅ Socket created successfully"; \
	else \
	  echo "❌ Socket not found"; \
	  kill $$PLUGIN_PID 2>/dev/null; \
	  exit 1; \
	fi; \
	kill $$PLUGIN_PID; \
	rm -f /tmp/atest-ext-ai.sock

help: ## Show available targets
	@printf "Available targets:\n"
	@awk -F':.*## ' '/^[a-zA-Z0-9_.-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
