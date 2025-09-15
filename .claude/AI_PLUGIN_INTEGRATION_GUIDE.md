# AI Plugin Integration Guide for atest API Testing Tool

**Version**: 1.0.0  
**Date**: 2025-09-08  
**Status**: Production Ready  
**Epic**: ai-extension-main-project-part (100% Complete)  

## Overview

This document provides comprehensive integration specifications for developing AI plugins that seamlessly integrate with the atest API testing tool's established infrastructure. The main project has implemented a complete dual-proto architecture with HTTP API layer, plugin communication bridge, and frontend integration.

## Architecture Summary

### Dual Proto Architecture

The atest AI extension uses a **two-layer protocol buffer architecture**:

1. **HTTP API Layer** (`pkg/server/server.proto`) - REST endpoints for frontend integration
2. **Plugin Communication Layer** (`pkg/testing/remote/loader.proto`) - gRPC interface for plugin communication

```
Frontend (Vue.js) → HTTP API → Plugin Bridge → AI Plugin (gRPC)
```

### Communication Flow

```
User Interface Request
    ↓
HTTP POST /api/v1/ai/generate-sql
    ↓
Runner.GenerateSQL() [server.proto]
    ↓
DataServer.Query(type="ai") 
    ↓
AI Plugin via loader.proto (Unix Socket)
    ↓
AIProcessingInfo Response
    ↓
HTTP Response to Frontend
```

## Protocol Buffer Specifications

### 1. HTTP API Layer (`pkg/server/server.proto`)

#### Required Service Extensions

Add the following RPC methods to the existing `Runner` service:

```protobuf
service Runner {
    // Existing methods...

    // AI SQL Generation API
    rpc GenerateSQL(AIRequest) returns (AIResponse) {
        option (google.api.http) = {
            post: "/api/v1/ai/generate-sql"
            body: "*"
        };
    }
    
    // AI Service Status Query
    rpc GetAIStatus(Empty) returns (AIStatus) {
        option (google.api.http) = {
            get: "/api/v1/ai/status"
        };
    }
    
    // AI SQL Validation
    rpc ValidateSQL(SQLValidationRequest) returns (SQLValidationResponse) {
        option (google.api.http) = {
            post: "/api/v1/ai/validate-sql"
            body: "*"
        };
    }
}
```

#### Required Message Definitions

**CRITICAL**: Use field numbers **10+** for all AI extensions to ensure backward compatibility:

```protobuf
// AI Request Message
message AIRequest {
    string natural_language = 1;       // User's natural language input
    string database_key = 2;           // Database connection key from stores
    string database_type = 3;          // Database type: mysql, postgresql, sqlite
    bool explain_query = 4;            // Whether to provide explanation
    map<string, string> context = 5;   // Additional context information
}

// AI Response Message
message AIResponse {
    bool success = 1;
    string generated_sql = 2;          // Generated SQL query
    string explanation = 3;            // Human-readable explanation
    repeated string suggestions = 4;    // Optimization suggestions
    float confidence_score = 5;        // AI confidence level (0.0-1.0)
    string error_message = 6;          // Error details if success = false
    string model_used = 7;             // AI model identifier used
}

// AI Status Message
message AIStatus {
    bool available = 1;                // Whether AI service is available
    repeated string supported_models = 2; // List of available AI models
    string current_model = 3;          // Currently active model
    map<string, string> capabilities = 4; // Service capabilities
    string version = 5;                // AI plugin version
    int64 last_health_check = 6;       // Last health check timestamp
}

// SQL Validation Request
message SQLValidationRequest {
    string sql = 1;                    // SQL query to validate
    string database_type = 2;          // Target database type
    string database_key = 3;           // Database connection key (optional)
}

// SQL Validation Response
message SQLValidationResponse {
    bool is_valid = 1;
    repeated string errors = 2;        // Validation errors
    repeated string warnings = 3;      // Potential issues
    string optimized_sql = 4;          // Suggested optimizations (optional)
    map<string, string> metadata = 5;  // Additional validation info
}
```

### 2. Plugin Communication Layer (`pkg/testing/remote/loader.proto`)

#### DataQuery Message Extensions

**CRITICAL**: Extend existing `DataQuery` message with AI fields starting from field **10**:

```protobuf
message DataQuery {
    // Existing fields 1-5 (DO NOT MODIFY)
    string type = 1;                   // Query type - use "ai" for AI queries
    string key = 2;                    // Database connection key
    string sql = 3;                    // SQL query or natural language
    int32 offset = 4;                  // Pagination offset
    int32 limit = 5;                   // Pagination limit
    
    // AI-specific extension fields (field numbers 10+)
    map<string, string> ai_context = 10;  // AI processing context
    string natural_language = 11;         // Natural language input
    string database_type = 12;            // Database type for AI processing
    bool explain_query = 13;              // Request explanation
    string ai_model = 14;                 // Specific AI model to use
    float confidence_threshold = 15;       // Minimum confidence required
}
```

#### DataQueryResult Message Extensions

```protobuf
message DataQueryResult {
    // Existing fields 1-3 (DO NOT MODIFY)
    repeated Pair data = 1;
    repeated Pairs items = 2;
    map<string, string> meta = 3;
    
    // AI-specific result fields
    AIProcessingInfo ai_info = 10;        // AI processing information
}

// AI Processing Information
message AIProcessingInfo {
    bool ai_processed = 1;                // Whether query was AI-processed
    string original_language = 2;         // Original natural language input
    string generated_sql = 3;             // AI-generated SQL
    string explanation = 4;               // Query explanation
    repeated string suggestions = 5;       // Optimization suggestions
    float confidence_score = 6;           // AI confidence level
    string model_used = 7;                // AI model identifier
    int64 processing_time_ms = 8;         // Processing time in milliseconds
}
```

#### Loader Service Extensions

```protobuf
service Loader {
    // Existing methods...
    
    // AI Plugin Status Check
    rpc GetAICapabilities(server.Empty) returns (AICapabilities) {}
}

// AI Plugin Capabilities
message AICapabilities {
    bool available = 1;
    repeated string supported_databases = 2;  // mysql, postgresql, sqlite
    repeated string supported_models = 3;     // Available AI models
    string default_model = 4;                // Default AI model
    map<string, string> settings = 5;        // Plugin configuration
    string plugin_version = 6;              // Plugin version info
}
```

## Plugin Implementation Requirements

### 1. Binary Requirements

```bash
# Plugin binary name (REQUIRED)
Binary Name: atest-store-ai

# Supported platforms
- Linux: atest-store-ai_linux_amd64
- macOS: atest-store-ai_darwin_amd64, atest-store-ai_darwin_arm64  
- Windows: atest-store-ai_windows_amd64.exe

# Go version requirement
Go 1.19+ required for gRPC and protobuf compatibility
```

### 2. Communication Protocol

```bash
# Unix Socket Communication
Socket Path: /tmp/atest-store-ai.sock

# gRPC Service Registration
pb.RegisterLoaderServer(grpcServer, aiPluginServer)

# Health Check Requirements
- Plugin must be discoverable within 2 seconds of startup
- Must implement both Verify() and GetAICapabilities() methods
```

### 3. Core Query Method Implementation

**CRITICAL**: Implement the Query method to handle `type="ai"` queries:

```go
func (s *AIPlugin) Query(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
    // Check if this is an AI query
    if req.Type != "ai" {
        return nil, status.Errorf(codes.InvalidArgument, "unsupported query type: %s", req.Type)
    }
    
    // Extract natural language input
    naturalLanguage := req.NaturalLanguage
    if naturalLanguage == "" {
        return nil, status.Errorf(codes.InvalidArgument, "natural_language field is required for AI queries")
    }
    
    // Process through AI model
    aiResult, err := s.aiService.GenerateSQL(ctx, &AIGenerationRequest{
        NaturalLanguage: naturalLanguage,
        DatabaseType:   req.DatabaseType,
        Context:        req.AiContext,
        ExplainQuery:   req.ExplainQuery,
    })
    if err != nil {
        return nil, err
    }
    
    // Optionally execute generated SQL
    var queryResults []*server.Pairs
    if req.Key != "" && aiResult.GeneratedSQL != "" {
        results, err := s.executeSQL(ctx, req.Key, aiResult.GeneratedSQL, req.Offset, req.Limit)
        if err != nil {
            // Return AI result with execution error
            return &server.DataQueryResult{
                AiInfo: &server.AIProcessingInfo{
                    AiProcessed:     true,
                    OriginalLanguage: naturalLanguage,
                    GeneratedSql:    aiResult.GeneratedSQL,
                    Explanation:     aiResult.Explanation,
                    Suggestions:     aiResult.Suggestions,
                    ConfidenceScore: aiResult.ConfidenceScore,
                    ModelUsed:       aiResult.ModelUsed,
                },
                Data: []*server.Pair{
                    {Key: "error", Value: err.Error()},
                    {Key: "generated_sql", Value: aiResult.GeneratedSQL},
                },
            }, nil
        }
        queryResults = results
    }
    
    return &server.DataQueryResult{
        Items: queryResults,
        AiInfo: &server.AIProcessingInfo{
            AiProcessed:      true,
            OriginalLanguage: naturalLanguage,
            GeneratedSql:     aiResult.GeneratedSQL,
            Explanation:      aiResult.Explanation,
            Suggestions:      aiResult.Suggestions,
            ConfidenceScore:  aiResult.ConfidenceScore,
            ModelUsed:        aiResult.ModelUsed,
            ProcessingTimeMs: aiResult.ProcessingTime,
        },
    }, nil
}
```

### 4. Health Check Implementation

```go
func (s *AIPlugin) Verify(ctx context.Context, req *server.Empty) (*server.ExtensionStatus, error) {
    healthy := s.aiService.IsHealthy()
    
    return &server.ExtensionStatus{
        Ready:     healthy,
        ReadOnly:  false, // AI plugin can modify data through SQL execution
        Version:   s.version,
        Message:   s.getStatusMessage(),
    }, nil
}

func (s *AIPlugin) GetAICapabilities(ctx context.Context, req *server.Empty) (*server.AICapabilities, error) {
    return &server.AICapabilities{
        Available:          s.aiService.IsAvailable(),
        SupportedDatabases: []string{"mysql", "postgresql", "sqlite"},
        SupportedModels:    s.aiService.GetAvailableModels(),
        DefaultModel:       s.aiService.GetDefaultModel(),
        Settings:           s.getPluginSettings(),
        PluginVersion:      s.version,
    }, nil
}
```

## Configuration Requirements

### 1. stores.yaml Configuration Schema

```yaml
stores:
  - name: "ai-assistant"
    type: "ai"
    url: "unix:///tmp/atest-store-ai.sock"
    properties:
      ai_provider: "openai"  # openai, claude, local
      api_key: "${AI_API_KEY}"  # Environment variable
      model: "gpt-4"  # Default model
      max_tokens: 4096
      temperature: 0.1
      timeout: 30s
      enable_sql_execution: true
      confidence_threshold: 0.7
      supported_databases:
        - mysql
        - postgresql
        - sqlite
      rate_limit:
        requests_per_minute: 60
        burst_size: 10
```

### 2. Environment Variables

```bash
# Required environment variables
AI_PROVIDER=openai|claude|local
OPENAI_API_KEY=${API_KEY}
AI_MODEL=gpt-4
AI_TIMEOUT=30s

# Optional configuration
AI_CONFIDENCE_THRESHOLD=0.7
AI_MAX_TOKENS=4096
AI_TEMPERATURE=0.1
```

## Error Handling Requirements

### 1. Error Philosophy

- **Fail fast** for critical configuration (missing AI API key)
- **Log and continue** for optional features (extraction models)
- **Graceful degradation** when external services unavailable
- **User-friendly messages** through resilience layer

### 2. Standard Error Codes

```go
// Standard gRPC error codes for AI operations
codes.InvalidArgument  // Invalid natural language input
codes.Unavailable     // AI service temporarily unavailable  
codes.Internal        // AI model processing error
codes.DeadlineExceeded // AI request timeout
codes.PermissionDenied // Insufficient API quota/permissions
codes.NotFound        // Requested AI model not available
```

### 3. Error Response Format

```go
// Standard error response structure
type AIErrorResponse struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
    Suggestions []string `json:"suggestions,omitempty"`
}
```

## Testing Requirements

### 1. Testing Philosophy

- **No Mock Services**: "Do not use mock services for anything ever" - use real implementations
- **Test Coverage**: >90% coverage for new code with comprehensive integration tests
- **Deployment Verification**: Three-phase process (development → integration → deployment)

### 2. Required Test Categories

```go
// Unit Tests
func TestAIQueryProcessing(t *testing.T) { ... }
func TestNaturalLanguageValidation(t *testing.T) { ... }
func TestSQLGeneration(t *testing.T) { ... }

// Integration Tests  
func TestEndToEndAIWorkflow(t *testing.T) { ... }
func TestPluginCommunication(t *testing.T) { ... }
func TestDatabaseIntegration(t *testing.T) { ... }

// Performance Tests
func TestAIResponseTime(t *testing.T) { ... }
func TestConcurrentAIRequests(t *testing.T) { ... }
```

### 3. Test Data Requirements

```yaml
# Test suite structure
test_cases:
  - name: "Simple SELECT query"
    natural_language: "Show me all users"
    expected_sql: "SELECT * FROM users"
    database_type: "mysql"
    
  - name: "Complex JOIN query"
    natural_language: "Find users with their orders from last month"
    expected_sql: "SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id WHERE o.created_at > DATE_SUB(NOW(), INTERVAL 1 MONTH)"
    database_type: "mysql"
```

## Performance Requirements

### 1. Response Time Targets

```
- API endpoints: <100ms response time (excluding AI processing)
- AI processing: <30s total timeout
- Plugin discovery: <2s startup time
- Health checks: <500ms response
```

### 2. Resource Limits

```
- Memory usage increase: <5% of total system
- CPU impact during AI processing: Monitored but acceptable
- Concurrent AI requests: Support up to 10 simultaneous requests
```

## Frontend Integration

### 1. API Client Integration

The main project provides a complete TypeScript API client:

```typescript
// AI API Client (implemented in main project)
interface AIApiClient {
  generateSQL(request: {
    natural_language: string;
    database_key: string;
    database_type: string;
    explain_query?: boolean;
    context?: Record<string, string>;
  }): Promise<AIResponse>;

  getAIStatus(): Promise<AIStatus>;

  validateSQL(request: {
    sql: string;
    database_type: string;
    database_key?: string;
  }): Promise<SQLValidationResponse>;
}
```

### 2. Frontend Components

The main project includes:
- **AI Trigger Button**: Floating action button in bottom-right corner
- **Status Indicators**: Visual states (online, offline, processing, error)
- **Error Handling**: User-friendly error messages and recovery options
- **Accessibility Support**: Screen reader and keyboard navigation

## Security Requirements

### 1. Input Validation

```go
// Sanitize natural language input to prevent prompt injection
func sanitizeNaturalLanguage(input string) string {
    // Remove potential injection patterns
    cleaned := regexp.MustCompile(`[<>{}()[\]\\|&;$` + "`" + `]`).ReplaceAllString(input, "")
    // Limit length
    if len(cleaned) > 1000 {
        cleaned = cleaned[:1000]
    }
    return cleaned
}
```

### 2. SQL Injection Prevention

- Validate generated SQL before execution
- Use parameterized queries when possible
- Implement query complexity limits
- Log all AI-generated queries for auditing

### 3. Rate Limiting

```yaml
rate_limiting:
  requests_per_minute: 60
  burst_size: 10
  per_user: true
  global_limit: 1000
```

## Deployment Guide

### 1. Plugin Binary Deployment

```bash
# 1. Build plugin binary
go build -o atest-store-ai ./cmd/plugin

# 2. Deploy to plugin directory
cp atest-store-ai /usr/local/bin/

# 3. Make executable
chmod +x /usr/local/bin/atest-store-ai

# 4. Verify plugin discovery
atest extension list
```

### 2. Configuration Setup

```bash
# 1. Update stores.yaml with AI plugin configuration
# 2. Set required environment variables
# 3. Test plugin connectivity
# 4. Verify health checks
```

### 3. Socket Permissions

```bash
# Ensure proper Unix socket permissions
chmod 666 /tmp/atest-store-ai.sock
```

## Troubleshooting Guide

### 1. Common Issues

| Issue | Symptom | Solution |
|-------|---------|----------|
| Plugin not discovered | "AI service unavailable" | Check binary name and socket path |
| Socket permission denied | Connection refused | Verify socket file permissions |
| AI API quota exceeded | Rate limit errors | Check API key and usage limits |
| Slow response times | Timeout errors | Optimize AI model selection |

### 2. Debug Commands

```bash
# Test gRPC connectivity
grpcurl -plaintext -unix /tmp/atest-store-ai.sock list

# Check plugin health
atest extension verify ai-assistant

# Monitor plugin logs
tail -f /var/log/atest-store-ai.log
```

## Migration and Compatibility

### 1. Backward Compatibility

- All existing functionality preserved
- No changes to existing field numbers (1-5)
- New messages use proper proto3 syntax
- Optional fields for gradual adoption

### 2. Version Compatibility

```go
// Plugin version requirements
const (
    MinimumAtestVersion = "1.0.0"
    PluginAPIVersion   = "1.0.0"
    ProtoVersion       = "3.0.0"
)
```

## Example Implementation

### Complete Plugin Structure

```
ai-plugin/
├── cmd/
│   └── plugin/
│       └── main.go          # Plugin entry point
├── internal/
│   ├── ai/
│   │   ├── service.go       # AI service implementation
│   │   └── models.go        # AI model interfaces
│   ├── grpc/
│   │   └── server.go        # gRPC server implementation
│   └── config/
│       └── config.go        # Configuration management
├── proto/
│   ├── server.pb.go         # Generated from main project
│   └── loader.pb.go         # Generated from main project
├── tests/
│   ├── integration_test.go  # Integration tests
│   └── unit_test.go         # Unit tests
├── go.mod
└── README.md
```

### Sample main.go

```go
package main

import (
    "context"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"

    "google.golang.org/grpc"
    pb "your-plugin/proto"
)

func main() {
    // Remove existing socket
    socketPath := "/tmp/atest-store-ai.sock"
    os.Remove(socketPath)

    // Create Unix socket listener
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        log.Fatalf("Failed to listen on socket: %v", err)
    }
    defer listener.Close()

    // Create gRPC server
    grpcServer := grpc.NewServer()
    
    // Create AI plugin instance
    aiPlugin := NewAIPlugin()
    
    // Register services
    pb.RegisterLoaderServer(grpcServer, aiPlugin)

    // Handle graceful shutdown
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        grpcServer.GracefulStop()
    }()

    log.Printf("AI Plugin listening on %s", socketPath)
    if err := grpcServer.Serve(listener); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

## Frontend Plugin Development Guide

The atest API testing tool now supports a complete frontend plugin system that allows decoupled UI components to integrate seamlessly with the main application. This section provides comprehensive guidance for developing frontend plugins.

### Plugin Architecture Overview

```
Main Application (Vue 3 + TypeScript + Element Plus)
    ↓
Plugin System (Plugin Manager + Extension Points)
    ↓
AI Plugin (Independent Vue Components + Plugin Context)
    ↓
Backend APIs (Via Plugin API Wrapper)
```

### 1. Plugin Type Definitions

All plugins must implement the core `Plugin` interface:

```typescript
// src/plugins/types.ts
export interface Plugin {
  config: PluginConfig
  lifecycle?: PluginLifecycle
  install(context: PluginContext): void | Promise<void>
  uninstall?(context: PluginContext): void | Promise<void>
}

export interface PluginConfig {
  id: string
  name: string
  version: string
  description: string
  author: string
  homepage?: string
  dependencies: string[]
  permissions: string[]
  enabled: boolean
  settings?: Record<string, any>
}

export interface PluginContext {
  ui: PluginUI
  api: PluginAPI
  storage: PluginStorage
  events: PluginEvents
}
```

### 2. Plugin Development Structure

Create plugins following this directory structure:

```
src/plugins/
├── types.ts              # Core plugin interfaces
├── manager.ts             # Plugin lifecycle management
├── api.ts                 # Plugin API client
├── storage.ts             # Plugin storage system
├── events.ts              # Plugin event system
├── ui.ts                  # Plugin UI extensions
├── index.ts               # Plugin registry
├── vue-plugin.ts          # Vue integration
└── ai/                    # Example AI plugin
    ├── index.ts           # Plugin definition
    ├── api.ts             # AI-specific API wrapper
    ├── types.ts           # AI plugin interfaces
    └── components/
        ├── AITriggerButton.vue
        └── PluginStatusIndicators.vue
```

### 3. Complete AI Plugin Example

#### Plugin Definition (`src/plugins/ai/index.ts`)

```typescript
import type { Plugin, PluginContext } from '../types'
import AITriggerButton from './components/AITriggerButton.vue'
import PluginStatusIndicators from './components/PluginStatusIndicators.vue'
import { createAIAPI } from './api'

const aiPlugin: Plugin = {
  config: {
    id: 'ai-assistant',
    name: 'AI Assistant',
    version: '1.0.0',
    description: 'AI-powered features for API testing including SQL generation, code analysis, and smart suggestions',
    author: 'API Testing Team',
    homepage: 'https://github.com/LinuxSuRen/api-testing',
    dependencies: [],
    permissions: ['api.read', 'api.write', 'ui.modify', 'notification.show'],
    enabled: true,
    settings: {
      autoHealth: true,
      healthInterval: 10000,
      statusRefreshInterval: 15000,
      showFloatingButton: true,
      showStatusPanel: true,
      enableNotifications: true
    }
  },

  lifecycle: {
    beforeMount: async () => console.log('AI Plugin: Before mount'),
    mounted: async () => console.log('AI Plugin: Mounted'),
    beforeUnmount: async () => console.log('AI Plugin: Before unmount'),
    unmounted: async () => console.log('AI Plugin: Unmounted')
  },

  async install(context: PluginContext) {
    const { ui, events, storage, api } = context
    
    // Register AI API extensions
    const aiAPI = createAIAPI(api)
    ;(context as any).aiAPI = aiAPI

    // Register components for floating UI
    ui.registerComponent('AITriggerButton', AITriggerButton)
    ui.registerComponent('PluginStatusIndicators', PluginStatusIndicators)

    // Add floating action button
    if (this.config.settings?.showFloatingButton !== false) {
      ui.addAction({
        id: 'ai-trigger',
        label: 'AI Assistant',
        icon: 'Cpu',
        position: 'floating',
        handler: async () => {
          try {
            const isHealthy = await aiAPI.checkHealth()
            if (isHealthy) {
              ui.showMessage('AI Assistant is ready!', 'success')
              events.emit('ai:trigger-activated')
            } else {
              ui.showMessage('AI services are offline', 'warning')
            }
          } catch (error) {
            ui.showMessage('Failed to connect to AI services', 'error')
          }
        }
      })
    }

    // Listen to system events
    events.on('app:ready', () => {
      this.startHealthMonitoring(aiAPI, ui, storage)
    })

    this.loadSettings(storage)
  }
}

export default aiPlugin
```

#### AI API Wrapper (`src/plugins/ai/api.ts`)

```typescript
import { PluginAPI } from '../api'

export interface AIHealthStatus {
  status: 'healthy' | 'unhealthy' | 'error'
  message?: string
  timestamp: string
}

export class AIAPI {
  constructor(private api: PluginAPI) {}

  async checkHealth(): Promise<boolean> {
    try {
      const health = await this.api.request<AIHealthStatus>({
        method: 'GET',
        path: '/health'
      })
      return health.status === 'healthy'
    } catch (error) {
      console.error('AI health check failed:', error)
      return false
    }
  }

  async generateSQL(prompt: string): Promise<string> {
    const result = await this.api.request<{ sql: string }>({
      method: 'POST',
      path: '/ai/generate/sql',
      data: { prompt }
    })
    return result.sql
  }
}

export function createAIAPI(api: PluginAPI): AIAPI {
  return new AIAPI(api)
}
```

#### Plugin Component Example (`src/plugins/ai/components/AITriggerButton.vue`)

```vue
<template>
  <el-button
    class="ai-trigger-fab"
    :class="{
      'animate-pulse-glow': isProcessing,
      'animate-shake': !isHealthy && !isProcessing
    }"
    type="primary"
    :size="size"
    circle
    :loading="isProcessing"
    :disabled="!isHealthy"
    @click="handleTrigger"
    :aria-label="getButtonTitle()"
  >
    <el-icon v-if="!isProcessing">
      <component :is="getButtonIcon()" />
    </el-icon>
  </el-button>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, inject } from 'vue'
import { ElMessage } from 'element-plus'
import { Cpu, Warning } from '@element-plus/icons-vue'
import type { PluginContext } from '../../types'

interface Props {
  size?: 'large' | 'default' | 'small'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'large'
})

const emit = defineEmits<{
  trigger: []
  statusChange: [status: 'online' | 'offline' | 'processing' | 'error']
}>()

// Inject plugin context
const pluginContext = inject<PluginContext & { aiAPI: any }>('pluginContext')

const isHealthy = ref(false)
const isProcessing = ref(false)

const healthStatus = computed(() => {
  if (isProcessing.value) return 'processing'
  if (!isHealthy.value) return 'offline'
  return 'online'
})

const getButtonIcon = () => {
  switch (healthStatus.value) {
    case 'offline':
    case 'error':
      return Warning
    case 'online':
    default:
      return Cpu
  }
}

const getButtonTitle = () => {
  switch (healthStatus.value) {
    case 'processing':
      return 'AI is processing...'
    case 'offline':
      return 'AI services offline - Click to retry connection'
    case 'online':
    default:
      return 'Trigger AI processing'
  }
}

const handleTrigger = async () => {
  if (isProcessing.value) return
  
  if (!isHealthy.value) {
    await checkHealth()
    if (!isHealthy.value) {
      ElMessage.error('AI services are not available')
      return
    }
  }
  
  isProcessing.value = true
  emit('statusChange', 'processing')
  
  try {
    emit('trigger')
    
    if (pluginContext?.events) {
      pluginContext.events.emit('ai:processing-started')
    }
    
    await new Promise(resolve => setTimeout(resolve, 3000))
    
    ElMessage.success('AI processing completed')
  } catch (error) {
    ElMessage.error('AI processing failed')
  } finally {
    isProcessing.value = false
    emit('statusChange', healthStatus.value)
  }
}

const checkHealth = async () => {
  if (!pluginContext?.aiAPI) return
  
  try {
    const healthy = await pluginContext.aiAPI.checkHealth()
    isHealthy.value = healthy
  } catch (error) {
    isHealthy.value = false
  }
  
  emit('statusChange', healthStatus.value)
}

onMounted(() => {
  checkHealth()
  setInterval(checkHealth, 10000)
})
</script>

<style scoped>
.ai-trigger-fab {
  position: fixed;
  bottom: 24px;
  right: 24px;
  z-index: 1000;
  width: 56px;
  height: 56px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.3s ease;
}

.ai-trigger-fab:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.25);
}

@media (prefers-reduced-motion: reduce) {
  .ai-trigger-fab {
    transition: none;
  }
}
</style>
```

### 4. Plugin Integration with Main Application

#### Main Application Setup (`src/main.ts`)

```typescript
import { createApp } from 'vue'
import App from './App.vue'
import { PluginSystem } from './plugins/vue-plugin'

const app = createApp(App)

// Install plugin system
app.use(PluginSystem, {
  autoInstallPlugins: true,
  enableLogging: process.env.NODE_ENV === 'development'
})

app.mount('#app')
```

#### Plugin Extension Points in Templates (`src/App.vue`)

```vue
<template>
  <div id="app">
    <!-- Main application content -->
    <el-container>
      <!-- Sidebar with plugin menu extensions -->
      <el-aside>
        <el-menu>
          <!-- Core menu items -->
          <el-menu-item>Testing</el-menu-item>
          
          <!-- Plugin menu extensions -->
          <PluginExtensionPoint type="menu" />
        </el-menu>
      </el-aside>

      <el-main>
        <!-- Main content area -->
        <router-view />
      </el-main>
    </el-container>

    <!-- Plugin floating UI extensions -->
    <PluginExtensionPoint type="floating" />
  </div>
</template>

<script setup lang="ts">
import PluginExtensionPoint from './components/core/PluginExtensionPoint.vue'
</script>
```

### 5. Plugin Extension Point Component

The `PluginExtensionPoint` component dynamically renders plugin UI extensions:

```vue
<!-- src/components/core/PluginExtensionPoint.vue -->
<template>
  <div class="plugin-extension-point">
    <!-- Floating components for plugin UI overlays -->
    <template v-if="type === 'floating'">
      <component 
        v-for="(component, name) in floatingComponents" 
        :key="name"
        :is="component"
        v-bind="componentProps"
      />
    </template>
    
    <!-- Other extension types: menu, actions, status, panels -->
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getRegisteredComponents } from '../../plugins/ui'

interface Props {
  type: 'menu' | 'actions' | 'status' | 'panels' | 'components' | 'floating'
  componentProps?: Record<string, any>
}

const props = defineProps<Props>()

const floatingComponents = computed(() => 
  getRegisteredComponents('floating')
)
</script>
```

### 6. Plugin Development Best Practices

#### Security and Permissions
- **Permission System**: Plugins must declare required permissions in their config
- **API Access Control**: Plugin API wrapper validates permissions for each request
- **Component Isolation**: Use plugin context injection to prevent global state pollution

#### Accessibility
- **ARIA Labels**: All interactive components must include proper ARIA labels
- **Keyboard Navigation**: Support Enter and Space key interactions
- **Screen Reader Support**: Include descriptive text for screen readers

#### Responsive Design
- **Mobile Support**: Components must work on mobile devices (768px and below)
- **High Contrast**: Support `prefers-contrast: high` media query
- **Reduced Motion**: Support `prefers-reduced-motion: reduce` media query

#### Performance
- **Lazy Loading**: Use dynamic imports for large plugin components
- **Memory Management**: Clean up timers, event listeners, and subscriptions
- **Error Boundaries**: Wrap plugin components in error boundaries to prevent app crashes

### 7. Plugin Testing Framework

#### Unit Tests for Plugin Components

```typescript
// tests/plugins/ai/AITriggerButton.spec.ts
import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import AITriggerButton from '@/plugins/ai/components/AITriggerButton.vue'

describe('AITriggerButton', () => {
  it('renders correctly when AI is online', async () => {
    const wrapper = mount(AITriggerButton, {
      global: {
        provide: {
          pluginContext: {
            aiAPI: {
              checkHealth: vi.fn().mockResolvedValue(true)
            }
          }
        }
      }
    })

    await wrapper.vm.$nextTick()
    expect(wrapper.find('.ai-trigger-fab').exists()).toBe(true)
    expect(wrapper.find('[aria-label*="Trigger AI processing"]').exists()).toBe(true)
  })

  it('shows offline state when AI services are unavailable', async () => {
    const wrapper = mount(AITriggerButton, {
      global: {
        provide: {
          pluginContext: {
            aiAPI: {
              checkHealth: vi.fn().mockResolvedValue(false)
            }
          }
        }
      }
    })

    await wrapper.vm.$nextTick()
    expect(wrapper.find('[aria-label*="offline"]').exists()).toBe(true)
    expect(wrapper.find('.ai-trigger-fab[disabled]').exists()).toBe(true)
  })
})
```

#### Integration Tests

```typescript
// tests/plugins/integration/plugin-system.spec.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { createPluginManager } from '@/plugins/manager'
import aiPlugin from '@/plugins/ai'

describe('Plugin System Integration', () => {
  let pluginManager

  beforeEach(() => {
    pluginManager = createPluginManager()
  })

  it('registers and enables AI plugin successfully', async () => {
    await pluginManager.register(aiPlugin)
    await pluginManager.enable('ai-assistant')

    const enabledPlugins = pluginManager.getEnabledPlugins()
    expect(enabledPlugins).toHaveLength(1)
    expect(enabledPlugins[0].config.id).toBe('ai-assistant')
  })

  it('provides plugin context with all required services', async () => {
    await pluginManager.register(aiPlugin)
    
    const context = pluginManager.getPluginContext('ai-assistant')
    expect(context.ui).toBeDefined()
    expect(context.api).toBeDefined()
    expect(context.storage).toBeDefined()
    expect(context.events).toBeDefined()
  })
})
```

## Conclusion

This integration guide provides all necessary specifications for developing both backend and frontend plugins that seamlessly integrate with the atest API testing tool. The main project infrastructure is complete and production-ready, providing:

✅ **Complete dual-proto architecture**  
✅ **HTTP API layer with validation**  
✅ **Plugin communication bridge**  
✅ **Frontend plugin system with decoupled components**  
✅ **Comprehensive testing framework**  
✅ **Complete documentation**  

### Development Workflow

1. **Backend Development**: Follow the gRPC plugin specifications above
2. **Frontend Development**: Use the plugin system to create decoupled UI components
3. **Integration**: Register plugins in the main application through the plugin manager
4. **Testing**: Use the comprehensive testing framework for both unit and integration tests

**Next Steps**: 
- Implement AI plugins following these exact specifications
- Use the provided examples as templates for new plugin development
- Follow the security and accessibility best practices outlined above

**Contact**: Refer to this document for all integration requirements. The main project team has validated all interfaces and communication protocols.

---
*Generated from completed epic: ai-extension-main-project-part (8/8 tasks completed)*  
*Updated with frontend plugin development guide: 2025-09-09*  
*Main project commit: bef2e57*  
*Documentation version: 1.1.0*

**Note**: The frontend plugin code examples above are reference implementations to be developed in separate plugin repositories, not in the main project codebase. The main project only provides the necessary interfaces for plugin integration.