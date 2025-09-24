# Ollama Integration Demo

This is a demonstration of how to use Ollama with the atest-ext-ai plugin for natural language to SQL conversion.

## Prerequisites

1. Install and run Ollama server
2. Download a compatible model (e.g., llama3.2:1b)
3. Ensure database access is available

## Environment Variables

Before running this demo, set the following environment variables:

```bash
# Required
export DB_PASSWORD="your_database_password"

# Optional (with defaults)
export OLLAMA_SERVER_URL="http://localhost:11434"  # Default
export OLLAMA_TEST_MODEL="llama3.2:1b"             # Default
export DB_HOST="localhost:3306"                     # Default
export DB_USER="root"                               # Default
export DB_NAME="test"                               # Default
export GRPC_URL="127.0.0.1:7071"                   # Default
```

## Usage

```bash
cd examples/ollama-demo
go run main.go grpc.go [-v]
```

The `-v` flag enables verbose logging.

## Security Note

This demo uses environment variables for sensitive configuration like database passwords. Never hardcode credentials in source code.