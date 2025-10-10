# Configuration Migration Guide

## Overview

This guide explains how to migrate from the legacy configuration format to the new v1 configuration format. The new format provides better structure, improved validation, and enhanced features.

## Why Migrate?

The legacy configuration format will be deprecated in v2.0.0. The new v1 format offers:

- **Better Organization**: Separate sections for server, plugin, AI, database, and logging
- **Enhanced Validation**: Automatic validation with clear error messages
- **Environment Variable Support**: Comprehensive environment variable bindings
- **Version Tracking**: Built-in version field for future migrations
- **Advanced Features**: Support for rate limiting, circuit breakers, and caching
- **Improved Documentation**: Better structured and more maintainable

## Format Comparison

### Legacy Format

```yaml
ai:
  provider: ollama
  endpoint: http://localhost:11434
  model: qwen2.5-coder:7b
  timeout: 60s
  api_key: ""
```

### New V1 Format

```yaml
version: v1

server:
  host: 0.0.0.0
  port: 8080
  socket_path: /tmp/atest-ext-ai.sock
  timeout: 30s
  read_timeout: 15s
  write_timeout: 15s
  max_connections: 100

plugin:
  name: atest-ext-ai
  version: 1.0.0
  debug: false
  log_level: info
  environment: production

ai:
  default_service: ollama
  timeout: 60s
  fallback_order:
    - ollama

  # Multiple AI service configurations
  services:
    ollama:
      enabled: true
      provider: ollama
      endpoint: http://localhost:11434
      model: qwen2.5-coder:7b
      max_tokens: 4096
      priority: 1
      timeout: 60s

    openai:
      enabled: false
      provider: openai
      api_key: ${OPENAI_API_KEY}
      model: gpt-4
      endpoint: https://api.openai.com/v1
      max_tokens: 8192
      priority: 2
      timeout: 60s

  # Advanced features
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    window_size: 1m

  circuit_breaker:
    enabled: true
    failure_threshold: 5
    success_threshold: 3
    timeout: 30s
    reset_timeout: 60s

  retry:
    enabled: true
    max_attempts: 3
    initial_delay: 1s
    max_delay: 30s
    multiplier: 2.0
    jitter: true

  cache:
    enabled: true
    ttl: 1h
    max_size: 1000
    provider: memory

database:
  enabled: false
  driver: sqlite
  dsn: "file:atest-ext-ai.db?cache=shared&mode=rwc"
  default_type: mysql
  max_connections: 10
  max_idle: 5
  max_lifetime: 1h

logging:
  level: info
  format: json
  output: stdout
  file:
    path: /var/log/atest-ext-ai.log
    max_size: 100MB
    max_backups: 3
    max_age: 28
    compress: true
  rotation:
    enabled: true
    size: 100MB
    count: 5
    age: 30d
    compress: true
```

## Migration Methods

### Method 1: Automatic Migration (Recommended)

Use the built-in migration tool to automatically convert your configuration:

```bash
# Backup your current configuration
cp config.yaml config.yaml.backup

# Run the migration tool (when implemented)
atest-ext-ai migrate-config --input config.yaml --output config.new.yaml

# Review the new configuration
cat config.new.yaml

# Test with the new configuration
atest-ext-ai --config config.new.yaml

# If everything works, replace the old config
mv config.new.yaml config.yaml
```

### Method 2: Manual Migration

If you prefer to migrate manually:

1. **Create a new configuration file** starting with the v1 template
2. **Map legacy values** to their new locations:
   - `ai.provider` → `ai.default_service`
   - `ai.endpoint` → `ai.services.<provider>.endpoint`
   - `ai.model` → `ai.services.<provider>.model`
   - `ai.api_key` → `ai.services.<provider>.api_key`
   - `ai.timeout` → `ai.timeout` and `ai.services.<provider>.timeout`

3. **Add required fields**:
   - `version: v1` at the top of the file
   - `plugin.name`, `plugin.version`
   - `server` section with defaults

4. **Configure advanced features** (optional):
   - Enable rate limiting if needed
   - Configure circuit breaker for resilience
   - Set up caching for performance

5. **Validate the configuration**:
   ```bash
   atest-ext-ai validate-config config.yaml
   ```

## Field Mapping Reference

| Legacy Field | New Field | Notes |
|-------------|-----------|-------|
| `ai.provider` | `ai.default_service` | Maps to service name |
| `ai.endpoint` | `ai.services.<name>.endpoint` | Now per-service |
| `ai.model` | `ai.services.<name>.model` | Now per-service |
| `ai.api_key` | `ai.services.<name>.api_key` | Now per-service |
| `ai.timeout` | `ai.timeout` | Global timeout |
| - | `ai.services.<name>.timeout` | Service-specific timeout |
| - | `ai.fallback_order` | New: service fallback |
| - | `version` | **Required**: must be "v1" |
| - | `server.*` | New: server configuration |
| - | `plugin.*` | New: plugin metadata |

## Environment Variables

The new format supports comprehensive environment variable overrides:

```bash
# Server configuration
export ATEST_EXT_AI_SERVER_HOST=localhost
export ATEST_EXT_AI_SERVER_PORT=8080
export ATEST_EXT_AI_SERVER_SOCKET_PATH=/tmp/atest-ext-ai.sock

# Plugin configuration
export ATEST_EXT_AI_DEBUG=true
export ATEST_EXT_AI_LOG_LEVEL=debug
export ATEST_EXT_AI_ENVIRONMENT=development

# AI provider
export ATEST_EXT_AI_AI_PROVIDER=openai
export ATEST_EXT_AI_AI_TIMEOUT=60s

# AI service configuration
export ATEST_EXT_AI_OLLAMA_ENDPOINT=http://localhost:11434
export ATEST_EXT_AI_AI_MODEL=qwen2.5-coder:7b
export ATEST_EXT_AI_OPENAI_API_KEY=sk-...
export ATEST_EXT_AI_OPENAI_MODEL=gpt-4

# Database
export ATEST_EXT_AI_DATABASE_ENABLED=true
export ATEST_EXT_AI_DATABASE_DRIVER=postgres
export ATEST_EXT_AI_DATABASE_DSN="host=localhost user=postgres password=secret"

# Logging
export ATEST_EXT_AI_LOG_FORMAT=json
export ATEST_EXT_AI_LOG_OUTPUT=stdout
```

## Migration Examples

### Example 1: Simple Ollama Configuration

**Legacy:**
```yaml
ai:
  provider: ollama
  endpoint: http://localhost:11434
  model: qwen2.5-coder:7b
```

**New V1:**
```yaml
version: v1

plugin:
  name: atest-ext-ai
  version: 1.0.0

ai:
  default_service: ollama
  services:
    ollama:
      enabled: true
      provider: ollama
      endpoint: http://localhost:11434
      model: qwen2.5-coder:7b
```

### Example 2: OpenAI Configuration

**Legacy:**
```yaml
ai:
  provider: openai
  api_key: sk-...
  model: gpt-4
```

**New V1:**
```yaml
version: v1

plugin:
  name: atest-ext-ai
  version: 1.0.0

ai:
  default_service: openai
  services:
    openai:
      enabled: true
      provider: openai
      api_key: ${OPENAI_API_KEY}  # Better: use environment variable
      model: gpt-4
      endpoint: https://api.openai.com/v1
```

### Example 3: Multi-Provider Setup with Fallback

**New V1 Only** (not possible with legacy format):
```yaml
version: v1

plugin:
  name: atest-ext-ai
  version: 1.0.0

ai:
  default_service: ollama
  timeout: 60s
  fallback_order:
    - ollama
    - openai

  services:
    ollama:
      enabled: true
      provider: ollama
      endpoint: http://localhost:11434
      model: qwen2.5-coder:7b
      priority: 1

    openai:
      enabled: true
      provider: openai
      api_key: ${OPENAI_API_KEY}
      model: gpt-4
      priority: 2
```

## Common Issues and Solutions

### Issue 1: "Unable to detect config version"

**Cause**: Configuration file is corrupted or has invalid YAML syntax.

**Solution**: Validate your YAML syntax using a YAML validator or `yamllint`:
```bash
yamllint config.yaml
```

### Issue 2: "Validation failed for field 'ai.services'"

**Cause**: Missing required fields in service configuration.

**Solution**: Ensure each enabled service has:
- `provider` field
- `endpoint` (for remote providers)
- `model` field

### Issue 3: Environment variables not working

**Cause**: Incorrect environment variable naming.

**Solution**: All environment variables must:
- Start with `ATEST_EXT_AI_` prefix
- Use underscores instead of dots
- Match the structure in the configuration

Example: `ai.services.ollama.endpoint` → `ATEST_EXT_AI_OLLAMA_ENDPOINT`

### Issue 4: Configuration not loading

**Cause**: File not in expected location.

**Solution**: Place configuration in one of:
- `./config.yaml` (current directory)
- `/etc/atest-ext-ai/config.yaml` (system-wide)
- Specify with `--config` flag: `atest-ext-ai --config /path/to/config.yaml`

## Best Practices

1. **Use Environment Variables for Secrets**
   ```yaml
   api_key: ${OPENAI_API_KEY}  # Good
   api_key: sk-actual-key      # Bad: hardcoded secret
   ```

2. **Enable Advanced Features for Production**
   ```yaml
   ai:
     rate_limit:
       enabled: true
     circuit_breaker:
       enabled: true
     retry:
       enabled: true
     cache:
       enabled: true
   ```

3. **Set Appropriate Timeouts**
   - Use shorter timeouts for local services (Ollama)
   - Use longer timeouts for remote APIs (OpenAI)

4. **Configure Fallback Services**
   ```yaml
   ai:
     fallback_order:
       - ollama      # Primary: fast, local
       - openai      # Fallback: cloud, reliable
   ```

5. **Use Structured Logging in Production**
   ```yaml
   logging:
     level: info        # Production
     format: json       # Structured, parseable
     output: stdout     # For container environments
   ```

## Validation

After migration, validate your configuration:

```bash
# Check configuration syntax and structure
atest-ext-ai validate-config config.yaml

# Test connection to AI services
atest-ext-ai test-connection

# Start with debug logging to verify settings
ATEST_EXT_AI_LOG_LEVEL=debug atest-ext-ai
```

## Rollback

If you need to rollback to the legacy format:

```bash
# Restore from backup
cp config.yaml.backup config.yaml

# Restart the service
systemctl restart atest-ext-ai
```

## Getting Help

If you encounter issues during migration:

1. **Check the logs**: `journalctl -u atest-ext-ai -f`
2. **Enable debug mode**: Set `ATEST_EXT_AI_LOG_LEVEL=debug`
3. **Validate configuration**: Run `atest-ext-ai validate-config`
4. **Report issues**: [GitHub Issues](https://github.com/linuxsuren/atest-ext-ai/issues)

## Timeline

- **v1.x**: Legacy format supported with deprecation warnings
- **v2.0**: Legacy format removed, v1 format required

**We recommend migrating as soon as possible** to take advantage of new features and ensure compatibility with future releases.
