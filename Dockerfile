# Build stage
# syntax=docker/dockerfile:1.7
ARG BUILDPLATFORM
ARG TARGETPLATFORM

FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder
WORKDIR /workspace

# Copy repository contents (needed because frontend build emits to ../pkg/plugin/assets)
COPY . .

# Cache npm modules to speed up rebuilds
RUN --mount=type=cache,target=/root/.npm npm --prefix frontend ci
RUN npm --prefix frontend run build

FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Enable module caching
ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/root/.cache/go-build

# Set working directory
WORKDIR /workspace

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Copy pre-built frontend assets
COPY --from=frontend-builder /workspace/pkg/plugin/assets ./pkg/plugin/assets

# Build the binary for the target platform
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o bin/atest-ext-ai ./cmd/atest-ext-ai

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
