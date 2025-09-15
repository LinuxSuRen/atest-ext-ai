# atest-ext-ai

AI Extension Plugin for API Testing Tool - Intelligent SQL Generation and Execution

## 🚀 Overview

This plugin extends the [API Testing Tool](https://github.com/linuxsuren/api-testing) with AI-powered SQL generation capabilities. It transforms natural language descriptions into executable SQL queries, supporting MySQL, PostgreSQL, and SQLite databases.

## ✨ Features

- **Natural Language to SQL**: Convert plain English to SQL queries
- **Multi-Database Support**: MySQL, PostgreSQL, SQLite
- **Local & Cloud AI**: Support for local models (Ollama) and online services (OpenAI, Claude)
- **Seamless Integration**: Native gRPC plugin architecture
- **Health Monitoring**: Real-time plugin status and health checks
- **High Performance**: Optimized for concurrent requests and low latency

## 🏗️ Architecture

The plugin implements the main project's `Loader` gRPC service interface:

```
Main API Testing System
        │
        ├─── HTTP API (/api/v1/ai/*)
        │
        └─── gRPC Bridge
                 │
           Unix Socket Communication
                 │
         atest-store-ai Plugin
                 │
        ├─── AI Engine (Ollama/OpenAI/Claude)
        └─── SQL Generation & Execution
```

## 📦 Installation

### Prerequisites

- Go 1.22+
- [API Testing Tool](https://github.com/linuxsuren/api-testing)
- For local AI: [Ollama](https://ollama.ai/) with a compatible model

### Build from Source

```bash
# Clone the repository
git clone https://github.com/linuxsuren/atest-ext-ai.git
cd atest-ext-ai

# Build the plugin
make build

# Install globally (optional)
make install
```

### Using Docker

```bash
# Build Docker image
make docker-build

# Run with Docker
make docker-run
```

## ⚙️ Configuration

### Environment Variables

```bash
export AI_PROVIDER="local"                          # local, openai, claude
export OLLAMA_ENDPOINT="http://localhost:11434"     # For local provider
export AI_MODEL="codellama"                         # Model name
export AI_API_KEY="your-api-key"                   # For online providers
export AI_PLUGIN_SOCKET_PATH="/tmp/atest-store-ai.sock"
```

### Configuration File

Create `config.yaml`:

```yaml
ai:
  provider: local
  ollama_endpoint: http://localhost:11434
  model: codellama
  confidence_threshold: 0.7
  enable_sql_execution: true
  supported_databases:
    - mysql
    - postgresql
    - sqlite
```

### Main Project Integration

Add to your `stores.yaml`:

```yaml
stores:
  - name: "ai-assistant"
    type: "ai"
    url: "unix:///tmp/atest-store-ai.sock"
    properties:
      - key: ai_provider
        value: local
      - key: ollama_endpoint
        value: http://localhost:11434
      - key: model
        value: codellama
      - key: confidence_threshold
        value: "0.7"
      - key: enable_sql_execution
        value: "true"
```

## 🚀 Usage

### Start the Plugin

```bash
# Development mode
make dev

# Production mode
./bin/atest-store-ai
```

### API Examples

Once integrated with the main API testing system:

```bash
# Generate SQL from natural language
curl -X POST http://localhost:8080/api/v1/data/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "natural_language": "Find all active users who registered last month",
    "database_type": "mysql"
  }'
```

Response:
```json
{
  "data": [
    {
      "key": "generated_sql",
      "value": "SELECT * FROM users WHERE status = 'active' AND created_at >= DATE_SUB(NOW(), INTERVAL 1 MONTH)"
    },
    {
      "key": "explanation", 
      "value": "This query finds all users with active status who were created in the last month"
    },
    {
      "key": "confidence_score",
      "value": "0.92"
    }
  ],
  "ai_info": {
    "processing_time_ms": 1200,
    "model_used": "codellama",
    "confidence_score": 0.92
  }
}
```

## 🧪 Development

### Development Setup

```bash
# Install dependencies
make deps

# Run tests
make test

# Run with live reload
make dev
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests  
make test-integration

# Benchmark tests
make benchmark
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Security check
make security
```

## 🏗️ Project Structure

```
atest-ext-ai/
├── cmd/
│   └── atest-store-ai/          # Main plugin entry point
├── pkg/
│   ├── ai/                      # AI engine implementations
│   ├── config/                  # Configuration management
│   └── plugin/                  # gRPC service implementation
├── config/                      # Configuration templates
├── docs/                        # Documentation
├── test/                        # Test files
└── scripts/                     # Build and deployment scripts
```

## 🤝 Integration with Main Project

This plugin integrates with the main API testing system via:

1. **gRPC Protocol**: Implements the `Loader` service from `pkg/testing/remote/loader.proto`
2. **Unix Socket**: Communicates via `/tmp/atest-store-ai.sock`
3. **Health Monitoring**: Automatic registration and health checks
4. **Configuration**: Follows `stores.yaml` format for seamless setup

## 🔧 Supported AI Providers

### Local (Ollama)
- **Models**: CodeLlama, Mistral, Llama2, etc.
- **Pros**: No API costs, privacy, offline capability
- **Cons**: Requires local setup and resources

### OpenAI
- **Models**: GPT-4, GPT-3.5-turbo
- **Pros**: High accuracy, fast response
- **Cons**: API costs, requires internet

### Claude (Anthropic)
- **Models**: Claude-3, Claude-2
- **Pros**: Good reasoning, safety-focused
- **Cons**: API costs, requires internet

## 📊 Performance

- **Response Time**: < 2s for simple queries, < 10s for complex queries
- **Concurrent Requests**: Up to 10 simultaneous AI requests
- **Memory Usage**: < 100MB baseline, < 500MB under load
- **SQL Accuracy**: > 85% for common query patterns

## 🛡️ Security

- **Input Validation**: Prevents SQL injection and prompt injection
- **Resource Limits**: Memory and processing time limits
- **Secure Communication**: Unix socket with file permissions
- **Audit Logging**: Complete request/response logging

## 🐳 Deployment

### Docker Compose

```yaml
version: '3.8'
services:
  atest-ai-plugin:
    build: .
    environment:
      - AI_PROVIDER=local
      - OLLAMA_ENDPOINT=http://ollama:11434
      - AI_MODEL=codellama
    volumes:
      - /tmp:/tmp
    depends_on:
      - ollama
  
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atest-ai-plugin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: atest-ai-plugin
  template:
    metadata:
      labels:
        app: atest-ai-plugin
    spec:
      containers:
      - name: atest-ai-plugin
        image: atest-ext-ai:latest
        env:
        - name: AI_PROVIDER
          value: "local"
        - name: OLLAMA_ENDPOINT
          value: "http://ollama-service:11434"
        volumeMounts:
        - name: plugin-sockets
          mountPath: /tmp
      volumes:
      - name: plugin-sockets
        emptyDir: {}
```

## 📝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [API Testing Tool](https://github.com/linuxsuren/api-testing) for the excellent plugin architecture
- [Ollama](https://ollama.ai/) for local AI model support
- The open source community for inspiration and contributions