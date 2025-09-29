# atest-ext-ai

AI plugin for API Testing Tool that converts natural language to SQL queries.

## Development

### Local Development

```bash
# Build and install plugin for local development (single command)
make install-local

# The plugin binary will be installed to ~/.config/atest/bin/
# and automatically discovered by the API Testing Server
# This command will replace any existing binary
```

### Release Process

```bash
# Build and push OCI image to GitHub Container Registry (single command)
make docker-release-github

# Or push to custom registry
make docker-release DOCKER_REGISTRY=your-registry.com

# This will:
# 1. Build the Docker image
# 2. Tag it with version and latest
# 3. Push to the specified registry (default: ghcr.io/linuxsuren)
```

### Available Commands

- `make install-local` - Build and install for local development (replaces existing)
- `make docker-release-github` - Build and push OCI image to GitHub Container Registry
- `make docker-release` - Build and push OCI image to custom registry
- `make test` - Run tests
- `make clean` - Clean build artifacts
- `make help` - Show all available commands
