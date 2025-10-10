# atest-ext-ai Quick Start Guide

## Prerequisites

- Go 1.23+ installed
- Ollama running on `http://localhost:11434` (or configure your own endpoint)
- At least one Ollama model installed (e.g., `ollama pull qwen2.5-coder:latest`)

## Quick Start (3 Steps)

### 1. Build the Plugin

```bash
# Build the plugin binary
task build

# Or manually:
go build -o bin/atest-ext-ai ./cmd/atest-ext-ai
```

### 2. Install Locally

```bash
# Install to local atest directory
task install-local

# This copies the binary to: ~/.config/atest/bin/atest-ext-ai
```

### 3. Start the Plugin

```bash
# Start the plugin (will use config.yaml or defaults)
./bin/atest-ext-ai

# Or run from installed location:
~/.config/atest/bin/atest-ext-ai
```

The plugin will:
- ✅ Create Unix socket at `/tmp/atest-ext-ai.sock`
- ✅ Load configuration from `config.yaml` (or use defaults)
- ✅ Connect to Ollama at `http://localhost:11434`
- ✅ Use model `qwen2.5-coder:latest` (or configured model)
- ✅ Ready to accept gRPC connections from main atest project

## Configuration

### Option 1: Use Provided config.yaml (Recommended)

The project includes a minimal `config.yaml` with working defaults:

```yaml
ai:
  default_service: ollama
  services:
    ollama:
      enabled: true
      endpoint: http://localhost:11434
      model: qwen2.5-coder:latest
```

### Option 2: Environment Variables

```bash
export OLLAMA_ENDPOINT=http://localhost:11434
export AI_MODEL=qwen2.5-coder:latest
./bin/atest-ext-ai
```

### Option 3: No Configuration (Use Defaults)

The plugin works out-of-the-box with built-in defaults:
- Endpoint: `http://localhost:11434`
- Model: `qwen2.5-coder:latest`

## Verify It's Working

### Check Socket File

```bash
ls -la /tmp/atest-ext-ai.sock
# Should show: srw-rw---- ... /tmp/atest-ext-ai.sock
```

### Check Logs

```bash
# The plugin logs should show:
✅ "AI plugin service initialized successfully"
✅ "Plugin ready to accept gRPC connections"
✅ "AI Plugin listening on Unix socket: /tmp/atest-ext-ai.sock"
```

### Test with Ollama

```bash
# Verify Ollama is running and model is available
curl http://localhost:11434/api/tags

# Should show your installed models including qwen2.5-coder:latest
```

## Integration with Main atest Project

The main [api-testing](https://github.com/linuxsuren/api-testing) project will connect to this plugin via the Unix socket.

### Configuration in Main Project

The main project should configure the plugin path:

```yaml
# In main atest config
extensions:
  - name: ai
    enabled: true
    socket: unix:///tmp/atest-ext-ai.sock
```

### Verify Integration

When the main atest project starts, it should:
1. Detect the socket at `/tmp/atest-ext-ai.sock`
2. Connect via gRPC
3. Call `Verify()` to check plugin health
4. Load plugin UI components

## Troubleshooting

### Issue: "Failed to initialize AI plugin service"

**Cause**: Configuration validation failed or model not found

**Solutions**:
1. Check if Ollama is running: `curl http://localhost:11434/api/tags`
2. Check if model exists: `ollama list`
3. Pull the default model: `ollama pull qwen2.5-coder:latest`
4. Or configure a different model in `config.yaml`

### Issue: "Socket not found"

**Cause**: Plugin not started or socket path incorrect

**Solutions**:
1. Start the plugin: `./bin/atest-ext-ai`
2. Check socket exists: `ls -la /tmp/atest-ext-ai.sock`
3. Check permissions: Socket should be `srw-rw----`

### Issue: "Provider not supported"

**Cause**: Invalid provider name in configuration

**Solutions**:
- Valid providers: `ollama`, `openai`, `deepseek`, `local` (alias for ollama)
- Check `config.yaml` has correct provider name
- See `config.example.yaml` for examples

### Issue: "Connection refused"

**Cause**: Ollama not running or wrong endpoint

**Solutions**:
```bash
# Start Ollama
ollama serve

# Test endpoint
curl http://localhost:11434/api/tags

# Configure correct endpoint in config.yaml or:
export OLLAMA_ENDPOINT=http://your-ollama-host:11434
```

## Development Mode

For development with debug logging:

```bash
task dev

# Or manually:
AI_PROVIDER="ollama" \
OLLAMA_ENDPOINT="http://localhost:11434" \
AI_MODEL="qwen2.5-coder:latest" \
LOG_LEVEL="debug" \
./bin/atest-ext-ai
```

## Advanced Configuration

See `config.example.yaml` for full configuration options including:
- Multiple AI providers (OpenAI, DeepSeek, etc.)
- Retry and rate limiting
- Custom models and endpoints
- Advanced logging options

## Need Help?

- Check logs for detailed error messages
- Verify Ollama is running and accessible
- Ensure model is installed: `ollama list`
- Check socket permissions and path
- Review `config.example.yaml` for configuration options

## Next Steps

Once the plugin is running:
1. Start the main atest project
2. Access the AI features in the web UI
3. Try generating SQL from natural language
4. Explore AI-powered testing features
