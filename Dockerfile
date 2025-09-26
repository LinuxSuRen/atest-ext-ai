# Multi-stage build for atest-ext-ai plugin
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o bin/atest-ext-ai ./cmd/atest-ext-ai

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -s /bin/sh aiuser

# Create directories
RUN mkdir -p /tmp /etc/atest-ai
RUN chown aiuser:aiuser /etc/atest-ai

# Copy binary from builder stage
COPY --from=builder /app/bin/atest-ext-ai /usr/local/bin/atest-ext-ai
RUN chmod +x /usr/local/bin/atest-ext-ai

# Copy configuration template (create empty if not exists)
RUN touch /etc/atest-ai/config.yaml.example && chown aiuser:aiuser /etc/atest-ai/config.yaml.example

# Switch to non-root user
USER aiuser

# Environment variables
ENV AI_PLUGIN_SOCKET_PATH="/tmp/atest-ext-ai.sock"
ENV AI_PROVIDER="local"
ENV OLLAMA_ENDPOINT="${OLLAMA_ENDPOINT:-http://ollama:11434}"
# AI_MODEL will be auto-detected from available models

# Expose Unix socket directory
VOLUME ["/tmp"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD test -S ${AI_PLUGIN_SOCKET_PATH} || exit 1

# Run the plugin
ENTRYPOINT ["/usr/local/bin/atest-ext-ai"]

# Metadata
LABEL org.opencontainers.image.title="atest-ext-ai"
LABEL org.opencontainers.image.description="AI Extension Plugin for API Testing Tool"
LABEL org.opencontainers.image.vendor="API Testing Authors"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/linuxsuren/atest-ext-ai"