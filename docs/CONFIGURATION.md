# Configuration Reference Guide

## Overview

The atest-ext-ai plugin supports multiple configuration methods to accommodate different deployment scenarios. This guide covers all configuration options, their formats, precedence, and usage examples.

## Table of Contents

- [Configuration Methods](#configuration-methods)
- [Configuration Structure](#configuration-structure)
- [AI Provider Settings](#ai-provider-settings)
- [Database Configuration](#database-configuration)
- [Security Settings](#security-settings)
- [Performance Tuning](#performance-tuning)
- [Logging Configuration](#logging-configuration)
- [Environment Variables](#environment-variables)
- [Advanced Settings](#advanced-settings)
- [Examples](#examples)

## Configuration Methods

### Configuration Precedence

Configuration is loaded in the following order (highest to lowest precedence):

1. **Command-line flags**
2. **Environment variables**
3. **Configuration file**
4. **Default values**

### Configuration File

The plugin supports YAML, JSON, and TOML configuration files:

```bash
# Specify config file location
atest-ext-ai --config /path/to/config.yaml
atest-ext-ai --config /path/to/config.json
atest-ext-ai --config /path/to/config.toml
```

### Environment Variables

All configuration options can be set via environment variables using the prefix `ATEST_AI_`:

```bash
export ATEST_AI_PROVIDER=local
export ATEST_AI_OLLAMA_ENDPOINT=http://localhost:11434
export ATEST_AI_MODEL=codellama
```

### Command-line Flags

```bash
atest-ext-ai \
  --ai-provider local \
  --ollama-endpoint http://localhost:11434 \
  --ai-model codellama \
  --log-level info
```

## Configuration Structure

### Complete Configuration Schema

```yaml
# AI Provider Configuration
ai:
  provider: local                    # local, openai, claude
  model: codellama                  # AI model name
  confidence_threshold: 0.7         # Minimum confidence for SQL generation
  enable_sql_execution: false       # Allow SQL execution
  max_concurrent_requests: 10       # Maximum concurrent AI requests
  request_timeout: 30s              # Request timeout
  cache_ttl: 3600s                  # Cache time-to-live

  # Provider-specific settings
  local:
    ollama_endpoint: http://localhost:11434
    models_path: /models
    pull_missing_models: true

  openai:
    api_key: ${OPENAI_API_KEY}
    api_base: https://api.openai.com/v1
    organization: ${OPENAI_ORGANIZATION}
    max_tokens: 1000
    temperature: 0.3

  claude:
    api_key: ${CLAUDE_API_KEY}
    api_version: 2023-06-01
    max_tokens: 1000
    temperature: 0.3

# Plugin Settings
plugin:
  socket_path: /tmp/atest-ext-ai.sock
  socket_permissions: 0660
  health_check_interval: 30s
  graceful_shutdown_timeout: 30s
  max_request_size: 10MB
  enable_metrics: true
  metrics_port: 9090
  metrics_path: /metrics

# Database Support
databases:
  mysql:
    max_connections: 10
    connection_timeout: 5s
    query_timeout: 30s
    enable_ssl: true

  postgresql:
    max_connections: 10
    connection_timeout: 5s
    query_timeout: 30s
    enable_ssl: true

  sqlite:
    max_connections: 1
    connection_timeout: 5s
    query_timeout: 30s

# Security Configuration
security:
  enable_audit_log: true
  audit_log_file: /var/log/atest-ai/audit.log
  max_request_size: 10MB
  allowed_hosts: []
  blocked_hosts: []

  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 10
    cleanup_interval: 1m

  sql_validation:
    enabled: true
    blocked_keywords: ["DROP", "DELETE", "TRUNCATE", "ALTER"]
    max_query_length: 5000
    allow_multiple_statements: false

# Logging Configuration
logging:
  level: info                       # debug, info, warn, error
  format: json                      # json, text
  output: stdout                    # stdout, stderr, file
  file_path: /var/log/atest-ai/app.log
  max_size: 100MB
  max_files: 5
  max_age: 30
  compress: true

  # Component-specific logging
  components:
    ai: info
    plugin: info
    database: warn
    security: info

# Performance Settings
performance:
  worker_pool_size: 10
  queue_size: 100
  memory_limit: 1GB
  cpu_limit: 2
  gc_percent: 100

  cache:
    enabled: true
    type: memory                    # memory, redis, file
    size: 100MB
    ttl: 1h
    cleanup_interval: 10m

    redis:
      address: localhost:6379
      password: ${REDIS_PASSWORD}
      database: 0
      pool_size: 10

# Monitoring and Observability
monitoring:
  enable_tracing: false
  tracing_endpoint: http://jaeger:14268
  enable_profiling: false
  profiling_port: 6060

  health_checks:
    startup_timeout: 60s
    liveness_timeout: 10s
    readiness_timeout: 5s

  metrics:
    enabled: true
    port: 9090
    path: /metrics
    namespace: atest_ai
    subsystem: plugin

# Development Settings
development:
  debug: false
  hot_reload: false
  mock_ai_responses: false
  verbose_errors: false
  enable_cors: false
```

## AI Provider Settings

### Local Provider (Ollama)

```yaml
ai:
  provider: local
  model: codellama
  local:
    ollama_endpoint: http://localhost:11434
    models_path: /home/user/.ollama/models
    pull_missing_models: true
    connection_timeout: 30s
    request_timeout: 120s
    max_retries: 3
    retry_delay: 5s
```

#### Supported Local Models

| Model | Description | Size | Use Case |
|-------|-------------|------|----------|
| `codellama` | Code generation and completion | 7B-34B | SQL generation, code analysis |
| `mistral` | General-purpose language model | 7B | Natural language processing |
| `llama2` | Meta's language model | 7B-70B | General queries |
| `wizardcoder` | Code-focused model | 15B-34B | Complex SQL generation |
| `sqlcoder` | SQL-specific model | 7B-15B | SQL optimization |

### OpenAI Provider

```yaml
ai:
  provider: openai
  model: gpt-4
  openai:
    api_key: ${OPENAI_API_KEY}
    api_base: https://api.openai.com/v1
    organization: ${OPENAI_ORGANIZATION}
    max_tokens: 1000
    temperature: 0.3
    top_p: 1.0
    frequency_penalty: 0.0
    presence_penalty: 0.0
    request_timeout: 60s
    max_retries: 3
    retry_delay: 2s
```

#### Supported OpenAI Models

| Model | Description | Context Length | Cost Tier |
|-------|-------------|----------------|-----------|
| `gpt-4` | Most capable model | 8,192 | High |
| `gpt-4-32k` | Extended context | 32,768 | Very High |
| `gpt-3.5-turbo` | Fast and efficient | 4,096 | Low |
| `gpt-3.5-turbo-16k` | Extended context | 16,384 | Medium |

### Claude Provider (Anthropic)

```yaml
ai:
  provider: claude
  model: claude-3-sonnet
  claude:
    api_key: ${CLAUDE_API_KEY}
    api_version: 2023-06-01
    api_base: https://api.anthropic.com
    max_tokens: 1000
    temperature: 0.3
    top_p: 1.0
    request_timeout: 60s
    max_retries: 3
    retry_delay: 2s
```

#### Supported Claude Models

| Model | Description | Context Length | Performance |
|-------|-------------|----------------|-------------|
| `claude-3-opus` | Most capable | 200k | Highest |
| `claude-3-sonnet` | Balanced | 200k | High |
| `claude-3-haiku` | Fast and efficient | 200k | Medium |

## Database Configuration

### MySQL Configuration

```yaml
databases:
  mysql:
    max_connections: 10
    connection_timeout: 5s
    query_timeout: 30s
    enable_ssl: true
    ssl_mode: preferred              # disabled, preferred, required
    charset: utf8mb4
    collation: utf8mb4_unicode_ci

    # Connection pool settings
    pool:
      max_idle_connections: 5
      max_open_connections: 10
      connection_max_lifetime: 1h
      connection_max_idle_time: 10m

    # Query execution limits
    limits:
      max_query_length: 5000
      max_result_rows: 1000
      max_execution_time: 30s
```

### PostgreSQL Configuration

```yaml
databases:
  postgresql:
    max_connections: 10
    connection_timeout: 5s
    query_timeout: 30s
    enable_ssl: true
    ssl_mode: prefer                 # disable, allow, prefer, require

    # Connection pool settings
    pool:
      max_idle_connections: 5
      max_open_connections: 10
      connection_max_lifetime: 1h
      connection_max_idle_time: 10m

    # PostgreSQL-specific settings
    search_path: public
    application_name: atest-ai-plugin
    statement_timeout: 30s
```

### SQLite Configuration

```yaml
databases:
  sqlite:
    max_connections: 1               # SQLite only supports 1 writer
    connection_timeout: 5s
    query_timeout: 30s

    # SQLite-specific settings
    journal_mode: WAL               # DELETE, TRUNCATE, PERSIST, MEMORY, WAL, OFF
    synchronous: NORMAL             # OFF, NORMAL, FULL, EXTRA
    cache_size: 2000                # Number of pages
    busy_timeout: 5000              # Milliseconds

    # File settings
    auto_vacuum: INCREMENTAL        # NONE, FULL, INCREMENTAL
    temp_store: MEMORY              # DEFAULT, FILE, MEMORY
```

## Security Settings

### Authentication

```yaml
security:
  authentication:
    enabled: true
    type: jwt                       # none, basic, jwt, api_key
    jwt:
      secret: ${JWT_SECRET}
      algorithm: HS256              # HS256, RS256
      expiration: 24h
      issuer: atest-ai-plugin

    api_key:
      header_name: X-API-Key
      query_param: api_key
      keys:
        - key: ${API_KEY_1}
          name: client1
          permissions: ["read", "write"]
```

### Authorization

```yaml
security:
  authorization:
    enabled: true
    default_policy: deny

    policies:
      - name: admin
        effect: allow
        actions: ["*"]
        resources: ["*"]

      - name: readonly
        effect: allow
        actions: ["query:generate", "status:read"]
        resources: ["*"]

      - name: limited
        effect: allow
        actions: ["query:generate"]
        resources: ["database:mysql", "database:postgresql"]
        conditions:
          confidence_threshold: ">= 0.8"
```

### Input Validation

```yaml
security:
  validation:
    enabled: true

    natural_language:
      min_length: 10
      max_length: 2000
      blocked_patterns:
        - "(?i)password"
        - "(?i)secret"
        - "(?i)api[_-]?key"

    sql:
      max_length: 5000
      blocked_keywords:
        - DROP
        - DELETE
        - TRUNCATE
        - ALTER
        - CREATE
        - INSERT
        - UPDATE
      allowed_functions:
        - SELECT
        - COUNT
        - SUM
        - AVG
        - MAX
        - MIN
        - GROUP_CONCAT
```

## Performance Tuning

### Memory Settings

```yaml
performance:
  memory:
    limit: 1GB
    gc_percent: 100
    allocation_rate_limit: 100MB/s

  worker_pools:
    ai_requests:
      size: 10
      queue_size: 100
      timeout: 60s

    database_queries:
      size: 5
      queue_size: 50
      timeout: 30s
```

### Caching Configuration

```yaml
performance:
  cache:
    enabled: true
    type: redis                     # memory, redis, file

    memory:
      size: 100MB
      max_entries: 10000
      ttl: 1h
      cleanup_interval: 10m

    redis:
      address: redis:6379
      password: ${REDIS_PASSWORD}
      database: 1
      pool_size: 10
      max_retries: 3
      retry_delay: 1s

    file:
      directory: /var/cache/atest-ai
      max_size: 1GB
      cleanup_interval: 1h
```

### Connection Pooling

```yaml
performance:
  connection_pools:
    ai_providers:
      max_idle_connections: 5
      max_active_connections: 20
      connection_timeout: 30s
      idle_timeout: 5m

    databases:
      max_idle_connections: 2
      max_active_connections: 10
      connection_timeout: 5s
      idle_timeout: 10m
```

## Logging Configuration

### Log Levels and Output

```yaml
logging:
  level: info
  format: json
  output: stdout

  # File output
  file:
    path: /var/log/atest-ai/app.log
    max_size: 100MB
    max_files: 5
    max_age: 30d
    compress: true

  # Component-specific levels
  components:
    ai_provider: info
    sql_generator: debug
    database: warn
    security: info
    metrics: error

  # Structured logging fields
  fields:
    service: atest-ai-plugin
    version: ${VERSION}
    environment: ${ENVIRONMENT}
```

### Audit Logging

```yaml
logging:
  audit:
    enabled: true
    file: /var/log/atest-ai/audit.log
    format: json
    max_size: 50MB
    max_files: 10

    events:
      - sql_generation
      - sql_execution
      - authentication
      - authorization
      - configuration_change

    include_request_body: true
    include_response_body: false
    sensitive_fields:
      - api_key
      - password
      - token
```

## Environment Variables

### Complete Environment Variable List

```bash
# AI Configuration
ATEST_AI_PROVIDER=local
ATEST_AI_MODEL=codellama
ATEST_AI_CONFIDENCE_THRESHOLD=0.7
ATEST_AI_ENABLE_SQL_EXECUTION=false

# Local Provider (Ollama)
ATEST_AI_OLLAMA_ENDPOINT=http://localhost:11434
ATEST_AI_OLLAMA_MODELS_PATH=/models

# OpenAI Provider
ATEST_AI_OPENAI_API_KEY=sk-...
ATEST_AI_OPENAI_ORGANIZATION=org-...
ATEST_AI_OPENAI_MAX_TOKENS=1000
ATEST_AI_OPENAI_TEMPERATURE=0.3

# Claude Provider
ATEST_AI_CLAUDE_API_KEY=sk-...
ATEST_AI_CLAUDE_API_VERSION=2023-06-01
ATEST_AI_CLAUDE_MAX_TOKENS=1000

# Plugin Settings
ATEST_AI_SOCKET_PATH=/tmp/atest-ext-ai.sock
ATEST_AI_SOCKET_PERMISSIONS=0660
ATEST_AI_HEALTH_CHECK_INTERVAL=30s
ATEST_AI_METRICS_PORT=9090

# Database Settings
ATEST_AI_MYSQL_MAX_CONNECTIONS=10
ATEST_AI_POSTGRESQL_MAX_CONNECTIONS=10
ATEST_AI_SQLITE_MAX_CONNECTIONS=1

# Security Settings
ATEST_AI_ENABLE_AUDIT_LOG=true
ATEST_AI_RATE_LIMIT_ENABLED=true
ATEST_AI_RATE_LIMIT_RPM=60

# Logging Settings
ATEST_AI_LOG_LEVEL=info
ATEST_AI_LOG_FORMAT=json
ATEST_AI_LOG_OUTPUT=stdout
ATEST_AI_LOG_FILE_PATH=/var/log/atest-ai/app.log

# Performance Settings
ATEST_AI_WORKER_POOL_SIZE=10
ATEST_AI_MEMORY_LIMIT=1GB
ATEST_AI_CACHE_ENABLED=true
ATEST_AI_CACHE_SIZE=100MB

# Development Settings
ATEST_AI_DEBUG=false
ATEST_AI_HOT_RELOAD=false
ATEST_AI_MOCK_AI_RESPONSES=false
```

## Advanced Settings

### Custom AI Prompt Templates

```yaml
ai:
  prompt_templates:
    sql_generation: |
      You are an expert SQL developer. Generate a SQL query for the following request:

      Natural Language: {natural_language}
      Database Type: {database_type}
      Schema Context: {schema_context}

      Requirements:
      - Generate syntactically correct {database_type} SQL
      - Include comments explaining complex parts
      - Optimize for performance
      - Use appropriate indexes

      Response format: JSON with fields 'sql', 'explanation', 'confidence'

    sql_optimization: |
      Optimize the following SQL query for {database_type}:

      Original Query: {original_sql}
      Schema Context: {schema_context}

      Provide optimized query with explanation of changes.
```

### Plugin Extensions

```yaml
plugins:
  extensions:
    enabled: true
    directory: /etc/atest-ai/extensions

    available:
      - name: schema_analyzer
        enabled: true
        config:
          cache_schema_info: true
          auto_refresh_interval: 1h

      - name: query_explainer
        enabled: true
        config:
          include_execution_plan: true
          estimate_cost: true
```

### Integration Settings

```yaml
integrations:
  main_api_tool:
    address: unix:///tmp/atest-main.sock
    timeout: 30s
    retry_attempts: 3

  external_services:
    schema_registry:
      enabled: false
      endpoint: http://schema-registry:8081

    query_cache:
      enabled: true
      type: redis
      endpoint: redis:6379

    metrics_collector:
      enabled: true
      type: prometheus
      endpoint: http://prometheus:9090
```

## Examples

### Development Configuration

```yaml
# config/development.yaml
ai:
  provider: local
  model: codellama
  confidence_threshold: 0.6
  local:
    ollama_endpoint: http://localhost:11434

plugin:
  socket_path: /tmp/atest-ext-ai-dev.sock
  metrics_port: 9090

logging:
  level: debug
  format: text
  output: stdout

security:
  rate_limit:
    enabled: false

development:
  debug: true
  hot_reload: true
  mock_ai_responses: false
```

### Production Configuration

```yaml
# config/production.yaml
ai:
  provider: openai
  model: gpt-4
  confidence_threshold: 0.8
  max_concurrent_requests: 50
  request_timeout: 60s
  cache_ttl: 3600s

  openai:
    api_key: ${OPENAI_API_KEY}
    max_tokens: 1000
    temperature: 0.2

plugin:
  socket_path: /tmp/atest-ext-ai.sock
  socket_permissions: 0660
  metrics_port: 9090

databases:
  mysql:
    max_connections: 20
    connection_timeout: 5s
  postgresql:
    max_connections: 20
    connection_timeout: 5s

security:
  enable_audit_log: true
  rate_limit:
    enabled: true
    requests_per_minute: 100
    burst_size: 20

  sql_validation:
    enabled: true
    blocked_keywords: ["DROP", "DELETE", "TRUNCATE", "ALTER"]

logging:
  level: warn
  format: json
  output: file
  file:
    path: /var/log/atest-ai/app.log
    max_size: 100MB
    max_files: 10

performance:
  worker_pool_size: 20
  memory_limit: 2GB
  cache:
    enabled: true
    type: redis
    redis:
      address: redis:6379
      password: ${REDIS_PASSWORD}
```

### Docker Configuration

```yaml
# config/docker.yaml
ai:
  provider: local
  model: codellama
  local:
    ollama_endpoint: http://ollama:11434

plugin:
  socket_path: /tmp/atest-ext-ai.sock
  health_check_interval: 30s

logging:
  level: info
  format: json
  output: stdout

monitoring:
  metrics:
    enabled: true
    port: 9090

security:
  rate_limit:
    enabled: true
    requests_per_minute: 60
```

### Kubernetes Configuration

```yaml
# config/kubernetes.yaml
ai:
  provider: ${AI_PROVIDER}
  model: ${AI_MODEL}
  confidence_threshold: 0.7

  openai:
    api_key: ${OPENAI_API_KEY}
  claude:
    api_key: ${CLAUDE_API_KEY}
  local:
    ollama_endpoint: http://ollama-service:11434

plugin:
  socket_path: /tmp/atest-ext-ai.sock
  metrics_port: 9090

security:
  enable_audit_log: true
  audit_log_file: /var/log/atest-ai/audit.log

  rate_limit:
    enabled: true
    requests_per_minute: ${RATE_LIMIT_RPM:-60}

logging:
  level: ${LOG_LEVEL:-info}
  format: json
  output: stdout

performance:
  memory_limit: ${MEMORY_LIMIT:-1GB}
  worker_pool_size: ${WORKER_POOL_SIZE:-10}

  cache:
    enabled: true
    type: redis
    redis:
      address: ${REDIS_ADDRESS:-redis:6379}
      password: ${REDIS_PASSWORD}
```

## Configuration Validation

The plugin validates configuration on startup and provides detailed error messages for invalid settings:

```bash
# Valid configuration check
atest-ext-ai --config config.yaml --validate

# Expected output for valid config
✅ Configuration valid
✅ AI provider accessible
✅ Database connections configured
✅ Security settings applied
✅ Performance limits set
```

For troubleshooting configuration issues, see the [Troubleshooting Guide](TROUBLESHOOTING.md).