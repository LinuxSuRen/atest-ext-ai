# atest-ext-ai

AI plugin for API Testing Tool that converts natural language to SQL queries.

## Configuration

The plugin uses a YAML configuration file. Copy `config.example.yaml` to `config.yaml` and customize:

```bash
cp config.example.yaml config.yaml
```

### Quick Start Configuration

For local development with Ollama:

```yaml
ai:
  default_service: ollama
  services:
    ollama:
      enabled: true
      endpoint: http://localhost:11434
      model: qwen2.5-coder:7b
```

### Environment Variables

You can override configuration with environment variables:

```bash
# AI Provider
export ATEST_EXT_AI_AI_PROVIDER=ollama
export ATEST_EXT_AI_OLLAMA_ENDPOINT=http://localhost:11434
export ATEST_EXT_AI_AI_MODEL=qwen2.5-coder:7b

# For OpenAI
export ATEST_EXT_AI_OPENAI_API_KEY=sk-...
export ATEST_EXT_AI_OPENAI_MODEL=gpt-4

# Logging
export ATEST_EXT_AI_LOG_LEVEL=debug
```

See `config.example.yaml` for all available options.

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
