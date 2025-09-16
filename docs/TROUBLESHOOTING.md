# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the atest-ext-ai plugin.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Installation Issues](#installation-issues)
- [Configuration Problems](#configuration-problems)
- [Runtime Issues](#runtime-issues)
- [Performance Problems](#performance-problems)
- [AI Provider Issues](#ai-provider-issues)
- [Database Connection Issues](#database-connection-issues)
- [Docker/Kubernetes Issues](#dockerkubernetes-issues)
- [Networking Problems](#networking-problems)
- [Debug Mode](#debug-mode)
- [Log Analysis](#log-analysis)
- [Common Error Messages](#common-error-messages)
- [Getting Help](#getting-help)

## Quick Diagnostics

### Health Check Command

```bash
# Quick health check script
#!/bin/bash
echo "=== atest-ext-ai Health Check ==="

# 1. Check if binary exists and is executable
if command -v atest-store-ai >/dev/null 2>&1; then
    echo "✓ Binary found: $(which atest-store-ai)"
    atest-store-ai --version 2>/dev/null || echo "⚠ Version command failed"
else
    echo "✗ Binary not found in PATH"
fi

# 2. Check socket
if [[ -S "/tmp/atest-store-ai.sock" ]]; then
    echo "✓ Unix socket exists"
else
    echo "✗ Unix socket not found"
fi

# 3. Check service status (systemd)
if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active --quiet atest-store-ai 2>/dev/null; then
        echo "✓ Service is running"
    else
        echo "✗ Service is not running"
    fi
fi

# 4. Check configuration
if [[ -f "/etc/atest-ai/config.yaml" ]]; then
    echo "✓ Configuration file exists"
else
    echo "✗ Configuration file not found"
fi

# 5. Check AI provider connectivity
case "${AI_PROVIDER:-local}" in
    "local")
        if curl -s http://localhost:11434/api/tags >/dev/null 2>&1; then
            echo "✓ Ollama is accessible"
        else
            echo "✗ Ollama is not accessible"
        fi
        ;;
    "openai")
        if [[ -n "${OPENAI_API_KEY:-}" ]]; then
            echo "✓ OpenAI API key is set"
        else
            echo "✗ OpenAI API key is missing"
        fi
        ;;
    "claude")
        if [[ -n "${CLAUDE_API_KEY:-}" ]]; then
            echo "✓ Claude API key is set"
        else
            echo "✗ Claude API key is missing"
        fi
        ;;
esac

# 6. Check metrics endpoint
if curl -f http://localhost:9090/metrics >/dev/null 2>&1; then
    echo "✓ Metrics endpoint is accessible"
else
    echo "✗ Metrics endpoint is not accessible"
fi

echo "=== End Health Check ==="
```

## Installation Issues

### Issue: "Permission denied" during installation

**Symptoms:**
```
permission denied: /usr/local/bin/atest-store-ai
```

**Solutions:**
1. **Use sudo for installation:**
   ```bash
   sudo make install
   ```

2. **Check directory permissions:**
   ```bash
   ls -la /usr/local/bin/
   sudo chown root:wheel /usr/local/bin/  # macOS
   sudo chown root:root /usr/local/bin/   # Linux
   ```

3. **Install to user directory:**
   ```bash
   mkdir -p ~/bin
   cp bin/atest-store-ai ~/bin/
   echo 'export PATH=$HOME/bin:$PATH' >> ~/.bashrc
   source ~/.bashrc
   ```

### Issue: "Binary not found" after installation

**Symptoms:**
```
bash: atest-store-ai: command not found
```

**Solutions:**
1. **Check PATH:**
   ```bash
   echo $PATH
   which atest-store-ai
   ```

2. **Update PATH:**
   ```bash
   export PATH=/usr/local/bin:$PATH
   # Add to ~/.bashrc or ~/.zshrc for persistence
   ```

3. **Use full path:**
   ```bash
   /usr/local/bin/atest-store-ai --version
   ```

### Issue: Go build failures

**Symptoms:**
```
go: golang.org/x/net: module lookup disabled
```

**Solutions:**
1. **Enable Go modules:**
   ```bash
   export GO111MODULE=on
   export GOPROXY=https://proxy.golang.org,direct
   ```

2. **Clear Go cache:**
   ```bash
   go clean -cache -modcache
   go mod download
   ```

3. **Check Go version:**
   ```bash
   go version  # Should be 1.22+
   ```

## Configuration Problems

### Issue: Invalid YAML configuration

**Symptoms:**
```
error parsing config: yaml: line 5: mapping values are not allowed in this context
```

**Solutions:**
1. **Validate YAML syntax:**
   ```bash
   # Install yamllint if not available
   yamllint /etc/atest-ai/config.yaml
   ```

2. **Check indentation:**
   ```yaml
   # Correct
   ai:
     provider: local
     model: codellama

   # Incorrect (mixed tabs/spaces)
   ai:
   	provider: local
       model: codellama
   ```

3. **Use configuration validator:**
   ```bash
   atest-store-ai --config /etc/atest-ai/config.yaml --validate
   ```

### Issue: Environment variables not loading

**Symptoms:**
```
using default value for AI_PROVIDER
```

**Solutions:**
1. **Check variable names:**
   ```bash
   # Correct prefix
   export ATEST_AI_PROVIDER=local

   # Not just
   export AI_PROVIDER=local
   ```

2. **Verify export:**
   ```bash
   env | grep ATEST_AI
   printenv ATEST_AI_PROVIDER
   ```

3. **Source environment file:**
   ```bash
   source /etc/default/atest-ai-plugin
   ```

### Issue: Configuration file permissions

**Symptoms:**
```
permission denied: /etc/atest-ai/config.yaml
```

**Solutions:**
1. **Fix permissions:**
   ```bash
   sudo chown atest:atest /etc/atest-ai/config.yaml
   sudo chmod 640 /etc/atest-ai/config.yaml
   ```

2. **Check user/group:**
   ```bash
   id atest
   groups atest
   ```

## Runtime Issues

### Issue: Plugin fails to start

**Symptoms:**
```
failed to start plugin: address already in use
```

**Solutions:**
1. **Check for existing process:**
   ```bash
   ps aux | grep atest-store-ai
   sudo fuser /tmp/atest-store-ai.sock
   ```

2. **Kill existing process:**
   ```bash
   sudo pkill -f atest-store-ai
   # Or
   sudo systemctl stop atest-store-ai
   ```

3. **Remove stale socket:**
   ```bash
   sudo rm -f /tmp/atest-store-ai.sock
   ```

4. **Change socket path:**
   ```bash
   export AI_PLUGIN_SOCKET_PATH="/tmp/atest-store-ai-$(date +%s).sock"
   ```

### Issue: High memory usage

**Symptoms:**
- System becomes slow
- Out of memory errors
- Plugin crashes

**Solutions:**
1. **Monitor memory usage:**
   ```bash
   ps aux | grep atest-store-ai
   cat /proc/$(pgrep atest-store-ai)/status | grep -E "(VmRSS|VmSize)"
   ```

2. **Adjust memory limits:**
   ```yaml
   # config.yaml
   performance:
     memory:
       limit: 1GB
       gc_percent: 50  # More aggressive GC
   ```

3. **Reduce cache size:**
   ```yaml
   performance:
     cache:
       size: 50MB  # Reduced from default
   ```

4. **Restart service periodically:**
   ```bash
   # Add to crontab for daily restart
   0 2 * * * /bin/systemctl restart atest-store-ai
   ```

### Issue: Plugin crashes on startup

**Symptoms:**
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Solutions:**
1. **Check logs:**
   ```bash
   journalctl -u atest-store-ai -n 50
   # Or for Docker
   docker logs atest-ai-plugin
   ```

2. **Run in debug mode:**
   ```bash
   export LOG_LEVEL=debug
   atest-store-ai --config /etc/atest-ai/config.yaml
   ```

3. **Check dependencies:**
   ```bash
   ldd $(which atest-store-ai)  # Check library dependencies
   ```

## Performance Problems

### Issue: Slow response times

**Symptoms:**
- Requests take >10 seconds
- Timeouts in client applications

**Solutions:**
1. **Check AI provider latency:**
   ```bash
   time curl -X POST http://localhost:11434/api/generate \
     -d '{"model": "codellama", "prompt": "SELECT * FROM users", "stream": false}'
   ```

2. **Optimize configuration:**
   ```yaml
   ai:
     request_timeout: 30s
     max_concurrent_requests: 20

   performance:
     worker_pool_size: 10
     cache:
       enabled: true
       size: 200MB
   ```

3. **Monitor metrics:**
   ```bash
   curl -s http://localhost:9090/metrics | grep -E "(duration|rate)"
   ```

### Issue: High CPU usage

**Symptoms:**
- System load is high
- CPU usage >80%

**Solutions:**
1. **Profile CPU usage:**
   ```bash
   go tool pprof http://localhost:9090/debug/pprof/profile
   ```

2. **Limit concurrent requests:**
   ```yaml
   performance:
     max_concurrent_requests: 5  # Reduce from default
   ```

3. **Check for busy loops:**
   ```bash
   strace -p $(pgrep atest-store-ai) -c
   ```

## AI Provider Issues

### Issue: Ollama connection failed

**Symptoms:**
```
failed to connect to ollama: connection refused
```

**Solutions:**
1. **Check Ollama status:**
   ```bash
   curl http://localhost:11434/api/tags
   systemctl status ollama  # If installed as service
   ```

2. **Start Ollama:**
   ```bash
   ollama serve &
   # Or
   systemctl start ollama
   ```

3. **Check port binding:**
   ```bash
   netstat -tulpn | grep 11434
   lsof -i :11434
   ```

4. **Update endpoint configuration:**
   ```yaml
   ai:
     local:
       ollama_endpoint: http://127.0.0.1:11434  # Try different address
   ```

### Issue: OpenAI API errors

**Symptoms:**
```
openai: invalid API key
openai: rate limit exceeded
```

**Solutions:**
1. **Verify API key:**
   ```bash
   curl -H "Authorization: Bearer $OPENAI_API_KEY" \
     https://api.openai.com/v1/models
   ```

2. **Check rate limits:**
   ```yaml
   ai:
     openai:
       max_tokens: 500        # Reduce token usage
       temperature: 0.1       # Faster responses
   ```

3. **Implement retry logic:**
   ```yaml
   ai:
     openai:
       max_retries: 3
       retry_delay: 5s
   ```

### Issue: Low confidence scores

**Symptoms:**
- All queries return confidence <0.5
- Many queries rejected

**Solutions:**
1. **Lower confidence threshold:**
   ```yaml
   ai:
     confidence_threshold: 0.4  # Reduce from 0.7
   ```

2. **Improve query context:**
   ```json
   {
     "natural_language": "Find all active users from the users table with status column",
     "schema_context": {
       "tables": [{"name": "users", "columns": ["id", "status", "email"]}]
     }
   }
   ```

3. **Try different models:**
   ```yaml
   ai:
     model: wizardcoder  # Better for SQL generation
   ```

## Database Connection Issues

### Issue: Database connection timeout

**Symptoms:**
```
dial tcp: connection timed out
```

**Solutions:**
1. **Test connectivity:**
   ```bash
   telnet postgres-host 5432
   nc -zv mysql-host 3306
   ```

2. **Check firewall:**
   ```bash
   sudo ufw status
   iptables -L
   ```

3. **Increase timeout:**
   ```yaml
   databases:
     mysql:
       connection_timeout: 10s  # Increase from 5s
   ```

### Issue: Authentication failure

**Symptoms:**
```
authentication failed for user "testuser"
```

**Solutions:**
1. **Verify credentials:**
   ```bash
   mysql -h localhost -u testuser -p testdb
   psql -h localhost -U testuser -d testdb
   ```

2. **Check user permissions:**
   ```sql
   -- MySQL
   SHOW GRANTS FOR 'testuser'@'localhost';

   -- PostgreSQL
   \du testuser
   ```

3. **Update configuration:**
   ```yaml
   # Use environment variables for sensitive data
   databases:
     mysql:
       username: ${DB_USERNAME}
       password: ${DB_PASSWORD}
   ```

## Docker/Kubernetes Issues

### Issue: Docker container won't start

**Symptoms:**
```
container exited with code 1
```

**Solutions:**
1. **Check container logs:**
   ```bash
   docker logs atest-ai-plugin
   docker-compose logs atest-ai-plugin
   ```

2. **Run interactively:**
   ```bash
   docker run -it --entrypoint /bin/sh atest-ext-ai:latest
   ```

3. **Check resource limits:**
   ```yaml
   # docker-compose.yml
   services:
     atest-ai-plugin:
       deploy:
         resources:
           limits:
             memory: 2G  # Increase if needed
   ```

### Issue: Kubernetes pod crashes

**Symptoms:**
```
CrashLoopBackOff
```

**Solutions:**
1. **Check pod logs:**
   ```bash
   kubectl logs -l app.kubernetes.io/name=atest-ext-ai -n atest-system
   kubectl describe pod -l app.kubernetes.io/name=atest-ext-ai -n atest-system
   ```

2. **Check resource limits:**
   ```bash
   kubectl top pods -n atest-system
   ```

3. **Update deployment:**
   ```yaml
   resources:
     requests:
       memory: "512Mi"
       cpu: "250m"
     limits:
       memory: "2Gi"
       cpu: "1000m"
   ```

## Networking Problems

### Issue: Port conflicts

**Symptoms:**
```
bind: address already in use
```

**Solutions:**
1. **Find process using port:**
   ```bash
   sudo lsof -i :9090
   sudo netstat -tulpn | grep :9090
   ```

2. **Kill conflicting process:**
   ```bash
   sudo kill $(sudo lsof -t -i:9090)
   ```

3. **Use different port:**
   ```yaml
   plugin:
     metrics_port: 9091  # Change from 9090
   ```

### Issue: DNS resolution failures

**Symptoms:**
```
no such host: api.openai.com
```

**Solutions:**
1. **Test DNS:**
   ```bash
   nslookup api.openai.com
   dig api.openai.com
   ```

2. **Check DNS configuration:**
   ```bash
   cat /etc/resolv.conf
   ```

3. **Use IP addresses:**
   ```yaml
   ai:
     openai:
       api_base: http://104.18.7.192/v1  # Use IP if DNS fails
   ```

## Debug Mode

### Enabling Debug Logging

1. **Environment variable:**
   ```bash
   export LOG_LEVEL=debug
   ```

2. **Configuration file:**
   ```yaml
   logging:
     level: debug
     format: text  # More readable than JSON
   ```

3. **Command line:**
   ```bash
   atest-store-ai --log-level debug
   ```

### Debug Information Collection

```bash
#!/bin/bash
# Collect debug information
echo "=== Debug Information Collection ==="

DEBUG_DIR="/tmp/atest-ai-debug-$(date +%s)"
mkdir -p "$DEBUG_DIR"

# System information
uname -a > "$DEBUG_DIR/system.txt"
cat /etc/os-release >> "$DEBUG_DIR/system.txt" 2>/dev/null

# Process information
ps aux | grep -E "(atest|ollama)" > "$DEBUG_DIR/processes.txt"

# Network information
netstat -tulpn > "$DEBUG_DIR/network.txt"
ss -tulpn >> "$DEBUG_DIR/network.txt" 2>/dev/null

# Service status
if command -v systemctl >/dev/null 2>&1; then
    systemctl status atest-store-ai > "$DEBUG_DIR/service-status.txt" 2>&1
fi

# Configuration
if [[ -f "/etc/atest-ai/config.yaml" ]]; then
    # Remove sensitive data
    sed 's/\(api_key:\s*\).*/\1[REDACTED]/' /etc/atest-ai/config.yaml > "$DEBUG_DIR/config.yaml"
fi

# Environment variables
env | grep -E "(ATEST|AI_|OLLAMA|OPENAI|CLAUDE)" | sed 's/\(.*KEY.*=\).*/\1[REDACTED]/' > "$DEBUG_DIR/environment.txt"

# Logs (last 100 lines)
if command -v journalctl >/dev/null 2>&1; then
    journalctl -u atest-store-ai -n 100 > "$DEBUG_DIR/service-logs.txt" 2>&1
fi

# Docker logs if applicable
if command -v docker >/dev/null 2>&1; then
    docker logs atest-ai-plugin > "$DEBUG_DIR/docker-logs.txt" 2>&1 || true
fi

# Create archive
tar -czf "$DEBUG_DIR.tar.gz" -C "$(dirname "$DEBUG_DIR")" "$(basename "$DEBUG_DIR")"
rm -rf "$DEBUG_DIR"

echo "Debug information collected: $DEBUG_DIR.tar.gz"
```

## Log Analysis

### Log Locations

- **Systemd:** `journalctl -u atest-store-ai`
- **Docker:** `docker logs atest-ai-plugin`
- **File:** `/var/log/atest-ai/app.log`

### Common Log Patterns

1. **Successful startup:**
   ```
   INFO[2024-01-01T12:00:00Z] Starting atest-ext-ai plugin
   INFO[2024-01-01T12:00:00Z] Configuration loaded from /etc/atest-ai/config.yaml
   INFO[2024-01-01T12:00:00Z] AI provider: local (ollama)
   INFO[2024-01-01T12:00:00Z] Plugin ready to accept connections
   ```

2. **Connection errors:**
   ```
   ERROR[2024-01-01T12:00:05Z] Failed to connect to AI provider: connection refused
   ERROR[2024-01-01T12:00:05Z] Retrying in 5 seconds...
   ```

3. **Configuration issues:**
   ```
   ERROR[2024-01-01T12:00:01Z] Invalid configuration: ai.provider is required
   ```

### Log Analysis Commands

```bash
# Recent errors
journalctl -u atest-store-ai --since "1 hour ago" | grep ERROR

# Connection attempts
journalctl -u atest-store-ai | grep -E "(connect|connection|dial)"

# Performance issues
journalctl -u atest-store-ai | grep -E "(timeout|slow|latency)"

# Memory issues
journalctl -u atest-store-ai | grep -E "(memory|oom|killed)"
```

## Common Error Messages

### "dial unix /tmp/atest-store-ai.sock: connect: no such file or directory"

**Cause:** Plugin is not running or socket file doesn't exist.

**Solutions:**
1. Start the plugin service
2. Check socket path configuration
3. Verify socket permissions

### "context deadline exceeded"

**Cause:** Request timeout due to slow AI provider response.

**Solutions:**
1. Increase timeout values
2. Check AI provider connectivity
3. Optimize model selection

### "permission denied"

**Cause:** Insufficient permissions for file/socket access.

**Solutions:**
1. Check file/directory permissions
2. Verify user/group ownership
3. Run with appropriate privileges

### "yaml: unmarshal errors"

**Cause:** Invalid YAML configuration syntax.

**Solutions:**
1. Validate YAML syntax
2. Check indentation (spaces vs tabs)
3. Quote special characters

### "bind: address already in use"

**Cause:** Port is already in use by another process.

**Solutions:**
1. Find and kill conflicting process
2. Use different port
3. Check for multiple plugin instances

## Getting Help

### Before Asking for Help

1. **Check this troubleshooting guide**
2. **Search existing GitHub issues**
3. **Collect debug information** using the script above
4. **Reproduce the issue** with minimal configuration

### Information to Include

When reporting issues, please provide:

1. **System information:**
   - Operating system and version
   - Go version
   - Plugin version

2. **Configuration:**
   - Configuration file (with secrets redacted)
   - Environment variables
   - Command line arguments

3. **Error details:**
   - Complete error messages
   - Log output (recent relevant entries)
   - Steps to reproduce

4. **Environment:**
   - Deployment method (binary, Docker, Kubernetes)
   - AI provider and model
   - Network setup

### Where to Get Help

1. **GitHub Issues:** https://github.com/linuxsuren/atest-ext-ai/issues
2. **Discussions:** https://github.com/linuxsuren/atest-ext-ai/discussions
3. **Documentation:** https://github.com/linuxsuren/atest-ext-ai/docs

### Creating a Minimal Reproduction

```bash
#!/bin/bash
# Minimal test configuration
cat > test-config.yaml << 'EOF'
ai:
  provider: local
  model: codellama
  ollama_endpoint: http://localhost:11434

plugin:
  socket_path: /tmp/atest-store-ai-test.sock
  log_level: debug

logging:
  level: debug
  format: text
  output: stdout
EOF

# Run with minimal config
atest-store-ai --config test-config.yaml
```

This provides a clean test environment to isolate issues from complex configurations.

---

For additional help, please refer to the [Operations Guide](OPERATIONS.md) for production deployment issues or the [User Guide](USER_GUIDE.md) for usage questions.