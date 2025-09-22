# atest-ext-ai

AI plugin for API Testing Tool that converts natural language to SQL queries.

## Quick Start

**Start with Docker:**
```bash
docker-compose up -d
```

**Build and run locally:**
```bash
make build
./bin/atest-ext-ai
```

**Test the plugin:**
```bash
# Check if plugin is running
ls -la /tmp/atest-ext-ai.sock

# Use with main project
atest run --stores-config stores.yaml your-test.yaml
```

## Configuration

Configure in `stores.yaml`:
```yaml
stores:
  - name: "ai-assistant"
    type: "ai"
    url: "unix:///tmp/atest-ext-ai.sock"
```

## Development

```bash
make dev        # Start development environment
make test       # Run tests
make build      # Build binary
```

---