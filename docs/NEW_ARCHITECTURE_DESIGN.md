# atest-ext-ai 新架构设计文档

**版本**: 2.0
**日期**: 2025-10-10
**状态**: 设计方案
**目的**: 详细说明新架构的设计理念、组件结构和实现细节

---

## 📐 设计理念

### 核心原则

1. **KISS (Keep It Simple, Stupid)**
   - 避免不必要的抽象
   - 代码简洁明了
   - 易于理解和维护

2. **YAGNI (You Aren't Gonna Need It)**
   - 不实现未来可能需要的功能
   - 聚焦当前实际需求
   - 需要时再扩展

3. **最小必要原则**
   - 只实现必须的功能
   - 移除冗余代码
   - 优化资源使用

4. **直接依赖**
   - 减少抽象层级
   - 使用具体类型
   - 降低间接引用

---

## 🏗️ 整体架构

### 架构分层

```
┌─────────────────────────────────────────────────────────┐
│                     gRPC Layer                          │
│                  (Plugin Service)                       │
│                                                         │
│  • Query: AI 生成请求                                    │
│  • Verify: 健康检查                                      │
│  • GetMenus: UI 菜单                                     │
│  • GetPageOfJS/CSS: UI 资源                             │
└───────────────────┬────────────────────────┬────────────┘
                    │                        │
                    ▼                        ▼
┌───────────────────────────┐  ┌──────────────────────────┐
│      Business Layer       │  │     Config Layer         │
│     (AI Engine)           │  │  (Simple Loader)         │
│                           │  │                          │
│  • SQL Generation         │  │  • File Loading          │
│  • Request Processing     │  │  • Env Override          │
│  • Response Formatting    │  │  • Validation            │
└─────────────┬─────────────┘  └──────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────┐
│                   Core Layer                            │
│                  (AIManager)                            │
│                                                         │
│  • Client Management                                    │
│  • Provider Discovery                                   │
│  • Model Listing                                        │
│  • Connection Testing                                   │
│  • Health Checking (on-demand)                          │
│  • Retry Logic (inline)                                 │
└───────────────────┬─────────────────────────────────────┘
                    │
     ┌──────────────┼──────────────┐
     │              │               │
     ▼              ▼               ▼
┌─────────┐   ┌──────────┐   ┌───────────┐
│ OpenAI  │   │Universal │   │ Discovery │
│ Client  │   │  Client  │   │  Service  │
│         │   │          │   │           │
│ (SDK)   │   │ (HTTP)   │   │ (Ollama)  │
└─────────┘   └──────────┘   └───────────┘
```

### 数据流

```
前端请求
   │
   ├─> gRPC: Generate SQL
   │   └─> Plugin Service
   │       └─> AI Engine
   │           └─> SQLGenerator
   │               └─> AIManager.Generate
   │                   └─> [Retry Logic]
   │                       └─> AIClient (OpenAI/Ollama)
   │                           └─> AI Service API
   │
   ├─> gRPC: Get Models
   │   └─> Plugin Service
   │       └─> AIManager.GetModels
   │           └─> AIClient.GetCapabilities
   │               └─> AI Service API
   │
   └─> gRPC: Test Connection
       └─> Plugin Service
           └─> AIManager.TestConnection
               └─> Create temp client
                   └─> HealthCheck
```

---

## 🧩 核心组件设计

### 1. AIManager（统一管理器）

**职责**: 统一管理所有 AI 客户端的生命周期和交互

**接口**:

```go
package ai

type AIManager struct {
    clients   map[string]interfaces.AIClient
    config    config.AIConfig
    discovery *discovery.OllamaDiscovery
    mu        sync.RWMutex
}

// 核心方法
func NewAIManager(cfg config.AIConfig) (*AIManager, error)
func (m *AIManager) Generate(ctx context.Context, req *Request) (*Response, error)
func (m *AIManager) GetModels(ctx context.Context, provider string) ([]ModelInfo, error)
func (m *AIManager) DiscoverProviders(ctx context.Context) ([]*ProviderInfo, error)
func (m *AIManager) TestConnection(ctx context.Context, cfg *Config) (*Result, error)
func (m *AIManager) HealthCheck(ctx context.Context, provider string) (*HealthStatus, error)
func (m *AIManager) HealthCheckAll(ctx context.Context) map[string]*HealthStatus
func (m *AIManager) Close() error
```

**设计要点**:

1. **客户端池管理**
   ```go
   // 懒加载 + 缓存
   clients map[string]interfaces.AIClient
   ```

2. **线程安全**
   ```go
   // RWMutex 保护并发访问
   mu sync.RWMutex
   ```

3. **生命周期管理**
   ```go
   // 初始化
   func (m *AIManager) initializeClients() error

   // 添加客户端
   func (m *AIManager) addClient(name string, config Config) error

   // 移除客户端
   func (m *AIManager) removeClient(name string) error

   // 清理
   func (m *AIManager) Close() error
   ```

4. **智能选择**
   ```go
   // 选择健康的客户端
   func (m *AIManager) selectHealthyClient() interfaces.AIClient {
       // 1. 优先使用默认服务
       // 2. 降级使用备选服务
       // 3. 返回任意可用客户端
   }
   ```

---

### 2. Simple Config Loader（简化配置）

**职责**: 加载、验证和管理配置

**流程**:

```
1. Load File
   ├─> Check: config.yaml
   ├─> Check: ~/.config/atest/config.yaml
   └─> Check: /etc/atest/config.yaml

2. Apply Env
   └─> ATEST_EXT_AI_* 环境变量覆盖

3. Apply Defaults
   └─> 使用内置默认值填充

4. Validate
   ├─> 验证必填字段
   ├─> 验证数据类型
   └─> 验证取值范围
```

**实现**:

```go
package config

func LoadConfig() (*Config, error) {
    // 1. 加载文件
    cfg, err := loadConfigFile()
    if err != nil {
        cfg = defaultConfig()
    }

    // 2. 环境变量覆盖
    applyEnvOverrides(cfg)

    // 3. 应用默认值
    applyDefaults(cfg)

    // 4. 验证
    if err := validateConfig(cfg); err != nil {
        return nil, err
    }

    return cfg, nil
}

func loadConfigFile() (*Config, error) {
    paths := []string{
        "config.yaml",
        filepath.Join(os.Getenv("HOME"), ".config", "atest", "config.yaml"),
        "/etc/atest/config.yaml",
    }

    for _, path := range paths {
        if data, err := os.ReadFile(path); err == nil {
            var cfg Config
            if err := yaml.Unmarshal(data, &cfg); err == nil {
                return &cfg, nil
            }
        }
    }

    return nil, errors.New("config file not found")
}
```

**优势**:
- 零依赖（使用标准库）
- 快速加载（~10ms vs 50ms）
- 代码简洁（~80 行 vs 583 行）

---

### 3. Inline Retry Logic（内联重试）

**设计**: 重试逻辑直接嵌入调用点，而非独立组件

**实现策略**:

```go
func (m *AIManager) Generate(ctx context.Context, req *Request) (*Response, error) {
    var lastErr error

    // 最多尝试 3 次
    for attempt := 0; attempt < 3; attempt++ {
        // 非首次尝试，计算退避延迟
        if attempt > 0 {
            delay := calculateExponentialBackoff(attempt)

            select {
            case <-time.After(delay):
                // 继续重试
            case <-ctx.Done():
                // 上下文取消，立即返回
                return nil, ctx.Err()
            }
        }

        // 选择客户端
        client := m.selectHealthyClient()
        if client == nil {
            lastErr = ErrNoHealthyClients
            continue
        }

        // 执行请求
        resp, err := client.Generate(ctx, req)

        // 成功，直接返回
        if err == nil {
            return resp, nil
        }

        // 失败，判断是否可重试
        if !IsRetryable(err) {
            // 不可重试的错误，立即返回
            return nil, err
        }

        // 可重试的错误，记录并继续
        lastErr = err
    }

    // 所有重试都失败
    return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

// 辅助函数：计算指数退避
func calculateExponentialBackoff(attempt int) time.Duration {
    // 基础延迟 1 秒
    base := time.Second

    // 指数增长: 1s, 2s, 4s, 8s, ...
    delay := base * time.Duration(1<<uint(attempt-1))

    // 限制最大延迟 10 秒
    if delay > 10*time.Second {
        delay = 10 * time.Second
    }

    // 添加随机抖动（±25%）
    jitter := time.Duration(rand.Int63n(int64(delay / 4)))
    return delay + jitter
}

// 辅助函数：判断错误是否可重试
func IsRetryable(err error) bool {
    if err == nil {
        return false
    }

    // 上下文错误不重试
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        return false
    }

    // 网络错误重试
    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Timeout() {
        return true
    }

    // 根据错误消息判断
    errMsg := strings.ToLower(err.Error())

    // 可重试的错误
    retryablePatterns := []string{
        "rate limit", "too many requests", "quota exceeded",
        "service unavailable", "bad gateway", "gateway timeout",
        "connection refused", "connection reset",
        "500", "502", "503", "504", "429",
    }

    for _, pattern := range retryablePatterns {
        if strings.Contains(errMsg, pattern) {
            return true
        }
    }

    // 不可重试的错误
    nonRetryablePatterns := []string{
        "unauthorized", "forbidden", "invalid api key",
        "authentication failed", "bad request", "malformed",
        "400", "401", "403", "404",
    }

    for _, pattern := range nonRetryablePatterns {
        if strings.Contains(errMsg, pattern) {
            return false
        }
    }

    // 默认不重试
    return false
}
```

**优势**:
- 逻辑清晰，易于理解
- 无接口抽象开销
- 可根据场景灵活调整
- 减少代码行数

---

### 4. On-Demand Health Check（按需健康检查）

**设计**: 取消后台协程，改为按需同步检查

**实现**:

```go
// 检查单个提供商
func (m *AIManager) HealthCheck(ctx context.Context, provider string) (*HealthStatus, error) {
    m.mu.RLock()
    client, exists := m.clients[provider]
    m.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("provider not found: %s", provider)
    }

    // 实时检查，不使用缓存
    return client.HealthCheck(ctx)
}

// 批量检查所有提供商
func (m *AIManager) HealthCheckAll(ctx context.Context) map[string]*HealthStatus {
    m.mu.RLock()
    clients := make(map[string]interfaces.AIClient)
    for name, client := range m.clients {
        clients[name] = client
    }
    m.mu.RUnlock()

    results := make(map[string]*HealthStatus)

    // 并发检查（可选优化）
    var wg sync.WaitGroup
    var mu sync.Mutex

    for name, client := range clients {
        wg.Add(1)

        go func(name string, client interfaces.AIClient) {
            defer wg.Done()

            status, err := client.HealthCheck(ctx)
            if err != nil {
                status = &HealthStatus{
                    Healthy: false,
                    Status:  err.Error(),
                }
            }

            mu.Lock()
            results[name] = status
            mu.Unlock()
        }(name, client)
    }

    wg.Wait()
    return results
}
```

**对比**:

| 方面 | 后台协程（旧） | 按需检查（新） |
|------|--------------|--------------|
| **响应时间** | <1ms（缓存） | ~50ms（实时） |
| **准确性** | 中（最多 30s 延迟） | 高（实时） |
| **资源占用** | 高（常驻协程） | 低（按需） |
| **复杂度** | 高（108 行） | 低（30 行） |

---

### 5. Client Factory Functions（工厂函数）

**设计**: 使用普通函数代替接口，简化客户端创建

**实现**:

```go
// 创建客户端（普通函数，非接口）
func createClient(provider string, cfg config.AIService) (interfaces.AIClient, error) {
    // 规范化 provider 名称
    provider = normalizeProvider(provider)

    switch provider {
    case "openai", "deepseek", "custom":
        return createOpenAICompatibleClient(provider, cfg)

    case "ollama":
        return createOllamaClient(cfg)

    case "claude":
        return createClaudeClient(cfg)

    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
}

// OpenAI 兼容客户端
func createOpenAICompatibleClient(provider string, cfg config.AIService) (interfaces.AIClient, error) {
    config := &openai.Config{
        APIKey:    cfg.APIKey,
        BaseURL:   cfg.Endpoint,
        Model:     cfg.Model,
        MaxTokens: cfg.MaxTokens,
        Timeout:   cfg.Timeout.Value(),
    }

    // DeepSeek 使用特定端点
    if provider == "deepseek" && config.BaseURL == "" {
        config.BaseURL = "https://api.deepseek.com/v1"
    }

    // Custom 要求必须指定 endpoint
    if provider == "custom" && config.BaseURL == "" {
        return nil, fmt.Errorf("endpoint is required for custom provider")
    }

    return openai.NewClient(config)
}

// Ollama 客户端
func createOllamaClient(cfg config.AIService) (interfaces.AIClient, error) {
    config := &universal.Config{
        Provider:  "ollama",
        Endpoint:  cfg.Endpoint,
        Model:     cfg.Model,
        MaxTokens: cfg.MaxTokens,
        Timeout:   cfg.Timeout.Value(),
    }

    // 默认 endpoint
    if config.Endpoint == "" {
        config.Endpoint = "http://localhost:11434"
    }

    return universal.NewUniversalClient(config)
}

// 规范化 provider 名称
func normalizeProvider(provider string) string {
    // "local" 是 "ollama" 的别名
    if provider == "local" {
        return "ollama"
    }
    return strings.ToLower(strings.TrimSpace(provider))
}
```

**优势**:
- 无接口包装开销
- 代码直接清晰
- 易于添加新 provider
- 符合 Go 习惯

---

## 📦 目录结构

```
atest-ext-ai/
├── cmd/
│   └── atest-ext-ai/
│       └── main.go                 # 入口点
│
├── pkg/
│   ├── ai/
│   │   ├── manager.go              # [新] AIManager（统一管理器）
│   │   ├── engine.go               # [保留] AI Engine
│   │   ├── generator.go            # [保留] SQL Generator
│   │   ├── sql.go                  # [保留] SQL Dialects
│   │   ├── retry.go                # [简化] 重试辅助函数
│   │   ├── types.go                # [简化] 类型定义
│   │   ├── capabilities.go         # [保留] 能力检测
│   │   ├── discovery/
│   │   │   └── ollama.go           # [保留] Ollama 发现
│   │   └── providers/
│   │       ├── openai/             # [保留] OpenAI SDK 客户端
│   │       │   └── client.go
│   │       └── universal/          # [保留] 通用 HTTP 客户端
│   │           ├── client.go
│   │           └── strategy.go
│   │
│   ├── config/
│   │   ├── simple_loader.go        # [新] 简化配置加载器
│   │   ├── types.go                # [保留] 配置类型定义
│   │   └── duration.go             # [保留] Duration 类型
│   │
│   ├── interfaces/
│   │   └── ai.go                   # [保留] AIClient 接口
│   │
│   ├── plugin/
│   │   └── service.go              # [更新] gRPC 服务实现
│   │
│   ├── logging/
│   │   └── logger.go               # [保留] 日志工具
│   │
│   └── errors/
│       └── errors.go               # [保留] 错误定义
│
├── docs/
│   ├── REFACTORING_PLAN.md         # [新] 重构计划
│   ├── ARCHITECTURE_COMPARISON.md  # [新] 架构对比
│   ├── MIGRATION_GUIDE.md          # [新] 迁移指南
│   ├── NEW_ARCHITECTURE_DESIGN.md  # [新] 本文档
│   └── ...
│
├── frontend/                        # [保留] Vue 3 前端
├── config.yaml                      # [保留] 示例配置
├── go.mod                          # [更新] 移除 Viper
└── README.md                       # [更新] 文档

删除的文件:
├── pkg/ai/client.go                # [删除] ClientManager
├── pkg/ai/provider_manager.go      # [删除] ProviderManager
└── pkg/config/loader.go            # [删除] Viper Loader
```

---

## 🔄 交互流程

### SQL 生成流程

```
1. 前端发起请求
   POST /api/v1/data/query
   {
     "type": "ai",
     "key": "generate",
     "sql": "{\"model\":\"...\",\"prompt\":\"...\"}"
   }

2. gRPC Service 接收
   func (s *AIPluginService) Query(ctx, req)
     ├─> 解析请求参数
     ├─> 提取 model, prompt, config
     └─> 调用 AI Engine

3. AI Engine 处理
   func (e *aiEngine) GenerateSQL(ctx, sqlReq)
     ├─> 构建 GenerateOptions
     ├─> 设置 preferred_model
     ├─> 解析 runtime config
     └─> 调用 Generator

4. SQL Generator 执行
   func (g *SQLGenerator) Generate(ctx, nl, opts)
     ├─> 构建 prompt（包含 schema、context）
     ├─> 选择 AI client（支持 runtime client）
     │   ├─> 检查是否有 runtime API key
     │   ├─> 如有，创建临时 client
     │   └─> 否则使用默认 client
     └─> 调用 AIManager

5. AIManager 执行（带重试）
   func (m *AIManager) Generate(ctx, req)
     ├─> for attempt := 0; attempt < 3; attempt++
     │   ├─> selectHealthyClient()
     │   ├─> client.Generate(ctx, req)
     │   ├─> 成功？返回结果
     │   └─> 失败？
     │       ├─> IsRetryable(err)?
     │       ├─> 是 -> 计算退避延迟，继续
     │       └─> 否 -> 立即返回错误
     └─> 返回结果或错误

6. 响应格式化
   func parseAIResponse(resp)
     ├─> 提取 SQL 和 explanation
     ├─> 计算 confidence score
     ├─> 检测 query type
     └─> 构建 GenerateSQLResponse

7. 返回前端
   {
     "data": [
       {"key": "content", "value": "sql:...\nexplanation:..."},
       {"key": "meta", "value": "{\"confidence\":0.8,...}"},
       {"key": "success", "value": "true"}
     ]
   }
```

### 模型列表流程

```
1. 前端发起请求
   POST /api/v1/data/query
   {
     "type": "ai",
     "key": "models",
     "sql": "{\"provider\":\"ollama\"}"
   }

2. gRPC Service 接收
   func (s *AIPluginService) handleGetModels(ctx, req)
     ├─> 解析 provider 参数
     ├─> 规范化名称（local -> ollama）
     └─> 调用 AIManager

3. AIManager 执行
   func (m *AIManager) GetModels(ctx, provider)
     ├─> 获取对应 client
     ├─> 调用 client.GetCapabilities(ctx)
     └─> 返回 models 列表

4. AI Client 执行
   func (c *Client) GetCapabilities(ctx)
     ├─> 根据 provider 类型
     │   ├─> Ollama: GET /api/tags
     │   └─> OpenAI: 返回预定义列表
     └─> 解析并返回 ModelInfo[]

5. 返回前端
   {
     "data": [
       {"key": "models", "value": "[{\"id\":\"...\",\"name\":\"...\"}...]"},
       {"key": "count", "value": "10"},
       {"key": "success", "value": "true"}
     ]
   }
```

### 连接测试流程

```
1. 前端发起请求
   POST /api/v1/data/query
   {
     "type": "ai",
     "key": "test_connection",
     "sql": "{\"provider\":\"ollama\",\"endpoint\":\"...\",\"model\":\"...\"}"
   }

2. gRPC Service 接收
   func (s *AIPluginService) handleTestConnection(ctx, req)
     ├─> 解析配置参数
     ├─> 规范化 provider 名称
     └─> 调用 AIManager

3. AIManager 执行
   func (m *AIManager) TestConnection(ctx, cfg)
     ├─> 创建临时 client
     │   └─> universal.NewUniversalClient(cfg)
     ├─> 执行健康检查
     │   └─> client.HealthCheck(ctx)
     └─> 返回测试结果

4. 返回前端
   {
     "data": [
       {"key": "result", "value": "{\"success\":true,...}"},
       {"key": "success", "value": "true"},
       {"key": "message", "value": "Connection successful"},
       {"key": "response_time_ms", "value": "45"}
     ]
   }
```

---

## 📊 性能设计目标

### 启动性能

| 指标 | 目标 | 实现方式 |
|------|------|----------|
| **总启动时间** | < 150ms | 简化配置加载、移除 Viper |
| **配置加载** | < 15ms | 直接 YAML 解析 |
| **客户端初始化** | < 50ms | 懒加载、并发创建 |
| **依赖初始化** | < 80ms | 减少依赖数量 |

### 运行时性能

| 指标 | 目标 | 实现方式 |
|------|------|----------|
| **内存占用** | < 20MB | 移除后台协程、优化数据结构 |
| **Goroutine 数** | Base only | 移除后台健康检查 |
| **SQL 生成延迟** | < 4s | 内联重试、优化提示词 |
| **健康检查延迟** | < 100ms | 按需检查、并发检查 |
| **模型列表延迟** | < 500ms | 客户端缓存 |

### 资源效率

```
Goroutine 模型:
└── Main goroutine
    └── 按需创建临时 goroutine
        ├─> 健康检查（并发）
        ├─> 模型发现（并发）
        └─> 请求处理

无常驻后台 goroutine
```

---

## 🧪 测试策略

### 单元测试

```go
// 测试 AIManager
func TestAIManager_Generate(t *testing.T) {
    // Mock AIClient
    mockClient := &mockAIClient{
        generateFunc: func(ctx, req) (*Response, error) {
            return &Response{SQL: "SELECT 1"}, nil
        },
    }

    // 创建 AIManager
    manager := &AIManager{
        clients: map[string]interfaces.AIClient{
            "test": mockClient,
        },
        config: Config{DefaultService: "test"},
    }

    // 执行测试
    resp, err := manager.Generate(context.Background(), &Request{Prompt: "test"})

    // 验证结果
    assert.NoError(t, err)
    assert.Equal(t, "SELECT 1", resp.SQL)
}

// 测试重试逻辑
func TestAIManager_GenerateWithRetry(t *testing.T) {
    attempts := 0

    mockClient := &mockAIClient{
        generateFunc: func(ctx, req) (*Response, error) {
            attempts++
            if attempts < 3 {
                return nil, errors.New("rate limit exceeded")  // 可重试
            }
            return &Response{SQL: "SELECT 1"}, nil
        },
    }

    manager := &AIManager{
        clients: map[string]interfaces.AIClient{"test": mockClient},
        config:  Config{DefaultService: "test"},
    }

    resp, err := manager.Generate(context.Background(), &Request{})

    assert.NoError(t, err)
    assert.Equal(t, 3, attempts)  // 验证重试了 3 次
    assert.NotNil(t, resp)
}
```

### 集成测试

```go
func TestIntegration_SQLGeneration(t *testing.T) {
    // 启动真实服务
    cfg := loadTestConfig()
    manager, err := NewAIManager(cfg.AI)
    require.NoError(t, err)
    defer manager.Close()

    // 执行实际请求
    resp, err := manager.Generate(context.Background(), &Request{
        Prompt: "查询所有用户",
    })

    // 验证结果
    assert.NoError(t, err)
    assert.Contains(t, resp.SQL, "SELECT")
    assert.Contains(t, resp.SQL, "users")
}
```

### 性能测试

```go
func BenchmarkAIManager_Generate(b *testing.B) {
    manager := setupTestManager()

    req := &Request{Prompt: "查询用户"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = manager.Generate(context.Background(), req)
    }
}

func BenchmarkConfigLoad(b *testing.B) {
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = config.LoadConfig()
    }
}
```

---

## 🔒 安全设计

### 配置安全

```go
// 1. API Key 不记录到日志
func (m *AIManager) addClient(name string, cfg Config) error {
    logging.Logger.Debug("Adding client",
        "provider", name,
        "endpoint", cfg.Endpoint,
        "api_key", maskAPIKey(cfg.APIKey),  // 掩码处理
    )
    // ...
}

func maskAPIKey(key string) string {
    if len(key) <= 8 {
        return "***"
    }
    return key[:4] + "***" + key[len(key)-4:]
}

// 2. 配置文件权限检查
func loadConfigFile() (*Config, error) {
    path := "config.yaml"

    // 检查文件权限
    info, err := os.Stat(path)
    if err != nil {
        return nil, err
    }

    // 警告：配置文件权限过宽
    if info.Mode().Perm() & 0077 != 0 {
        logging.Logger.Warn("Config file has overly permissive permissions",
            "path", path,
            "mode", info.Mode(),
        )
    }

    // 读取配置
    data, err := os.ReadFile(path)
    // ...
}
```

### 请求安全

```go
// 1. 上下文超时
func (m *AIManager) Generate(ctx context.Context, req *Request) (*Response, error) {
    // 确保有超时设置
    if _, ok := ctx.Deadline(); !ok {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
        defer cancel()
    }

    // ...
}

// 2. 输入验证
func validateRequest(req *Request) error {
    if req.Prompt == "" {
        return errors.New("prompt is required")
    }

    if len(req.Prompt) > 10000 {
        return errors.New("prompt too long (max 10000 characters)")
    }

    // 防止注入攻击
    if containsSQLInjection(req.Prompt) {
        return errors.New("prompt contains potentially malicious content")
    }

    return nil
}
```

---

## 📈 可扩展性设计

### 添加新 Provider

```go
// 1. 在 createClient 函数中添加分支
func createClient(provider string, cfg config.AIService) (interfaces.AIClient, error) {
    switch provider {
    // ... 现有 providers

    case "new-provider":
        return createNewProviderClient(cfg)

    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
}

// 2. 实现创建函数
func createNewProviderClient(cfg config.AIService) (interfaces.AIClient, error) {
    config := &newprovider.Config{
        APIKey:   cfg.APIKey,
        Endpoint: cfg.Endpoint,
        Model:    cfg.Model,
    }

    return newprovider.NewClient(config)
}

// 3. 配置文件添加服务
ai:
  services:
    new-provider:
      enabled: true
      provider: new-provider
      endpoint: https://api.newprovider.com
      model: model-name
      api_key: ${NEW_PROVIDER_API_KEY}
```

### 添加新功能

```go
// 在 AIManager 中添加方法
func (m *AIManager) NewFeature(ctx context.Context, params Params) (*Result, error) {
    // 1. 验证参数
    if err := validateParams(params); err != nil {
        return nil, err
    }

    // 2. 选择客户端
    client := m.selectHealthyClient()

    // 3. 执行操作（带重试）
    result, err := m.executeWithRetry(ctx, func() (*Result, error) {
        return client.NewMethod(ctx, params)
    })

    // 4. 返回结果
    return result, err
}

// 在 gRPC Service 中添加处理函数
func (s *AIPluginService) handleNewFeature(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
    // 解析参数
    var params Params
    if err := json.Unmarshal([]byte(req.Sql), &params); err != nil {
        return nil, err
    }

    // 调用 AIManager
    result, err := s.aiManager.NewFeature(ctx, params)
    if err != nil {
        return nil, err
    }

    // 格式化响应
    return formatResult(result), nil
}
```

---

## 📝 编码规范

### Go 代码风格

```go
// 1. 使用具体类型，避免接口（除非确实需要）
// ❌ 不推荐
type Factory interface {
    Create() Thing
}

// ✅ 推荐
func CreateThing(config Config) (*Thing, error) {
    return &Thing{}, nil
}

// 2. 错误处理要明确
// ❌ 不推荐
if err := doSomething(); err != nil {
    return nil, err  // 丢失上下文
}

// ✅ 推荐
if err := doSomething(); err != nil {
    return nil, fmt.Errorf("failed to do something: %w", err)
}

// 3. 使用有意义的变量名
// ❌ 不推荐
func process(c *C, r *R) (*Rs, error)

// ✅ 推荐
func processRequest(ctx context.Context, req *Request) (*Response, error)

// 4. 简洁的函数
// 函数应该短小精悍，一个函数只做一件事
func (m *AIManager) Generate(ctx context.Context, req *Request) (*Response, error) {
    // 验证
    if err := validateRequest(req); err != nil {
        return nil, err
    }

    // 选择客户端
    client := m.selectClient()

    // 执行请求（复杂逻辑提取到独立函数）
    return m.executeRequest(ctx, client, req)
}
```

### 注释规范

```go
// Package ai provides unified AI client management and request processing.
//
// This package implements a simplified architecture focusing on actual needs
// rather than potential future requirements, following KISS and YAGNI principles.
package ai

// AIManager manages multiple AI clients and provides unified access.
//
// It handles:
// - Client lifecycle management
// - Provider discovery
// - Model listing
// - Connection testing
// - Health checking (on-demand)
// - Retry logic (inline)
//
// Usage:
//   manager, err := NewAIManager(config)
//   if err != nil {
//       return err
//   }
//   defer manager.Close()
//
//   resp, err := manager.Generate(ctx, request)
type AIManager struct {
    // clients holds all initialized AI clients indexed by provider name
    clients map[string]interfaces.AIClient

    // config stores the AI configuration
    config config.AIConfig

    // discovery handles Ollama service discovery
    discovery *discovery.OllamaDiscovery

    // mu protects concurrent access to clients map
    mu sync.RWMutex
}
```

---

## 🎓 设计权衡

### 为什么不用 X？

**Q: 为什么不保留 Viper？**
- A: Viper 提供了很多高级功能（热重载、远程配置、加密等），但插件场景下都用不到。简单的 YAML 解析足够满足需求，还能减少依赖和提升性能。

**Q: 为什么不保留接口抽象？**
- A: Go 的哲学是"接受接口，返回结构体"。内部组件使用接口会增加复杂度，而当前只有一个实现时，接口是不必要的。需要时再抽象也不迟（YAGNI）。

**Q: 为什么不保留后台健康检查？**
- A: 对于单机插件场景，AI 服务不会频繁宕机。实时按需检查更准确，且节省资源。如果真需要持续监控，可以在上层（如 Kubernetes）实现。

**Q: 为什么不使用依赖注入框架？**
- A: Go 不鼓励使用依赖注入框架（如 Wire、Dig），因为显式依赖传递更清晰。对于小型项目，手动管理依赖足够简单且易于理解。

**Q: 为什么不实现更多的设计模式？**
- A: 设计模式是解决特定问题的工具，不是目标。只有当问题出现时才应用相应的模式，而不是预先实现所有可能用到的模式。

---

## 📚 参考资料

### Go 设计哲学

- [Go Proverbs](https://go-proverbs.github.io/)
  - "Don't communicate by sharing memory, share memory by communicating"
  - "The bigger the interface, the weaker the abstraction"
  - "Clear is better than clever"

- [Rob Pike - Simplicity is Complicated](https://www.youtube.com/watch?v=rFejpH_tAHM)
  - "Complexity is multiplicative"
  - "Features add up, complexity multiplies"

- [Dave Cheney - Practical Go](https://dave.cheney.net/practical-go)
  - "Prefer functions over interfaces"
  - "Return early, return often"
  - "Accept interfaces, return structs"

### 架构设计

- [Martin Fowler - YAGNI](https://martinfowler.com/bliki/Yagni.html)
- [KISS Principle](https://en.wikipedia.org/wiki/KISS_principle)
- [The Twelve-Factor App](https://12factor.net/)

---

**文档结束**

本设计文档详细说明了新架构的设计理念、组件结构和实现细节。如有疑问或建议，请参考其他配套文档或联系架构团队。
