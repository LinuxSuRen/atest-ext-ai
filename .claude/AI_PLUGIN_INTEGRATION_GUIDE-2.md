# AI Plugin Integration Guide

This guide provides comprehensive information for developing and integrating AI plugins with the API Testing platform.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Plugin Development](#plugin-development)
4. [Integration APIs](#integration-apis)
5. [Frontend Integration](#frontend-integration)
6. [Testing](#testing)
7. [Deployment](#deployment)
8. [Troubleshooting](#troubleshooting)

## Overview

The API Testing platform supports AI plugins through a decoupled architecture where:

- **Main Project**: Provides plugin management interfaces and integration points
- **AI Plugins**: Separate repositories/codebases that implement specific AI functionality
- **Communication**: Unix socket-based communication between main system and plugins

### Key Benefits

- **Decoupled Architecture**: Plugins are independent of main system
- **Language Agnostic**: Plugins can be written in any language supporting Unix sockets
- **Hot Reload**: Plugins can be added/removed without system restart
- **Health Monitoring**: Automatic health checking and status reporting

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Main API Testing System                  │
├─────────────────────┬───────────────────┬───────────────────┤
│   Frontend UI       │   ExtManager      │   HTTP API        │
│                     │                   │                   │
│ ┌─────────────────┐ │ ┌───────────────┐ │ ┌───────────────┐ │
│ │ AI Trigger      │ │ │ Plugin        │ │ │ /api/v1/ai/   │ │
│ │ Button          │ │ │ Registry      │ │ │ endpoints     │ │
│ │                 │ │ │               │ │ │               │ │
│ │ Status          │ │ │ Health        │ │ │ - discover    │ │
│ │ Indicators      │ │ │ Monitor       │ │ │ - health      │ │
│ └─────────────────┘ │ │               │ │ │ - register    │ │
│                     │ │ Lifecycle     │ │ │ - unregister  │ │
│                     │ │ Management    │ │ │               │ │
│                     │ └───────────────┘ │ └───────────────┘ │
└─────────────────────┴───────────────────┴───────────────────┘
                              │
                    Unix Socket Communication
                              │
┌─────────────────────────────────────────────────────────────┐
│                      AI Plugins                             │
├───────────────┬───────────────┬───────────────┬─────────────┤
│  SQL AI       │  Code AI      │  NLP AI       │   Custom    │
│  Plugin       │  Plugin       │  Plugin       │   AI Plugin │
│               │               │               │             │
│ - Generate    │ - Code        │ - Text        │ - Domain    │
│   SQL         │   Analysis    │   Analysis    │   Specific  │
│ - Optimize    │ - Bug         │ - Language    │   AI Tasks  │
│   Queries     │   Detection   │   Detection   │             │
│               │ - Refactor    │ - Sentiment   │             │
│               │   Suggest     │   Analysis    │             │
└───────────────┴───────────────┴───────────────┴─────────────┘
```

### Communication Flow

1. **Plugin Registration**: AI plugin registers with ExtManager via HTTP API
2. **Health Monitoring**: ExtManager periodically checks plugin health via socket
3. **User Interaction**: User triggers AI functionality via frontend UI
4. **Request Routing**: Main system routes requests to appropriate AI plugin
5. **Response Handling**: Plugin responses are processed and returned to user

## Plugin Development

### Plugin Interface Requirements

All AI plugins must implement the following interface:

```go
type AIPluginInterface interface {
    // Health check endpoint
    HealthCheck() (*HealthStatus, error)
    
    // Process AI request
    ProcessRequest(request *AIRequest) (*AIResponse, error)
    
    // Get plugin capabilities
    GetCapabilities() []string
    
    // Initialize plugin
    Initialize(config *PluginConfig) error
    
    // Shutdown plugin gracefully
    Shutdown() error
}
```

### Plugin Metadata Structure

```go
type AIPluginInfo struct {
    Name         string            `json:"name"`
    Version      string            `json:"version"`
    Description  string            `json:"description"`
    Capabilities []string          `json:"capabilities"`
    SocketPath   string            `json:"socketPath"`
    Metadata     map[string]string `json:"metadata"`
}
```

### Example Plugin Implementation

#### Go Plugin Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"
)

type SQLGeneratorPlugin struct {
    socket net.Listener
}

type AIRequest struct {
    Type    string                 `json:"type"`
    Data    map[string]interface{} `json:"data"`
    Context map[string]string      `json:"context"`
}

type AIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func (p *SQLGeneratorPlugin) ProcessRequest(req *AIRequest) (*AIResponse, error) {
    switch req.Type {
    case "generate_sql":
        // Implement SQL generation logic
        sql := p.generateSQL(req.Data["description"].(string))
        return &AIResponse{
            Success: true,
            Data: map[string]interface{}{
                "sql": sql,
                "explanation": "Generated SQL query based on description",
            },
        }, nil
    default:
        return &AIResponse{
            Success: false,
            Error:   "Unknown request type",
        }, nil
    }
}

func (p *SQLGeneratorPlugin) generateSQL(description string) string {
    // Implement your AI logic here
    // This could integrate with OpenAI, local models, etc.
    return "SELECT * FROM users WHERE active = true;"
}

func main() {
    plugin := &SQLGeneratorPlugin{}
    
    socketPath := "/tmp/sql-generator-plugin.sock"
    os.Remove(socketPath) // Clean up any existing socket
    
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        log.Fatal("Failed to create socket:", err)
    }
    plugin.socket = listener
    
    // Register with main system
    err = registerPlugin()
    if err != nil {
        log.Fatal("Failed to register plugin:", err)
    }
    
    // Handle shutdown gracefully
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-c
        plugin.Shutdown()
        os.Exit(0)
    }()
    
    // Start serving requests
    plugin.serve()
}
```

#### Python Plugin Example

```python
#!/usr/bin/env python3
import socket
import json
import os
import threading
import signal
import sys
import requests

class NLPAnalysisPlugin:
    def __init__(self, socket_path="/tmp/nlp-analysis-plugin.sock"):
        self.socket_path = socket_path
        self.sock = None
        self.running = True
        
    def process_request(self, request):
        """Process AI request and return response"""
        req_type = request.get('type')
        
        if req_type == 'analyze_text':
            return self.analyze_text(request['data'])
        elif req_type == 'detect_language':
            return self.detect_language(request['data'])
        else:
            return {
                'success': False,
                'error': f'Unknown request type: {req_type}'
            }
    
    def analyze_text(self, data):
        """Analyze text for sentiment, entities, etc."""
        text = data.get('text', '')
        
        # Implement your NLP analysis logic here
        # This could use spaCy, NLTK, transformers, etc.
        
        return {
            'success': True,
            'data': {
                'sentiment': 'positive',
                'confidence': 0.85,
                'entities': ['API', 'testing'],
                'language': 'english'
            }
        }
    
    def detect_language(self, data):
        """Detect language of input text"""
        text = data.get('text', '')
        
        # Implement language detection logic
        return {
            'success': True,
            'data': {
                'language': 'english',
                'confidence': 0.95
            }
        }
    
    def health_check(self):
        """Return health status"""
        return {
            'status': 'online',
            'timestamp': time.time(),
            'version': '1.0.0'
        }
    
    def register_plugin(self):
        """Register plugin with main system"""
        plugin_info = {
            'name': 'nlp-analysis-plugin',
            'version': '1.0.0',
            'description': 'Natural Language Processing analysis plugin',
            'capabilities': ['text-analysis', 'language-detection', 'sentiment-analysis'],
            'socketPath': f'unix://{self.socket_path}',
            'metadata': {
                'author': 'AI Team',
                'type': 'ai'
            }
        }
        
        try:
            response = requests.post(
                'http://localhost:8080/api/v1/ai/plugins/register',
                json=plugin_info
            )
            response.raise_for_status()
            print(f"Plugin registered successfully: {response.json()}")
        except requests.exceptions.RequestException as e:
            print(f"Failed to register plugin: {e}")
            sys.exit(1)
    
    def start(self):
        """Start the plugin server"""
        # Remove existing socket
        try:
            os.unlink(self.socket_path)
        except OSError:
            pass
        
        # Create Unix socket
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.bind(self.socket_path)
        self.sock.listen(5)
        
        # Register with main system
        self.register_plugin()
        
        print(f"NLP Analysis Plugin listening on {self.socket_path}")
        
        while self.running:
            try:
                conn, addr = self.sock.accept()
                threading.Thread(target=self.handle_connection, args=(conn,)).start()
            except socket.error:
                break
    
    def handle_connection(self, conn):
        """Handle incoming connection"""
        try:
            data = conn.recv(4096)
            if data:
                request = json.loads(data.decode())
                response = self.process_request(request)
                conn.send(json.dumps(response).encode())
        except Exception as e:
            error_response = {
                'success': False,
                'error': str(e)
            }
            conn.send(json.dumps(error_response).encode())
        finally:
            conn.close()
    
    def shutdown(self):
        """Shutdown plugin gracefully"""
        self.running = False
        if self.sock:
            self.sock.close()
        try:
            os.unlink(self.socket_path)
        except OSError:
            pass

def signal_handler(sig, frame):
    print("Shutting down plugin...")
    plugin.shutdown()
    sys.exit(0)

if __name__ == "__main__":
    plugin = NLPAnalysisPlugin()
    
    # Handle shutdown signals
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    plugin.start()
```

## Integration APIs

### HTTP Endpoints

The main system provides the following HTTP endpoints for AI plugin management:

#### Plugin Registration
```http
POST /api/v1/ai/plugins/register
Content-Type: application/json

{
    "name": "my-ai-plugin",
    "version": "1.0.0",
    "description": "My custom AI plugin",
    "capabilities": ["text-analysis", "code-generation"],
    "socketPath": "unix:///tmp/my-ai-plugin.sock",
    "metadata": {
        "author": "Developer Name",
        "type": "ai"
    }
}
```

#### Plugin Discovery
```http
GET /api/v1/ai/plugins/discover

Response:
{
    "success": true,
    "data": [
        {
            "name": "my-ai-plugin",
            "version": "1.0.0",
            "capabilities": ["text-analysis"]
        }
    ]
}
```

#### Health Check (Individual)
```http
GET /api/v1/ai/plugins/{name}/health

Response:
{
    "success": true,
    "data": {
        "name": "my-ai-plugin",
        "status": "online",
        "lastCheckAt": "2025-09-10T10:00:00Z",
        "responseTime": 45
    }
}
```

#### Health Check (All Plugins)
```http
GET /api/v1/ai/plugins/health

Response:
{
    "success": true,
    "data": {
        "plugin-1": {
            "status": "online",
            "lastCheckAt": "2025-09-10T10:00:00Z"
        },
        "plugin-2": {
            "status": "offline",
            "errorMessage": "Plugin socket not found"
        }
    }
}
```

#### Plugin Unregistration
```http
DELETE /api/v1/ai/plugins/{name}

Response:
{
    "success": true,
    "message": "Plugin unregistered successfully"
}
```

### Frontend Integration

The main system provides Vue.js components for AI plugin integration:

#### AI Status Indicator Component
```vue
<template>
  <div class="ai-status-indicator">
    <AIStatusIndicator />
  </div>
</template>
```

#### AI Trigger Button Component
```vue
<template>
  <div class="ai-trigger">
    <AITriggerButton @ai-trigger-clicked="handleAITrigger" />
  </div>
</template>
```

### JavaScript API Client

```javascript
import { API } from './net'

// Discover available AI plugins
API.DiscoverAIPlugins((plugins) => {
    console.log('Available AI plugins:', plugins)
})

// Check plugin health
API.CheckAIPluginHealth('my-plugin', (health) => {
    console.log('Plugin health:', health)
})

// Register new plugin
const pluginInfo = {
    name: 'my-custom-plugin',
    version: '1.0.0',
    socketPath: 'unix:///tmp/my-plugin.sock'
}

API.RegisterAIPlugin(pluginInfo, (response) => {
    console.log('Plugin registered:', response)
})
```

## Testing

### Running Integration Tests

```bash
# Run all AI integration tests
cd pkg/server
go test -v -run TestAIIntegration

# Run HTTP API tests
go test -v -run TestAIPluginHTTP

# Run performance benchmarks
go test -bench=BenchmarkAIPlugin -benchmem

# Run performance test suite
chmod +x scripts/ai_performance_test.sh
./scripts/ai_performance_test.sh
```

### Frontend Component Tests

```bash
cd console/atest-ui
npm test -- ai-components.spec.ts
```

### Mock Plugin for Testing

A mock plugin is provided for testing purposes:

```go
type MockAIPlugin struct {
    socketPath    string
    responseDelay time.Duration
    shouldError   bool
}

func NewMockAIPlugin(socketPath string) *MockAIPlugin {
    return &MockAIPlugin{
        socketPath:    socketPath,
        responseDelay: 100 * time.Millisecond,
        shouldError:   false,
    }
}
```

## Deployment

### Plugin Deployment Steps

1. **Build Plugin**: Compile your plugin for the target platform
2. **Deploy Binary**: Copy plugin binary to target server
3. **Configure Socket**: Ensure socket path is accessible
4. **Start Plugin**: Start plugin process
5. **Register Plugin**: Plugin auto-registers with main system
6. **Verify Health**: Check plugin appears in health dashboard

### Docker Deployment Example

```dockerfile
FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o sql-ai-plugin .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/sql-ai-plugin .
CMD ["./sql-ai-plugin"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sql-ai-plugin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sql-ai-plugin
  template:
    metadata:
      labels:
        app: sql-ai-plugin
    spec:
      containers:
      - name: sql-ai-plugin
        image: my-registry/sql-ai-plugin:v1.0.0
        volumeMounts:
        - name: plugin-sockets
          mountPath: /tmp
      volumes:
      - name: plugin-sockets
        hostPath:
          path: /tmp/ai-plugins
```

## Troubleshooting

### Common Issues

#### Plugin Not Detected
- **Cause**: Socket file not created or wrong permissions
- **Solution**: Check socket path and file permissions

#### Health Check Fails
- **Cause**: Plugin process crashed or socket blocked
- **Solution**: Check plugin logs and restart process

#### Performance Issues
- **Cause**: Plugin taking too long to respond
- **Solution**: Optimize plugin code, check resource usage

### Debugging Commands

```bash
# Check if socket exists
ls -la /tmp/my-plugin.sock

# Test socket connectivity
nc -U /tmp/my-plugin.sock

# Monitor plugin health
curl http://localhost:8080/api/v1/ai/plugins/health

# Check plugin logs
docker logs my-ai-plugin

# Performance monitoring
./scripts/ai_performance_test.sh
```

### Log Analysis

Plugin logs should include:
- Registration attempts and results
- Health check responses
- Request processing times
- Error details with stack traces

### Performance Monitoring

Key metrics to monitor:
- Response time (<100ms for trigger, <500ms for health)
- CPU overhead (<5% system impact)
- Memory usage (<10% overhead)
- Error rates and types

## Best Practices

### Plugin Development
1. **Stateless Design**: Keep plugins stateless when possible
2. **Error Handling**: Implement comprehensive error handling
3. **Logging**: Add detailed logging for debugging
4. **Resource Management**: Properly manage memory and file handles
5. **Graceful Shutdown**: Handle shutdown signals properly

### Performance Optimization
1. **Connection Pooling**: Reuse connections when possible
2. **Caching**: Cache frequently requested data
3. **Async Processing**: Use asynchronous operations for long tasks
4. **Resource Limits**: Set appropriate memory/CPU limits

### Security Considerations
1. **Input Validation**: Validate all input data
2. **Socket Permissions**: Set appropriate socket file permissions
3. **Process Isolation**: Run plugins in isolated environments
4. **Regular Updates**: Keep dependencies updated

## Contributing

To contribute to the AI plugin ecosystem:

1. Fork the main repository
2. Create your plugin in a separate repository
3. Follow the plugin interface specifications
4. Add comprehensive tests
5. Submit plugin registry entry
6. Provide documentation and examples

For questions and support, please refer to the project documentation or create an issue in the main repository.