# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy all source code first (needed for go.mod replace directive)
COPY . .

# Download dependencies (replace directive requires source to be present)
RUN go mod download

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

# Copy config.yaml for default configuration
COPY --from=builder /app/config.yaml /app/config.yaml

# Change ownership to appuser
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Set the entrypoint
ENTRYPOINT ["/app/atest-ext-ai"]
