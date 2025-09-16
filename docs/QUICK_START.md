# Quick Start Guide

Get up and running with atest-ext-ai plugin in minutes! This guide covers the fastest path to start generating SQL from natural language.

## Prerequisites

Before you begin, ensure you have:

- [API Testing Tool](https://github.com/linuxsuren/api-testing) installed
- Go 1.22+ (for building from source)
- [Ollama](https://ollama.ai/) installed (for local AI) **OR** OpenAI/Claude API key

## 🚀 Quick Setup (5 Minutes)

### Step 1: Install the Plugin

#### Option A: Download Binary (Recommended)

```bash
# Download latest release for Linux
curl -L https://github.com/linuxsuren/atest-ext-ai/releases/latest/download/atest-store-ai-linux-amd64.tar.gz | tar xz

# Move to PATH
sudo mv atest-store-ai /usr/local/bin/
chmod +x /usr/local/bin/atest-store-ai
```

#### Option B: Build from Source

```bash
git clone https://github.com/linuxsuren/atest-ext-ai.git
cd atest-ext-ai
make build
sudo make install
```

### Step 2: Set Up Local AI (Ollama)

```bash
# Install Ollama (if not already installed)
curl -fsSL https://ollama.ai/install.sh | sh

# Start Ollama service
ollama serve &

# Pull a code-focused model
ollama pull codellama
```

### Step 3: Start the Plugin

```bash
# Start with default settings
atest-store-ai
```

You should see:
```
INFO[2024-01-01T12:00:00Z] Starting atest-ext-ai plugin
INFO[2024-01-01T12:00:00Z] AI provider: local (ollama)
INFO[2024-01-01T12:00:00Z] Model: codellama
INFO[2024-01-01T12:00:00Z] Unix socket: /tmp/atest-store-ai.sock
INFO[2024-01-01T12:00:00Z] Plugin ready to accept connections
```

### Step 4: Configure Main API Testing Tool

Create or update `stores.yaml`:

```yaml
stores:
  - name: "ai-assistant"
    type: "ai"
    url: "unix:///tmp/atest-store-ai.sock"
    properties:
      - key: ai_provider
        value: local
      - key: model
        value: codellama
      - key: confidence_threshold
        value: "0.7"
```

### Step 5: Test It Out!

#### Via API Testing Tool UI

1. Navigate to your API Testing Tool interface
2. Go to Data Stores → AI Assistant
3. Enter: "Find all users who registered last month"
4. Select database type: MySQL
5. Click Generate

#### Via HTTP API

```bash
curl -X POST http://localhost:8080/api/v1/data/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "natural_language": "Find all users who registered last month",
    "database_type": "mysql"
  }'
```

Expected response:
```json
{
  "data": [
    {
      "key": "generated_sql",
      "value": "SELECT * FROM users WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 MONTH)"
    },
    {
      "key": "explanation",
      "value": "This query retrieves all user records where the creation date is within the last month"
    },
    {
      "key": "confidence_score",
      "value": "0.92"
    }
  ],
  "ai_info": {
    "processing_time_ms": 1200,
    "model_used": "codellama",
    "confidence_score": 0.92,
    "provider": "local"
  }
}
```

## 🎯 Common Use Cases

### 1. Basic SELECT Queries

**Input:** "Show all active products with price above 100"

**Generated SQL:**
```sql
SELECT * FROM products
WHERE status = 'active' AND price > 100
ORDER BY price DESC;
```

### 2. JOIN Queries

**Input:** "Find customers with their order count and total spent"

**Generated SQL:**
```sql
SELECT
    c.id, c.name, c.email,
    COUNT(o.id) as order_count,
    COALESCE(SUM(o.total_amount), 0) as total_spent
FROM customers c
LEFT JOIN orders o ON c.id = o.customer_id
GROUP BY c.id, c.name, c.email
ORDER BY total_spent DESC;
```

### 3. Aggregate Queries

**Input:** "Monthly sales report for current year"

**Generated SQL:**
```sql
SELECT
    DATE_FORMAT(created_at, '%Y-%m') as month,
    COUNT(*) as order_count,
    SUM(total_amount) as revenue,
    AVG(total_amount) as avg_order_value
FROM orders
WHERE YEAR(created_at) = YEAR(NOW())
GROUP BY DATE_FORMAT(created_at, '%Y-%m')
ORDER BY month;
```

## ⚙️ Configuration Options

### Environment Variables (Quick Setup)

```bash
# AI Provider Settings
export AI_PROVIDER="local"                    # local, openai, claude
export AI_MODEL="codellama"                   # Model name
export OLLAMA_ENDPOINT="http://localhost:11434"  # For local provider
export AI_API_KEY="your-api-key"             # For cloud providers

# Plugin Settings
export AI_PLUGIN_SOCKET_PATH="/tmp/atest-store-ai.sock"
export LOG_LEVEL="info"                      # debug, info, warn, error
export AI_CONFIDENCE_THRESHOLD="0.7"        # Minimum confidence (0.0-1.0)
```

### Configuration File (Advanced)

Create `config.yaml`:

```yaml
ai:
  provider: local
  model: codellama
  confidence_threshold: 0.7
  ollama_endpoint: http://localhost:11434

plugin:
  socket_path: /tmp/atest-store-ai.sock
  log_level: info
  metrics_enabled: true
  metrics_port: 9090

security:
  rate_limit:
    enabled: true
    requests_per_minute: 60
```

Then start with:
```bash
atest-store-ai --config config.yaml
```

## 🔧 Troubleshooting

### Plugin Won't Start

**Problem:** Permission denied on socket
```
ERROR: failed to create unix socket: permission denied
```

**Solution:** Check socket directory permissions
```bash
sudo mkdir -p /tmp
sudo chmod 755 /tmp
# OR change socket path
export AI_PLUGIN_SOCKET_PATH="$HOME/atest-store-ai.sock"
```

### Ollama Connection Failed

**Problem:** Can't connect to Ollama
```
ERROR: failed to connect to ollama: connection refused
```

**Solution:** Ensure Ollama is running
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# If not, start it
ollama serve &

# Wait a moment, then test again
curl http://localhost:11434/api/tags
```

### Low Confidence Scores

**Problem:** Generated SQL has low confidence

**Solutions:**
1. **Lower threshold:**
   ```bash
   export AI_CONFIDENCE_THRESHOLD="0.5"
   ```

2. **Provide more context:**
   ```json
   {
     "natural_language": "Find all active users who registered in the last 30 days from the users table",
     "database_type": "mysql",
     "schema_context": {
       "tables": [
         {
           "name": "users",
           "columns": ["id", "username", "email", "status", "created_at"]
         }
       ]
     }
   }
   ```

3. **Try a different model:**
   ```bash
   export AI_MODEL="wizardcoder"  # Better for complex SQL
   ```

### Main API Tool Can't Connect

**Problem:** API Testing Tool can't find the plugin

**Solution:** Verify socket path in stores.yaml matches plugin configuration:
```yaml
stores:
  - name: "ai-assistant"
    type: "ai"
    url: "unix:///tmp/atest-store-ai.sock"  # Must match plugin socket_path
```

## 📊 Monitoring

### Health Check

```bash
# Check if plugin is running
test -S /tmp/atest-store-ai.sock && echo "Plugin running" || echo "Plugin not running"

# Check metrics (if enabled)
curl http://localhost:9090/metrics
```

### Logs

```bash
# View plugin logs (if running in foreground)
atest-store-ai --log-level debug

# Or check system logs
journalctl -u atest-store-ai -f
```

## 🚀 Next Steps

Now that you have the basic setup working, explore these advanced features:

1. **[Configuration Guide](CONFIGURATION.md)** - Detailed configuration options
2. **[API Documentation](API.md)** - Complete API reference
3. **[User Guide](USER_GUIDE.md)** - Advanced usage patterns
4. **[Deployment Guide](DEPLOYMENT.md)** - Production deployment
5. **[Troubleshooting](TROUBLESHOOTING.md)** - Common issues and solutions

### Try Different AI Providers

#### OpenAI Setup
```bash
export AI_PROVIDER="openai"
export AI_MODEL="gpt-4"
export OPENAI_API_KEY="sk-your-key-here"
atest-store-ai
```

#### Claude Setup
```bash
export AI_PROVIDER="claude"
export AI_MODEL="claude-3-sonnet"
export CLAUDE_API_KEY="sk-your-key-here"
atest-store-ai
```

### Docker Quick Start

```bash
# Run with Docker Compose
git clone https://github.com/linuxsuren/atest-ext-ai.git
cd atest-ext-ai
docker-compose up -d

# Test the setup
curl -X POST http://localhost:8080/api/v1/data/query \
  -H "Content-Type: application/json" \
  -d '{"type": "ai", "natural_language": "Show all users", "database_type": "mysql"}'
```

## 💡 Tips for Better Results

### 1. Be Specific
❌ **Vague:** "Get user data"
✅ **Specific:** "Get all active users with their email addresses and registration dates"

### 2. Include Context
❌ **No Context:** "Show sales report"
✅ **With Context:** "Show monthly sales report for 2024 with total revenue and order count"

### 3. Specify Database Type
Different databases have different syntax. Always specify:
- MySQL: `DATE_SUB()`, `LIMIT`
- PostgreSQL: `INTERVAL`, `LIMIT`
- SQLite: `datetime()`, `LIMIT`

### 4. Use Schema Information
Provide table and column names when possible for better results.

Happy querying! 🎉