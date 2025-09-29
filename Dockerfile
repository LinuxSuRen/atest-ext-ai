# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o bin/atest-ext-ai \
    ./cmd/atest-ext-ai

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy the binary
COPY --from=builder /app/bin/atest-ext-ai /app/atest-ext-ai

# Change ownership to appuser
RUN chown appuser:appuser /app/atest-ext-ai

# Switch to non-root user
USER appuser

# Set the entrypoint
ENTRYPOINT ["/app/atest-ext-ai"]