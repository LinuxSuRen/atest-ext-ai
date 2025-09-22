# Quick Start

## Prerequisites
- Go 1.22+
- Docker (optional, for easy setup)

## Start with Docker
```bash
git clone https://github.com/linuxsuren/atest-ext-ai.git
cd atest-ext-ai
docker-compose up -d
```

## Build and Run
```bash
# Build
make build

# Run
./bin/atest-ext-ai
```

## Test
```bash
# Check plugin is running
ls -la /tmp/atest-ext-ai.sock

# Run tests
make test
```

## Configuration

Create `stores.yaml` in your main project:
```yaml
stores:
  - name: "ai-assistant"
    type: "ai"
    url: "unix:///tmp/atest-ext-ai.sock"
```

That's it! The plugin is ready to convert natural language to SQL queries.