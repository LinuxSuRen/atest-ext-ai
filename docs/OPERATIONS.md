# Operations Guide

This guide covers production deployment, monitoring, maintenance, and troubleshooting for the atest-ext-ai plugin.

## Table of Contents

- [Production Deployment](#production-deployment)
- [Monitoring & Observability](#monitoring--observability)
- [Performance Tuning](#performance-tuning)
- [Backup & Recovery](#backup--recovery)
- [Maintenance Procedures](#maintenance-procedures)
- [Scaling Strategies](#scaling-strategies)
- [Troubleshooting](#troubleshooting)
- [Disaster Recovery](#disaster-recovery)

## Production Deployment

### Infrastructure Requirements

#### Minimum System Requirements

| Resource | Minimum | Recommended | High-Load |
|----------|---------|-------------|-----------|
| CPU | 2 cores | 4 cores | 8+ cores |
| Memory | 2GB | 4GB | 8GB+ |
| Storage | 10GB | 50GB | 100GB+ |
| Network | 100Mbps | 1Gbps | 10Gbps |

#### Supported Platforms

- **Linux**: Ubuntu 20.04+, CentOS 8+, RHEL 8+, Debian 11+
- **Containers**: Docker, Kubernetes, Podman
- **Cloud**: AWS, GCP, Azure, DigitalOcean
- **Architecture**: x86_64, ARM64

### Deployment Options

#### 1. Systemd Service (Recommended)

Create service file `/etc/systemd/system/atest-ai-plugin.service`:

```ini
[Unit]
Description=atest-ext-ai Plugin
Documentation=https://github.com/linuxsuren/atest-ext-ai
After=network.target
Wants=network.target

[Service]
Type=simple
User=atest
Group=atest
ExecStart=/usr/local/bin/atest-ext-ai --config /etc/atest-ai/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=atest-ai-plugin

# Security
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/tmp /var/log/atest-ai
PrivateTmp=yes

# Resource limits
LimitNOFILE=65536
MemoryMax=2G
CPUQuota=200%

# Environment
Environment=AI_PLUGIN_SOCKET_PATH=/tmp/atest-ext-ai.sock
Environment=LOG_LEVEL=info
EnvironmentFile=-/etc/default/atest-ai-plugin

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable atest-ai-plugin
sudo systemctl start atest-ai-plugin
sudo systemctl status atest-ai-plugin
```

#### 2. Docker Deployment

```yaml
# docker-compose.prod.yml
version: '3.8'
services:
  atest-ai-plugin:
    image: ghcr.io/linuxsuren/atest-ext-ai:latest
    restart: unless-stopped
    environment:
      - AI_PROVIDER=${AI_PROVIDER}
      - AI_MODEL=${AI_MODEL}
      - LOG_LEVEL=info
    volumes:
      - socket_volume:/tmp
      - ./config/production.yaml:/etc/atest-ai/config.yaml:ro
      - logs:/var/log/atest-ai
    healthcheck:
      test: ["CMD", "test", "-S", "/tmp/atest-ext-ai.sock"]
      interval: 30s
      timeout: 10s
      retries: 3
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "5"
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '2.0'
        reservations:
          memory: 1G
          cpus: '1.0'

volumes:
  socket_volume:
  logs:
```

#### 3. Kubernetes Deployment

Apply the manifests from the `k8s/` directory:

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/pvc.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/hpa.yaml
```

Verify deployment:

```bash
kubectl get pods -n atest-system
kubectl describe pod -l app.kubernetes.io/name=atest-ext-ai -n atest-system
kubectl logs -l app.kubernetes.io/name=atest-ext-ai -n atest-system
```

### Configuration Management

#### Environment-Specific Configs

```bash
# Development
cp config/development.yaml /etc/atest-ai/config.yaml

# Staging
cp config/staging.yaml /etc/atest-ai/config.yaml

# Production
cp config/production.yaml /etc/atest-ai/config.yaml
```

#### Configuration Validation

```bash
# Validate configuration
atest-ext-ai --config /etc/atest-ai/config.yaml --validate

# Test connectivity
atest-ext-ai --config /etc/atest-ai/config.yaml --test-providers
```

#### Secrets Management

##### Using HashiCorp Vault

```yaml
# vault-config.yaml
secrets:
  vault:
    enabled: true
    address: https://vault.example.com
    auth_method: kubernetes
    role: atest-ai-plugin
    mount_path: secret/atest-ai

    mappings:
      - vault_path: "openai/api_key"
        env_var: "OPENAI_API_KEY"
      - vault_path: "claude/api_key"
        env_var: "CLAUDE_API_KEY"
```

##### Using Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: atest-ai-secrets
type: Opaque
stringData:
  openai-api-key: "sk-..."
  claude-api-key: "sk-..."
  jwt-secret: "your-strong-jwt-secret"
```

## Monitoring & Observability

### Metrics Collection

#### Prometheus Metrics

The plugin exposes metrics on `/metrics` endpoint (default port 9090):

```yaml
# Key metrics to monitor
- atest_ai_requests_total{provider,model,status}
- atest_ai_request_duration_seconds{provider,model}
- atest_ai_confidence_score_bucket{provider,model}
- atest_ai_active_connections
- atest_ai_cache_hits_total
- atest_ai_cache_misses_total
- atest_ai_errors_total{type,provider}
- atest_ai_memory_usage_bytes
- atest_ai_cpu_usage_percent
```

#### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'atest-ai-plugin'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: /metrics
    scrape_interval: 15s
    scrape_timeout: 10s
```

#### Grafana Dashboard

Import the dashboard from `monitoring/grafana/dashboard.json`:

```json
{
  "dashboard": {
    "title": "atest-ext-ai Plugin Monitoring",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(atest_ai_requests_total[5m])",
            "legendFormat": "{{status}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(atest_ai_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          }
        ]
      }
    ]
  }
}
```

### Health Checks

#### Liveness Probe

```bash
#!/bin/bash
# /usr/local/bin/liveness-check.sh
test -S /tmp/atest-ext-ai.sock
```

#### Readiness Probe

```bash
#!/bin/bash
# /usr/local/bin/readiness-check.sh
curl -f http://localhost:9090/health/ready || exit 1
```

#### Startup Probe

```yaml
# Kubernetes startup probe
startupProbe:
  exec:
    command:
    - /usr/local/bin/startup-check.sh
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 30
```

### Logging

#### Structured Logging

```yaml
logging:
  level: info
  format: json
  output: stdout

  # Additional context
  fields:
    service: atest-ai-plugin
    version: ${VERSION}
    environment: ${ENVIRONMENT}
    hostname: ${HOSTNAME}

  # Component-specific levels
  components:
    ai_provider: info
    sql_generator: info
    database: warn
    metrics: error
```

#### Log Aggregation

##### Using Fluentd

```yaml
# fluentd-config.yaml
<source>
  @type tail
  path /var/log/atest-ai/*.log
  pos_file /var/log/fluentd/atest-ai.log.pos
  tag atest-ai
  format json
</source>

<match atest-ai>
  @type elasticsearch
  host elasticsearch.logging.svc.cluster.local
  port 9200
  index_name atest-ai
  type_name _doc
</match>
```

##### Using Filebeat

```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/atest-ai/*.log
  fields:
    service: atest-ai-plugin
  fields_under_root: true

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "atest-ai-%{+yyyy.MM.dd}"
```

### Alerting

#### Prometheus AlertManager Rules

```yaml
# alerts.yml
groups:
- name: atest-ai-plugin
  rules:
  - alert: PluginDown
    expr: up{job="atest-ai-plugin"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "atest-ext-ai plugin is down"
      description: "Plugin has been down for more than 1 minute"

  - alert: HighErrorRate
    expr: rate(atest_ai_errors_total[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value }} errors/second"

  - alert: LowConfidenceQueries
    expr: histogram_quantile(0.5, rate(atest_ai_confidence_score_bucket[5m])) < 0.7
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Many low-confidence queries"
      description: "Median confidence score is {{ $value }}"

  - alert: MemoryUsageHigh
    expr: atest_ai_memory_usage_bytes > 1.5e9  # 1.5GB
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage"
      description: "Memory usage is {{ $value | humanize1024 }}"
```

#### Notification Channels

```yaml
# alertmanager.yml
route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
  - match:
      severity: critical
    receiver: 'critical-alerts'

receivers:
- name: 'default'
  email_configs:
  - to: 'ops-team@company.com'
    subject: '[ALERT] {{ .GroupLabels.alertname }}'
    body: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

- name: 'critical-alerts'
  email_configs:
  - to: 'ops-team@company.com'
    subject: '[CRITICAL] {{ .GroupLabels.alertname }}'
  slack_configs:
  - api_url: 'https://hooks.slack.com/services/...'
    channel: '#ops-alerts'
    text: 'Critical alert: {{ .GroupLabels.alertname }}'
```

## Performance Tuning

### Memory Optimization

#### Go Runtime Tuning

```bash
# Environment variables for Go runtime
export GOGC=100              # GC target percentage
export GOMEMLIMIT=2GiB       # Memory limit
export GOMAXPROCS=4          # CPU cores to use
```

#### Configuration Tuning

```yaml
performance:
  memory:
    limit: 2GB
    gc_percent: 100
    allocation_rate_limit: 100MB/s

  worker_pools:
    ai_requests:
      size: 10
      queue_size: 100
      timeout: 60s

  cache:
    enabled: true
    size: 500MB
    ttl: 3600s
    cleanup_interval: 300s
```

### Connection Pooling

```yaml
performance:
  connection_pools:
    ai_providers:
      max_idle_connections: 10
      max_active_connections: 50
      connection_timeout: 30s
      idle_timeout: 300s

    databases:
      max_idle_connections: 5
      max_active_connections: 20
      connection_timeout: 5s
      idle_timeout: 600s
```

### Caching Strategies

#### Redis Cache Configuration

```yaml
performance:
  cache:
    type: redis
    redis:
      address: redis:6379
      password: ${REDIS_PASSWORD}
      database: 0
      pool_size: 10
      max_retries: 3
      retry_delay: 1s

    # Cache policies
    policies:
      sql_queries:
        ttl: 3600s
        max_size: 100MB
      schema_info:
        ttl: 7200s
        max_size: 10MB
```

#### Cache Warming

```bash
#!/bin/bash
# cache-warm.sh - Pre-populate cache with common queries

COMMON_QUERIES=(
  "Show all users"
  "Get product count"
  "Calculate total revenue"
  "Find active customers"
)

for query in "${COMMON_QUERIES[@]}"; do
  curl -X POST http://localhost:8080/api/v1/data/query \
    -H "Content-Type: application/json" \
    -d "{\"type\": \"ai\", \"natural_language\": \"$query\", \"database_type\": \"mysql\"}"
  sleep 1
done
```

### Load Balancing

#### HAProxy Configuration

```
# haproxy.cfg
global
    daemon
    log stdout local0

defaults
    mode http
    timeout connect 5000ms
    timeout client 50000ms
    timeout server 50000ms

backend atest-ai-plugins
    balance roundrobin
    option httpchk GET /health
    server plugin1 127.0.0.1:9091 check
    server plugin2 127.0.0.1:9092 check
    server plugin3 127.0.0.1:9093 check

frontend atest-ai-frontend
    bind *:9090
    default_backend atest-ai-plugins
```

## Backup & Recovery

### Configuration Backup

```bash
#!/bin/bash
# backup-config.sh

BACKUP_DIR="/var/backups/atest-ai"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/config_$TIMESTAMP.tar.gz"

mkdir -p "$BACKUP_DIR"

# Backup configuration files
tar -czf "$BACKUP_FILE" \
  /etc/atest-ai/ \
  /etc/systemd/system/atest-ai-plugin.service

# Keep only last 30 days of backups
find "$BACKUP_DIR" -name "config_*.tar.gz" -mtime +30 -delete

echo "Configuration backup completed: $BACKUP_FILE"
```

### Log Backup

```bash
#!/bin/bash
# backup-logs.sh

LOG_DIR="/var/log/atest-ai"
BACKUP_DIR="/var/backups/atest-ai/logs"
TIMESTAMP=$(date +%Y%m%d)

mkdir -p "$BACKUP_DIR"

# Compress and backup logs
tar -czf "$BACKUP_DIR/logs_$TIMESTAMP.tar.gz" -C "$LOG_DIR" .

# Upload to cloud storage (optional)
if command -v aws &> /dev/null; then
    aws s3 cp "$BACKUP_DIR/logs_$TIMESTAMP.tar.gz" \
        s3://your-backup-bucket/atest-ai/logs/
fi

# Keep local backups for 7 days
find "$BACKUP_DIR" -name "logs_*.tar.gz" -mtime +7 -delete
```

### Recovery Procedures

#### Configuration Recovery

```bash
#!/bin/bash
# restore-config.sh

BACKUP_FILE="$1"

if [[ ! -f "$BACKUP_FILE" ]]; then
    echo "Backup file not found: $BACKUP_FILE"
    exit 1
fi

# Stop service
sudo systemctl stop atest-ai-plugin

# Backup current config
sudo mv /etc/atest-ai /etc/atest-ai.bak

# Restore from backup
sudo tar -xzf "$BACKUP_FILE" -C /

# Restart service
sudo systemctl start atest-ai-plugin
sudo systemctl status atest-ai-plugin
```

#### Disaster Recovery

```yaml
# disaster-recovery.yml
recovery_procedures:
  data_center_failure:
    rto: "4 hours"        # Recovery Time Objective
    rpo: "1 hour"         # Recovery Point Objective
    steps:
      1. "Activate backup data center"
      2. "Restore configuration from backups"
      3. "Update DNS records"
      4. "Validate service functionality"
      5. "Notify stakeholders"

  service_corruption:
    rto: "30 minutes"
    rpo: "15 minutes"
    steps:
      1. "Stop corrupted service"
      2. "Restore from latest backup"
      3. "Validate configuration"
      4. "Restart service"
      5. "Monitor for stability"
```

## Maintenance Procedures

### Routine Maintenance

#### Daily Tasks

```bash
#!/bin/bash
# daily-maintenance.sh

# Check service status
systemctl status atest-ai-plugin

# Check disk space
df -h /var/log/atest-ai

# Check memory usage
free -h

# Rotate logs if needed
logrotate /etc/logrotate.d/atest-ai-plugin

# Check for errors in last 24 hours
journalctl -u atest-ai-plugin --since "24 hours ago" | grep -i error
```

#### Weekly Tasks

```bash
#!/bin/bash
# weekly-maintenance.sh

# Update system packages
sudo apt update && sudo apt upgrade -y

# Clean up old logs
find /var/log/atest-ai -name "*.log.gz" -mtime +30 -delete

# Vacuum cache if using file-based cache
find /var/cache/atest-ai -name "*.cache" -mtime +7 -delete

# Check certificate expiration
openssl x509 -in /etc/ssl/certs/plugin.crt -noout -dates
```

#### Monthly Tasks

```bash
#!/bin/bash
# monthly-maintenance.sh

# Performance analysis
echo "=== Performance Report ===" > /tmp/perf-report.txt
curl -s http://localhost:9090/metrics >> /tmp/perf-report.txt

# Security audit
echo "=== Security Audit ===" >> /tmp/audit-report.txt
grep -i "authentication_failure" /var/log/atest-ai/audit.log >> /tmp/audit-report.txt

# Configuration review
atest-ext-ai --config /etc/atest-ai/config.yaml --validate

# Update AI models (for local provider)
if [[ "$AI_PROVIDER" == "local" ]]; then
    ollama pull "$AI_MODEL"
fi
```

### Updates and Upgrades

#### Plugin Updates

```bash
#!/bin/bash
# update-plugin.sh

VERSION="$1"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# Download new version
wget "https://github.com/linuxsuren/atest-ext-ai/releases/download/v${VERSION}/atest-ext-ai-linux-amd64.tar.gz"

# Verify checksum
wget "https://github.com/linuxsuren/atest-ext-ai/releases/download/v${VERSION}/SHA256SUMS"
sha256sum -c SHA256SUMS

# Backup current version
sudo cp /usr/local/bin/atest-ext-ai /usr/local/bin/atest-ext-ai.bak

# Stop service
sudo systemctl stop atest-ai-plugin

# Install new version
tar -xzf "atest-ext-ai-linux-amd64.tar.gz"
sudo cp atest-ext-ai /usr/local/bin/
sudo chmod +x /usr/local/bin/atest-ext-ai

# Start service
sudo systemctl start atest-ai-plugin

# Verify update
atest-ext-ai --version
sudo systemctl status atest-ai-plugin
```

#### Rolling Updates (Kubernetes)

```bash
# Update container image
kubectl set image deployment/atest-ai-plugin \
    atest-ai-plugin=ghcr.io/linuxsuren/atest-ext-ai:v1.2.0 \
    -n atest-system

# Monitor rollout
kubectl rollout status deployment/atest-ai-plugin -n atest-system

# Verify update
kubectl get pods -l app.kubernetes.io/name=atest-ext-ai -n atest-system
```

### Configuration Changes

#### Safe Configuration Updates

```bash
#!/bin/bash
# safe-config-update.sh

CONFIG_FILE="/etc/atest-ai/config.yaml"
BACKUP_FILE="/etc/atest-ai/config.yaml.backup"

# Backup current configuration
sudo cp "$CONFIG_FILE" "$BACKUP_FILE"

# Validate new configuration
if atest-ext-ai --config "$CONFIG_FILE" --validate; then
    echo "Configuration valid, restarting service..."
    sudo systemctl restart atest-ai-plugin

    # Wait and check status
    sleep 5
    if systemctl is-active --quiet atest-ai-plugin; then
        echo "Service restarted successfully"
        rm "$BACKUP_FILE"
    else
        echo "Service failed to start, rolling back..."
        sudo cp "$BACKUP_FILE" "$CONFIG_FILE"
        sudo systemctl restart atest-ai-plugin
        exit 1
    fi
else
    echo "Configuration invalid, keeping backup"
    exit 1
fi
```

## Scaling Strategies

### Horizontal Scaling

#### Multiple Plugin Instances

```yaml
# kubernetes scaling
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atest-ai-plugin
spec:
  replicas: 5  # Scale to 5 instances
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
```

#### Load Distribution

```yaml
# Service with session affinity
apiVersion: v1
kind: Service
metadata:
  name: atest-ai-plugin-service
spec:
  sessionAffinity: ClientIP
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 300
  ports:
  - port: 9090
    targetPort: 9090
  selector:
    app: atest-ai-plugin
```

### Vertical Scaling

#### Resource Scaling

```yaml
# Increase resources
resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "4Gi"    # Increased from 2Gi
    cpu: "2000m"     # Increased from 1000m
```

#### Auto-scaling Configuration

```yaml
# HPA configuration
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: atest-ai-plugin-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: atest-ai-plugin
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

## Troubleshooting

### Common Issues

#### 1. Plugin Won't Start

**Symptoms:**
- Service fails to start
- "Permission denied" errors
- Socket creation failures

**Diagnosis:**
```bash
# Check service logs
journalctl -u atest-ai-plugin -n 50

# Check file permissions
ls -la /tmp/atest-ext-ai.sock
ls -la /etc/atest-ai/

# Check configuration
atest-ext-ai --config /etc/atest-ai/config.yaml --validate
```

**Solutions:**
```bash
# Fix permissions
sudo chown atest:atest /tmp/atest-ext-ai.sock
sudo chmod 660 /tmp/atest-ext-ai.sock

# Check user exists
id atest || sudo useradd -r -s /bin/false atest

# Validate configuration syntax
yamllint /etc/atest-ai/config.yaml
```

#### 2. High Memory Usage

**Symptoms:**
- Memory usage above 80%
- OOM killer messages
- Slow response times

**Diagnosis:**
```bash
# Check memory usage
ps aux | grep atest-ext-ai
cat /proc/$(pgrep atest-ext-ai)/status | grep -E "(VmRSS|VmSize)"

# Check for memory leaks
curl http://localhost:9090/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**Solutions:**
```yaml
# Adjust configuration
performance:
  memory:
    limit: 1GB
    gc_percent: 50  # More aggressive GC

  cache:
    size: 100MB     # Reduce cache size
```

#### 3. AI Provider Connection Issues

**Symptoms:**
- "Connection refused" errors
- Timeout errors
- Authentication failures

**Diagnosis:**
```bash
# Test connectivity
curl -v http://localhost:11434/api/tags  # Ollama
curl -v https://api.openai.com/v1/models  # OpenAI

# Check DNS resolution
nslookup api.openai.com

# Test with curl
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
     https://api.openai.com/v1/models
```

**Solutions:**
```bash
# Restart Ollama
sudo systemctl restart ollama

# Check firewall
sudo ufw status
sudo iptables -L

# Update configuration
export AI_PROVIDER=local
export OLLAMA_ENDPOINT=http://localhost:11434
```

#### 4. Low Performance

**Symptoms:**
- Slow response times
- High CPU usage
- Request timeouts

**Diagnosis:**
```bash
# Check performance metrics
curl http://localhost:9090/metrics | grep -E "(duration|rate)"

# Profile CPU usage
curl http://localhost:9090/debug/pprof/profile > cpu.prof
go tool pprof cpu.prof

# Check system resources
top -p $(pgrep atest-ext-ai)
iostat -x 1
```

**Solutions:**
```yaml
# Optimize configuration
performance:
  worker_pool_size: 20
  cache:
    enabled: true
    size: 500MB
    ttl: 3600s

ai:
  request_timeout: 30s
  max_concurrent_requests: 20
```

### Debug Mode

#### Enable Debug Logging

```bash
# Temporary debug mode
sudo systemctl edit atest-ai-plugin --full
# Add: Environment=LOG_LEVEL=debug

sudo systemctl restart atest-ai-plugin
```

#### Debug Tools

```bash
# Memory profiling
go tool pprof http://localhost:9090/debug/pprof/heap

# CPU profiling
go tool pprof http://localhost:9090/debug/pprof/profile

# Goroutine analysis
go tool pprof http://localhost:9090/debug/pprof/goroutine

# Trace analysis
wget http://localhost:9090/debug/pprof/trace
go tool trace trace
```

### Recovery Procedures

#### Emergency Shutdown

```bash
#!/bin/bash
# emergency-shutdown.sh

echo "Initiating emergency shutdown..."

# Stop accepting new requests
curl -X POST http://localhost:9090/admin/maintenance-mode

# Wait for current requests to complete
sleep 30

# Force shutdown
sudo systemctl stop atest-ai-plugin

echo "Emergency shutdown completed"
```

#### Service Recovery

```bash
#!/bin/bash
# service-recovery.sh

echo "Starting service recovery..."

# Check prerequisites
systemctl status ollama
systemctl status docker

# Restore from backup if needed
if [[ ! -f "/etc/atest-ai/config.yaml" ]]; then
    echo "Restoring configuration from backup..."
    sudo tar -xzf /var/backups/atest-ai/config_latest.tar.gz -C /
fi

# Start service
sudo systemctl start atest-ai-plugin

# Wait for startup
sleep 10

# Verify functionality
if curl -f http://localhost:9090/health; then
    echo "Service recovered successfully"
else
    echo "Service recovery failed"
    exit 1
fi
```

## Disaster Recovery

### Recovery Time Objectives (RTO)

| Disaster Type | RTO | RPO | Priority |
|---------------|-----|-----|----------|
| Service crash | 5 minutes | 0 | P1 |
| Host failure | 15 minutes | 5 minutes | P1 |
| Data center outage | 4 hours | 1 hour | P2 |
| Region failure | 8 hours | 2 hours | P3 |

### Backup Strategy

```yaml
backup:
  schedule:
    configuration: "daily at 02:00"
    logs: "hourly"
    metrics: "daily at 03:00"

  retention:
    daily: 7
    weekly: 4
    monthly: 12

  locations:
    primary: "/var/backups/atest-ai"
    secondary: "s3://backup-bucket/atest-ai"
    offsite: "gcs://dr-backups/atest-ai"
```

### Failover Procedures

#### Automatic Failover (Kubernetes)

```yaml
# Multi-zone deployment
apiVersion: apps/v1
kind: Deployment
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - atest-ai-plugin
            topologyKey: kubernetes.io/hostname
```

#### Manual Failover

```bash
#!/bin/bash
# manual-failover.sh

PRIMARY_HOST="primary.example.com"
SECONDARY_HOST="secondary.example.com"

echo "Initiating manual failover..."

# Stop service on primary
ssh "$PRIMARY_HOST" "sudo systemctl stop atest-ai-plugin"

# Start service on secondary
ssh "$SECONDARY_HOST" "sudo systemctl start atest-ai-plugin"

# Update load balancer
curl -X POST http://loadbalancer/api/backend \
     -d '{"remove": "'$PRIMARY_HOST'", "add": "'$SECONDARY_HOST'"}'

echo "Failover completed"
```

For additional troubleshooting information, see the [Troubleshooting Guide](TROUBLESHOOTING.md).