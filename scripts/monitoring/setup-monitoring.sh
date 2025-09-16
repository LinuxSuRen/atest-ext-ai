#!/bin/bash
# Setup monitoring stack for atest-ext-ai plugin
set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
MONITORING_DIR="$PROJECT_DIR/monitoring"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
info() { echo -e "${BLUE}[INFO]${NC} $*"; }

# Create monitoring directory structure
setup_directories() {
    log "Creating monitoring directories..."

    mkdir -p "$MONITORING_DIR"/{prometheus,grafana/dashboards,grafana/provisioning/{dashboards,datasources},alertmanager}

    log "Directories created"
}

# Setup Prometheus configuration
setup_prometheus() {
    log "Setting up Prometheus configuration..."

    cat > "$MONITORING_DIR/prometheus/prometheus.yml" << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'atest-ai-plugin'
    static_configs:
      - targets: ['atest-ai-plugin:9090']
    metrics_path: /metrics
    scrape_interval: 15s

  - job_name: 'ollama'
    static_configs:
      - targets: ['ollama:11434']
    metrics_path: /metrics
    scrape_interval: 30s

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres_exporter:9187']
    scrape_interval: 30s

  - job_name: 'redis'
    static_configs:
      - targets: ['redis_exporter:9121']
    scrape_interval: 30s

  - job_name: 'node'
    static_configs:
      - targets: ['node_exporter:9100']
    scrape_interval: 30s
EOF

    # Create alert rules
    mkdir -p "$MONITORING_DIR/prometheus/rules"

    cat > "$MONITORING_DIR/prometheus/rules/atest-ai.yml" << 'EOF'
groups:
- name: atest-ai-plugin
  rules:
  - alert: PluginDown
    expr: up{job="atest-ai-plugin"} == 0
    for: 1m
    labels:
      severity: critical
      service: atest-ai-plugin
    annotations:
      summary: "atest-ext-ai plugin is down"
      description: "Plugin has been down for more than 1 minute"

  - alert: HighErrorRate
    expr: rate(atest_ai_errors_total[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
      service: atest-ai-plugin
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value }} errors/second"

  - alert: LowConfidenceQueries
    expr: histogram_quantile(0.5, rate(atest_ai_confidence_score_bucket[5m])) < 0.7
    for: 5m
    labels:
      severity: warning
      service: atest-ai-plugin
    annotations:
      summary: "Many low-confidence queries"
      description: "Median confidence score is {{ $value }}"

  - alert: MemoryUsageHigh
    expr: atest_ai_memory_usage_bytes > 1.5e9
    for: 5m
    labels:
      severity: warning
      service: atest-ai-plugin
    annotations:
      summary: "High memory usage"
      description: "Memory usage is {{ $value | humanize1024 }}"

  - alert: RequestRateHigh
    expr: rate(atest_ai_requests_total[5m]) > 10
    for: 2m
    labels:
      severity: info
      service: atest-ai-plugin
    annotations:
      summary: "High request rate"
      description: "Request rate is {{ $value }} requests/second"

  - alert: OllamaDown
    expr: up{job="ollama"} == 0
    for: 2m
    labels:
      severity: critical
      service: ollama
    annotations:
      summary: "Ollama service is down"
      description: "Ollama has been unreachable for more than 2 minutes"
EOF

    log "Prometheus configuration created"
}

# Setup Grafana configuration
setup_grafana() {
    log "Setting up Grafana configuration..."

    # Datasource configuration
    cat > "$MONITORING_DIR/grafana/provisioning/datasources/prometheus.yml" << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
EOF

    # Dashboard provisioning
    cat > "$MONITORING_DIR/grafana/provisioning/dashboards/dashboards.yml" << 'EOF'
apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    options:
      path: /etc/grafana/provisioning/dashboards
EOF

    # Main dashboard
    cat > "$MONITORING_DIR/grafana/dashboards/atest-ai-dashboard.json" << 'EOF'
{
  "dashboard": {
    "id": null,
    "title": "atest-ext-ai Plugin Monitoring",
    "tags": ["atest", "ai", "plugin"],
    "timezone": "browser",
    "refresh": "30s",
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "panels": [
      {
        "id": 1,
        "title": "Request Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(atest_ai_requests_total[5m])",
            "legendFormat": "Requests/sec"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 0}
      },
      {
        "id": 2,
        "title": "Error Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(atest_ai_errors_total[5m])",
            "legendFormat": "Errors/sec"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0}
      },
      {
        "id": 3,
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(atest_ai_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(atest_ai_request_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ],
        "gridPos": {"h": 8, "w": 24, "x": 0, "y": 8}
      },
      {
        "id": 4,
        "title": "Confidence Score Distribution",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(atest_ai_confidence_score_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(atest_ai_confidence_score_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 16}
      },
      {
        "id": 5,
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "atest_ai_memory_usage_bytes",
            "legendFormat": "Memory Usage"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 16}
      },
      {
        "id": 6,
        "title": "Active Connections",
        "type": "stat",
        "targets": [
          {
            "expr": "atest_ai_active_connections",
            "legendFormat": "Connections"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 24}
      },
      {
        "id": 7,
        "title": "Cache Hit Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(atest_ai_cache_hits_total[5m]) / (rate(atest_ai_cache_hits_total[5m]) + rate(atest_ai_cache_misses_total[5m])) * 100",
            "legendFormat": "Hit Rate %"
          }
        ],
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 24}
      }
    ]
  }
}
EOF

    log "Grafana configuration created"
}

# Setup AlertManager configuration
setup_alertmanager() {
    log "Setting up AlertManager configuration..."

    cat > "$MONITORING_DIR/alertmanager/alertmanager.yml" << 'EOF'
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alertmanager@example.com'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'
  routes:
  - match:
      severity: critical
    receiver: 'critical-alerts'
  - match:
      severity: warning
    receiver: 'warning-alerts'

receivers:
- name: 'web.hook'
  webhook_configs:
  - url: 'http://webhook:5001/'

- name: 'critical-alerts'
  email_configs:
  - to: 'ops-team@example.com'
    subject: '[CRITICAL] {{ .GroupLabels.alertname }}'
    body: |
      {{ range .Alerts }}
      Alert: {{ .Annotations.summary }}
      Description: {{ .Annotations.description }}
      {{ end }}
  webhook_configs:
  - url: 'http://webhook:5001/critical'

- name: 'warning-alerts'
  email_configs:
  - to: 'ops-team@example.com'
    subject: '[WARNING] {{ .GroupLabels.alertname }}'
    body: |
      {{ range .Alerts }}
      Alert: {{ .Annotations.summary }}
      Description: {{ .Annotations.description }}
      {{ end }}

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'dev', 'instance']
EOF

    log "AlertManager configuration created"
}

# Create Docker Compose for monitoring stack
setup_monitoring_compose() {
    log "Creating monitoring Docker Compose..."

    cat > "$MONITORING_DIR/docker-compose.monitoring.yml" << 'EOF'
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: atest-prometheus
    restart: unless-stopped
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=200h'
      - '--web.enable-lifecycle'
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus:/etc/prometheus
      - prometheus_data:/prometheus
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: atest-grafana
    restart: unless-stopped
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin123
      - GF_USERS_ALLOW_SIGN_UP=false
      - GF_SECURITY_DISABLE_GRAVATAR=true
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
    networks:
      - monitoring

  alertmanager:
    image: prom/alertmanager:latest
    container_name: atest-alertmanager
    restart: unless-stopped
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager:/etc/alertmanager
      - alertmanager_data:/alertmanager
    networks:
      - monitoring

  node_exporter:
    image: prom/node-exporter:latest
    container_name: atest-node-exporter
    restart: unless-stopped
    ports:
      - "9100:9100"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.sysfs=/host/sys'
      - '--collector.filesystem.ignored-mount-points=^/(sys|proc|dev|host|etc)($$|/)'
    networks:
      - monitoring

  postgres_exporter:
    image: prometheuscommunity/postgres-exporter:latest
    container_name: atest-postgres-exporter
    restart: unless-stopped
    environment:
      - DATA_SOURCE_NAME=postgresql://testuser:testpass@postgres:5432/testdb?sslmode=disable
    ports:
      - "9187:9187"
    networks:
      - monitoring
      - atest-network

  redis_exporter:
    image: oliver006/redis_exporter:latest
    container_name: atest-redis-exporter
    restart: unless-stopped
    environment:
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=redispass
    ports:
      - "9121:9121"
    networks:
      - monitoring
      - atest-network

networks:
  monitoring:
    driver: bridge
  atest-network:
    external: true

volumes:
  prometheus_data:
  grafana_data:
  alertmanager_data:
EOF

    log "Monitoring Docker Compose created"
}

# Create monitoring startup script
create_startup_script() {
    log "Creating monitoring startup script..."

    cat > "$MONITORING_DIR/start-monitoring.sh" << 'EOF'
#!/bin/bash
# Start monitoring stack
set -euo pipefail

MONITORING_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { echo "[INFO] $*"; }
error() { echo "[ERROR] $*"; exit 1; }

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    error "Docker is not running"
fi

# Check if Docker Compose is available
if ! command -v docker-compose >/dev/null 2>&1; then
    error "Docker Compose is not installed"
fi

cd "$MONITORING_DIR"

log "Starting monitoring stack..."

# Create network if it doesn't exist
docker network create atest-network 2>/dev/null || true

# Start monitoring services
docker-compose -f docker-compose.monitoring.yml up -d

# Wait for services to be ready
log "Waiting for services to be ready..."
sleep 30

# Check service health
log "Checking service health..."

if curl -f http://localhost:9090 >/dev/null 2>&1; then
    log "✓ Prometheus is accessible at http://localhost:9090"
else
    error "✗ Prometheus health check failed"
fi

if curl -f http://localhost:3000 >/dev/null 2>&1; then
    log "✓ Grafana is accessible at http://localhost:3000"
    log "  Default login: admin / admin123"
else
    error "✗ Grafana health check failed"
fi

if curl -f http://localhost:9093 >/dev/null 2>&1; then
    log "✓ AlertManager is accessible at http://localhost:9093"
else
    error "✗ AlertManager health check failed"
fi

log "Monitoring stack started successfully!"
log ""
log "Access URLs:"
log "  Grafana:     http://localhost:3000 (admin/admin123)"
log "  Prometheus:  http://localhost:9090"
log "  AlertManager: http://localhost:9093"
log ""
EOF

    chmod +x "$MONITORING_DIR/start-monitoring.sh"

    log "Startup script created"
}

# Create monitoring stop script
create_stop_script() {
    log "Creating monitoring stop script..."

    cat > "$MONITORING_DIR/stop-monitoring.sh" << 'EOF'
#!/bin/bash
# Stop monitoring stack
set -euo pipefail

MONITORING_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { echo "[INFO] $*"; }

cd "$MONITORING_DIR"

log "Stopping monitoring stack..."

docker-compose -f docker-compose.monitoring.yml down -v

log "Monitoring stack stopped"
EOF

    chmod +x "$MONITORING_DIR/stop-monitoring.sh"

    log "Stop script created"
}

# Create monitoring README
create_readme() {
    log "Creating monitoring README..."

    cat > "$MONITORING_DIR/README.md" << 'EOF'
# Monitoring Stack for atest-ext-ai

This directory contains the monitoring setup for the atest-ext-ai plugin using Prometheus, Grafana, and AlertManager.

## Components

- **Prometheus**: Metrics collection and storage
- **Grafana**: Visualization and dashboards
- **AlertManager**: Alert routing and notification
- **Node Exporter**: System metrics
- **Postgres Exporter**: PostgreSQL metrics
- **Redis Exporter**: Redis metrics

## Quick Start

1. Start the monitoring stack:
   ```bash
   ./start-monitoring.sh
   ```

2. Access the services:
   - Grafana: http://localhost:3000 (admin/admin123)
   - Prometheus: http://localhost:9090
   - AlertManager: http://localhost:9093

3. Stop the monitoring stack:
   ```bash
   ./stop-monitoring.sh
   ```

## Configuration

### Prometheus
- Configuration: `prometheus/prometheus.yml`
- Alert rules: `prometheus/rules/`

### Grafana
- Dashboards: `grafana/dashboards/`
- Datasources: `grafana/provisioning/datasources/`

### AlertManager
- Configuration: `alertmanager/alertmanager.yml`

## Metrics

The atest-ext-ai plugin exposes the following metrics:

- `atest_ai_requests_total`: Total number of AI requests
- `atest_ai_request_duration_seconds`: Request duration histogram
- `atest_ai_confidence_score`: Confidence score histogram
- `atest_ai_active_connections`: Number of active connections
- `atest_ai_cache_hits_total`: Cache hits counter
- `atest_ai_cache_misses_total`: Cache misses counter
- `atest_ai_errors_total`: Error counter
- `atest_ai_memory_usage_bytes`: Memory usage gauge

## Alerts

The following alerts are configured:

- **PluginDown**: Plugin is unreachable
- **HighErrorRate**: Error rate exceeds threshold
- **LowConfidenceQueries**: Many queries with low confidence
- **MemoryUsageHigh**: Memory usage is high
- **RequestRateHigh**: Request rate is high
- **OllamaDown**: Ollama service is unreachable

## Customization

To customize the monitoring setup:

1. Edit configuration files in their respective directories
2. Restart the monitoring stack: `./stop-monitoring.sh && ./start-monitoring.sh`

## Troubleshooting

- Check service logs: `docker-compose -f docker-compose.monitoring.yml logs [service]`
- Verify network connectivity: `docker network ls`
- Check port availability: `netstat -tulpn | grep :3000`
EOF

    log "README created"
}

# Main setup function
main() {
    log "Setting up monitoring stack for atest-ext-ai plugin..."

    setup_directories
    setup_prometheus
    setup_grafana
    setup_alertmanager
    setup_monitoring_compose
    create_startup_script
    create_stop_script
    create_readme

    log "Monitoring stack setup completed!"
    info ""
    info "Next steps:"
    info "1. Start monitoring: cd $MONITORING_DIR && ./start-monitoring.sh"
    info "2. Access Grafana: http://localhost:3000 (admin/admin123)"
    info "3. Configure alerts: Edit alertmanager/alertmanager.yml"
    info ""
    info "For more information, see: $MONITORING_DIR/README.md"
}

# Run setup
main "$@"