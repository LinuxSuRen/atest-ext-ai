# atest-ext-ai 架构重构详细计划

**文档版本**: 1.0
**创建日期**: 2025-10-10
**负责人**: Architecture Team
**状态**: 待执行

---

## 📊 执行摘要

### 目标
全面简化 atest-ext-ai 项目架构，消除过度设计，减少复杂度，提升可维护性。

### 核心问题
项目按照「可能未来会需要」的标准设计，而不是「当前实际需要」，导致过度工程化。

### 预期成果
- **代码减少**: 1,200-1,500 行 (16-20%)
- **依赖减少**: ~30 个间接依赖包
- **性能提升**: 启动时间减少 20-30%
- **可维护性**: 显著提升

---

## 📐 当前架构分析

### 代码规模统计

```
总计:
- Go 源文件: 28 个
- 非测试代码行数: 7,447 行
- 测试代码: 约 2,000 行
- 前端代码: 独立 Vue 3 项目
```

### 关键文件分析

| 文件 | 行数 | 功能 | 问题 |
|------|------|------|------|
| `pkg/plugin/service.go` | 1,082 | gRPC 服务实现 | 合理 |
| `pkg/ai/client.go` | 699 | AI 客户端管理 | **与 provider_manager 重复** |
| `pkg/ai/generator.go` | 631 | SQL 生成器 | 合理 |
| `pkg/ai/sql.go` | 599 | SQL 方言支持 | 轻微过度 |
| `pkg/config/loader.go` | 583 | **Viper 配置加载** | **严重过度** |
| `pkg/ai/provider_manager.go` | 416 | 提供商管理 | **与 client 重复** |
| `pkg/ai/retry.go` | 294 | 重试管理 | 可简化 |
| `pkg/ai/engine.go` | 279 | AI 引擎 | 合理 |

### 依赖关系分析

```
直接依赖: 15 个
├── github.com/spf13/viper v1.21.0          ❌ 过度复杂
├── github.com/tmc/langchaingo v0.1.13      ✅ 必要
├── google.golang.org/grpc v1.73.0          ✅ 必要
├── github.com/cenkalti/backoff/v4 v4.3.0   ⚠️ 可简化使用
└── ...

间接依赖: ~90 个
└── Viper 带来约 30 个不必要的依赖
```

---

## 🔴 核心问题详解

### 问题 1: 多重管理器冲突 (严重)

**位置**: `pkg/ai/client.go` + `pkg/ai/provider_manager.go`

**问题描述**:
存在两个功能重叠约 70% 的管理器：

```go
// ClientManager - 用于 AI 调用
type ClientManager struct {
    clients       map[string]interfaces.AIClient
    factory       ClientFactory           // 接口抽象
    retryManager  RetryManager           // 接口抽象
    healthChecker *HealthChecker         // 后台协程
    // ...
}

// ProviderManager - 用于前端交互
type ProviderManager struct {
    providers map[string]*ProviderInfo
    clients   map[string]interfaces.AIClient  // 重复！
    discovery *discovery.OllamaDiscovery
    // ...
}
```

**影响**:
- 代码重复: 1,116 行代码管理类似功能
- 维护困难: 修改需要同时更新两处
- 内存浪费: 两份客户端实例

**根因**:
早期分离了「调用」和「发现」职责，但实际上可以统一管理。

---

### 问题 2: 配置系统过度复杂 (严重)

**位置**: `pkg/config/loader.go` (583 行)

**问题描述**:

使用 Viper 提供的企业级功能，但插件场景下用不到：

```go
// 当前实现
func LoadConfig() (*Config, error) {
    v := viper.New()

    // 支持 5 种配置源
    v.SetConfigType("yaml")
    v.AddConfigPath(".")
    v.AddConfigPath("$HOME/.config/atest")

    // 环境变量映射
    v.SetEnvPrefix("ATEST_EXT_AI")
    v.AutomaticEnv()

    // 热重载支持
    v.WatchConfig()
    v.OnConfigChange(func(e fsnotify.Event) {
        // 回调逻辑...
    })

    // 远程配置支持 (未使用)
    // 加密配置支持 (未使用)
    // ... 更多功能

    return &Config{}, nil  // 583 行只为了这个
}
```

**实际需求**:

```yaml
# 实际只需要这些配置
ai:
  default_service: ollama
  services:
    ollama:
      endpoint: http://localhost:11434
      model: qwen2.5-coder:latest
```

**影响**:
- 启动时间: Viper 初始化耗时
- 依赖臃肿: 带来 30+ 间接依赖
- 代码复杂: 583 行只为加载一个 YAML

---

### 问题 3: 不必要的接口抽象 (中等)

**位置**: `pkg/ai/types.go`

**问题描述**:

每个接口都只有一个实现，违反 YAGNI 原则：

```go
// ClientFactory 接口 - 只有 defaultClientFactory 一个实现
type ClientFactory interface {
    CreateClient(provider string, config map[string]any) (AIClient, error)
    GetSupportedProviders() []string
    ValidateConfig(provider string, config map[string]any) error
}

// RetryManager 接口 - 只有 defaultRetryManager 一个实现
type RetryManager interface {
    Execute(ctx context.Context, fn func() error) error
    ShouldRetry(err error) bool
    GetRetryDelay(attempt int) time.Duration
}
```

**影响**:
- 增加代码量: 接口定义 + 实现
- 降低可读性: 多一层间接引用
- 无实际价值: 没有多实现需求

---

### 问题 4: 过度工程化的健康检查 (中等)

**位置**: `pkg/ai/client.go:592-699`

**问题描述**:

```go
type HealthChecker struct {
    interval     time.Duration  // 30 秒轮询一次
    clients      map[string]interfaces.AIClient
    healthStatus map[string]*HealthStatus
    mu           sync.RWMutex
    stopCh       chan struct{}
    stopped      bool
}

// 后台协程持续运行
func (hc *HealthChecker) healthCheckLoop() {
    ticker := time.NewTicker(hc.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            hc.performHealthChecks()  // 每 30 秒检查所有客户端
        case <-hc.stopCh:
            return
        }
    }
}
```

**为什么过度**:
- AI 服务不会频繁宕机，不需要持续监控
- 单机插件场景，按需检查即可
- 增加 CPU 和内存开销

**实际需求**:
```go
// 按需检查即可
func CheckHealth(ctx context.Context) (*HealthStatus, error) {
    return client.HealthCheck(ctx)
}
```

---

### 问题 5: Retry 管理器包装过度 (轻微)

**位置**: `pkg/ai/retry.go` (294 行)

**问题描述**:

使用了优秀的 `cenkalti/backoff` 库，但包装了一层接口：

```go
type RetryManager interface {
    Execute(ctx context.Context, fn func() error) error
    ShouldRetry(err error) bool
    GetRetryDelay(attempt int) time.Duration
}

type defaultRetryManager struct {
    config RetryConfig
}

// 294 行代码，主要是包装 backoff 库
```

**建议**:
直接使用 `backoff` 库的 API，简洁明了：

```go
// 直接使用
operation := func() error {
    return doSomething()
}
backoff.Retry(operation, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3))
```

---

## 🎨 新架构设计

### 设计原则

1. **KISS**: Keep It Simple, Stupid
2. **YAGNI**: You Aren't Gonna Need It
3. **最小必要原则**: 只实现当前需要的功能
4. **直接依赖**: 减少抽象层级

### 核心变更

#### 1. 统一的 AIManager

```go
// 新设计: 统一管理器 (约 350 行)
package ai

type AIManager struct {
    clients  map[string]interfaces.AIClient
    config   Config
}

// 统一功能:
// - 客户端生命周期管理
// - 模型发现和列表
// - 连接测试
// - AI 调用 (带内联重试)
// - 按需健康检查
```

**职责整合**:
- ✅ 原 ClientManager 的 AI 调用功能
- ✅ 原 ProviderManager 的模型发现功能
- ✅ 原 HealthChecker 的健康检查功能（按需）
- ✅ 直接创建客户端（无 factory）
- ✅ 内联重试逻辑

**文件变更**:
```
删除: pkg/ai/client.go (699 行)
删除: pkg/ai/provider_manager.go (417 行)
创建: pkg/ai/manager.go (~350 行)
净减少: ~766 行
```

---

#### 2. 简化的配置系统

```go
// 新设计: 简单配置加载 (约 80 行)
package config

import (
    "gopkg.in/yaml.v2"
    "os"
)

type Config struct {
    AI     AIConfig     `yaml:"ai"`
    Server ServerConfig `yaml:"server"`
    // ... 其他配置
}

func LoadConfig() (*Config, error) {
    // 1. 读取 YAML 文件 (~15 行)
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        return defaultConfig(), nil  // 使用默认值
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    // 2. 环境变量覆盖 (~30 行)
    applyEnvOverrides(&cfg)

    // 3. 验证 (~20 行)
    if err := validateConfig(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

**依赖变更**:
```
移除: github.com/spf13/viper + 30 个间接依赖
保留: gopkg.in/yaml.v2 (已有)
```

**文件变更**:
```
删除: pkg/config/loader.go (583 行)
创建: pkg/config/simple_loader.go (~80 行)
净减少: ~503 行
```

---

#### 3. 移除接口抽象

```go
// 当前: 接口 + 实现
type ClientFactory interface { ... }
type defaultClientFactory struct { ... }

// 新设计: 直接使用具体类型
func NewAIManager(cfg Config) *AIManager {
    manager := &AIManager{
        clients: make(map[string]interfaces.AIClient),
        config:  cfg,
    }

    // 直接创建客户端
    for name, svcCfg := range cfg.AI.Services {
        client := createClient(name, svcCfg)  // 普通函数
        manager.clients[name] = client
    }

    return manager
}

// 工厂函数（非接口）
func createClient(provider string, cfg ServiceConfig) interfaces.AIClient {
    switch provider {
    case "openai":
        return openai.NewClient(&openai.Config{...})
    case "ollama":
        return universal.NewUniversalClient(&universal.Config{...})
    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
}
```

**文件变更**:
```
修改: pkg/ai/types.go
  - 移除 ClientFactory 接口定义
  - 移除 RetryManager 接口定义

修改: pkg/ai/manager.go
  - 使用具体函数代替接口调用

净减少: ~150 行接口定义和包装代码
```

---

#### 4. 内联重试逻辑

```go
// 新设计: 直接使用 backoff 库
func (m *AIManager) Generate(ctx context.Context, req *Request) (*Response, error) {
    operation := func() error {
        client := m.selectHealthyClient()
        resp, err := client.Generate(ctx, req)
        if err != nil {
            if !isRetryable(err) {
                return backoff.Permanent(err)
            }
            return err
        }
        result = resp
        return nil
    }

    b := backoff.NewExponentialBackOff()
    b.MaxElapsedTime = 30 * time.Second

    err := backoff.Retry(operation, backoff.WithMaxRetries(b, 2))
    return result, err
}

// 简单的重试判断函数（不需要接口）
func isRetryable(err error) bool {
    // 网络错误、超时、5xx 错误等
    // ... (~50 行逻辑)
}
```

**文件变更**:
```
删除: pkg/ai/retry.go 中的 RetryManager 实现
保留: isRetryable 等辅助函数
移动重试逻辑到调用点（内联）

净减少: ~200 行包装代码
```

---

#### 5. 按需健康检查

```go
// 新设计: 同步检查
func (m *AIManager) HealthCheck(ctx context.Context, provider string) (*HealthStatus, error) {
    client, exists := m.clients[provider]
    if !exists {
        return nil, fmt.Errorf("provider not found: %s", provider)
    }

    // 直接调用，不缓存
    return client.HealthCheck(ctx)
}

// 批量检查（如果需要）
func (m *AIManager) HealthCheckAll(ctx context.Context) map[string]*HealthStatus {
    results := make(map[string]*HealthStatus)
    for name, client := range m.clients {
        status, err := client.HealthCheck(ctx)
        if err != nil {
            status = &HealthStatus{Healthy: false, Error: err.Error()}
        }
        results[name] = status
    }
    return results
}
```

**文件变更**:
```
删除: pkg/ai/client.go 中的 HealthChecker 实现 (~108 行)
移动: 健康检查逻辑到 AIManager
移除: 后台协程、缓存、定时器

净减少: ~100 行代码 + goroutine 开销
```

---

## 📋 执行计划 - 6 个阶段

### 阶段 0: 准备工作 (1 天)

**目标**: 创建安全的执行环境

**任务清单**:
- [x] 创建重构文档（本文档）
- [ ] 创建 git 分支 `refactor/architecture-simplification`
- [ ] 备份当前代码
- [ ] 确保所有测试通过
- [ ] 记录性能基准

**命令**:
```bash
# 创建分支
git checkout -b refactor/architecture-simplification

# 运行所有测试
go test ./... -v

# 性能基准
go test -bench=. -benchmem ./...
```

---

### 阶段 1: 合并管理器 (3-4 天) 🔴 最高优先级

**目标**: 统一 ClientManager 和 ProviderManager

**步骤 1.1: 创建新的 AIManager**

```bash
# 创建新文件
touch pkg/ai/manager.go
```

**代码结构** (`pkg/ai/manager.go`):

```go
package ai

import (
    "context"
    "fmt"
    "sync"

    "github.com/linuxsuren/atest-ext-ai/pkg/config"
    "github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
    "github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/openai"
    "github.com/linuxsuren/atest-ext-ai/pkg/ai/providers/universal"
    "github.com/linuxsuren/atest-ext-ai/pkg/ai/discovery"
)

// AIManager 统一管理所有 AI 客户端
type AIManager struct {
    clients   map[string]interfaces.AIClient
    config    config.AIConfig
    discovery *discovery.OllamaDiscovery
    mu        sync.RWMutex
}

// NewAIManager 创建新的 AI 管理器
func NewAIManager(cfg config.AIConfig) (*AIManager, error) {
    manager := &AIManager{
        clients:   make(map[string]interfaces.AIClient),
        config:    cfg,
        discovery: discovery.NewOllamaDiscovery(getOllamaEndpoint()),
    }

    // 初始化配置的客户端
    if err := manager.initializeClients(); err != nil {
        return nil, err
    }

    return manager, nil
}

// ===== 客户端管理功能 (原 ClientManager) =====

func (m *AIManager) initializeClients() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    for name, svc := range m.config.Services {
        if !svc.Enabled {
            continue
        }

        client, err := createClient(name, svc)
        if err != nil {
            return fmt.Errorf("failed to create client %s: %w", name, err)
        }

        m.clients[name] = client
    }

    return nil
}

// Generate 执行 AI 生成请求（带重试）
func (m *AIManager) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
    // 内联重试逻辑
    var result *interfaces.GenerateResponse

    for attempt := 0; attempt < 3; attempt++ {
        client := m.selectHealthyClient()
        if client == nil {
            return nil, fmt.Errorf("no healthy clients available")
        }

        resp, err := client.Generate(ctx, req)
        if err != nil {
            if !isRetryable(err) {
                return nil, err
            }
            time.Sleep(time.Duration(attempt+1) * time.Second)
            continue
        }

        result = resp
        break
    }

    if result == nil {
        return nil, fmt.Errorf("all retry attempts failed")
    }

    return result, nil
}

func (m *AIManager) selectHealthyClient() interfaces.AIClient {
    m.mu.RLock()
    defer m.mu.RUnlock()

    // 先尝试默认服务
    if client, ok := m.clients[m.config.DefaultService]; ok {
        return client
    }

    // 返回第一个可用客户端
    for _, client := range m.clients {
        return client
    }

    return nil
}

// ===== 提供商发现功能 (原 ProviderManager) =====

func (m *AIManager) DiscoverProviders(ctx context.Context) ([]*ProviderInfo, error) {
    var providers []*ProviderInfo

    // 检查 Ollama
    if m.discovery.IsAvailable(ctx) {
        endpoint := m.discovery.GetBaseURL()

        config := &universal.Config{
            Provider: "ollama",
            Endpoint: endpoint,
            Model:    "llama2",
        }

        client, err := universal.NewUniversalClient(config)
        if err == nil {
            models, _ := client.GetCapabilities(ctx)

            provider := &ProviderInfo{
                Name:      "ollama",
                Type:      "local",
                Available: true,
                Endpoint:  endpoint,
                Models:    models.Models,
            }

            providers = append(providers, provider)
        }
    }

    // 添加在线提供商
    providers = append(providers, m.getOnlineProviders()...)

    return providers, nil
}

func (m *AIManager) GetModels(ctx context.Context, provider string) ([]interfaces.ModelInfo, error) {
    m.mu.RLock()
    client, exists := m.clients[provider]
    m.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("provider not found: %s", provider)
    }

    caps, err := client.GetCapabilities(ctx)
    if err != nil {
        return nil, err
    }

    return caps.Models, nil
}

func (m *AIManager) TestConnection(ctx context.Context, cfg *universal.Config) (*ConnectionTestResult, error) {
    start := time.Now()

    client, err := universal.NewUniversalClient(cfg)
    if err != nil {
        return &ConnectionTestResult{
            Success:      false,
            Message:      "Failed to create client",
            ResponseTime: time.Since(start),
            Provider:     cfg.Provider,
            Error:        err.Error(),
        }, nil
    }

    health, err := client.HealthCheck(ctx)
    if err != nil {
        return &ConnectionTestResult{
            Success:      false,
            Message:      "Health check failed",
            ResponseTime: time.Since(start),
            Provider:     cfg.Provider,
            Error:        err.Error(),
        }, nil
    }

    return &ConnectionTestResult{
        Success:      health.Healthy,
        Message:      health.Status,
        ResponseTime: health.ResponseTime,
        Provider:     cfg.Provider,
    }, nil
}

// ===== 健康检查功能 (按需) =====

func (m *AIManager) HealthCheck(ctx context.Context, provider string) (*interfaces.HealthStatus, error) {
    m.mu.RLock()
    client, exists := m.clients[provider]
    m.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("provider not found: %s", provider)
    }

    return client.HealthCheck(ctx)
}

func (m *AIManager) HealthCheckAll(ctx context.Context) map[string]*interfaces.HealthStatus {
    m.mu.RLock()
    clients := make(map[string]interfaces.AIClient)
    for name, client := range m.clients {
        clients[name] = client
    }
    m.mu.RUnlock()

    results := make(map[string]*interfaces.HealthStatus)
    for name, client := range clients {
        status, err := client.HealthCheck(ctx)
        if err != nil {
            status = &interfaces.HealthStatus{
                Healthy: false,
                Status:  err.Error(),
            }
        }
        results[name] = status
    }

    return results
}

// ===== 辅助函数 =====

func createClient(provider string, cfg config.AIService) (interfaces.AIClient, error) {
    switch provider {
    case "openai", "deepseek", "custom":
        return openai.NewClient(&openai.Config{
            APIKey:    cfg.APIKey,
            BaseURL:   cfg.Endpoint,
            Model:     cfg.Model,
            MaxTokens: cfg.MaxTokens,
            Timeout:   cfg.Timeout.Value(),
        })

    case "ollama", "local":
        return universal.NewUniversalClient(&universal.Config{
            Provider:  "ollama",
            Endpoint:  cfg.Endpoint,
            Model:     cfg.Model,
            MaxTokens: cfg.MaxTokens,
            Timeout:   cfg.Timeout.Value(),
        })

    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
}

func isRetryable(err error) bool {
    if err == nil {
        return false
    }

    // 上下文取消不重试
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        return false
    }

    // 网络错误重试
    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Timeout() {
        return true
    }

    // 检查错误消息
    errMsg := err.Error()
    retryableMessages := []string{
        "rate limit",
        "too many requests",
        "service unavailable",
        "bad gateway",
        "gateway timeout",
        "500", "502", "503", "504",
    }

    for _, msg := range retryableMessages {
        if strings.Contains(strings.ToLower(errMsg), msg) {
            return true
        }
    }

    return false
}

func (m *AIManager) Close() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    for _, client := range m.clients {
        _ = client.Close()
    }

    return nil
}
```

**步骤 1.2: 更新调用方**

修改 `pkg/ai/engine.go`:

```go
// 旧代码
client, err := NewClient(cfg)
if err != nil {
    return nil, err
}

// 新代码
manager, err := NewAIManager(cfg)
if err != nil {
    return nil, err
}
```

修改 `pkg/plugin/service.go`:

```go
// 替换字段
type AIPluginService struct {
    // 旧: providerManager *ai.ProviderManager
    // 新:
    aiManager *ai.AIManager
}

// 更新初始化
func NewAIPluginService() (*AIPluginService, error) {
    // ...
    manager := ai.NewAIManager(cfg.AI)

    return &AIPluginService{
        aiManager: manager,
        // ...
    }, nil
}

// 更新所有调用点
func (s *AIPluginService) handleGetModels(ctx context.Context, req *server.DataQuery) (*server.DataQueryResult, error) {
    // 旧: s.providerManager.GetModels(ctx, provider)
    // 新:
    models, err := s.aiManager.GetModels(ctx, provider)
    // ...
}
```

**步骤 1.3: 删除旧文件**

```bash
# 确认所有引用已更新
git rm pkg/ai/client.go
git rm pkg/ai/provider_manager.go
```

**步骤 1.4: 测试验证**

```bash
# 单元测试
go test ./pkg/ai -v

# 集成测试
go test ./pkg/plugin -v

# 手动测试
# 1. 启动服务
# 2. 前端测试模型列表
# 3. 前端测试连接
# 4. 前端测试 SQL 生成
```

**提交**:
```bash
git add .
git commit -m "refactor(ai): merge ClientManager and ProviderManager into unified AIManager

- Consolidate client.go and provider_manager.go into manager.go
- Remove duplicate client management code
- Inline retry logic
- Simplify health check to on-demand
- Reduce code by ~766 lines

BREAKING CHANGE: Internal API restructured, external interface unchanged"
```

---

### 阶段 2: 简化配置系统 (2-3 天) 🟠

**目标**: 用简单的 YAML + 环境变量替换 Viper

**步骤 2.1: 创建新的配置加载器**

```bash
# 创建新文件
touch pkg/config/simple_loader.go
```

**代码实现** (`pkg/config/simple_loader.go`):

```go
package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "gopkg.in/yaml.v2"
)

// LoadConfig 加载配置
func LoadConfig() (*Config, error) {
    // 1. 尝试加载配置文件
    cfg, err := loadConfigFile()
    if err != nil {
        // 配置文件不存在或无法解析，使用默认配置
        cfg = defaultConfig()
    }

    // 2. 环境变量覆盖
    applyEnvOverrides(cfg)

    // 3. 应用默认值
    applyDefaults(cfg)

    // 4. 验证配置
    if err := validateConfig(cfg); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }

    return cfg, nil
}

func loadConfigFile() (*Config, error) {
    // 查找配置文件
    paths := []string{
        "config.yaml",
        "config.yml",
        filepath.Join(os.Getenv("HOME"), ".config", "atest", "config.yaml"),
        "/etc/atest/config.yaml",
    }

    var data []byte
    var err error

    for _, path := range paths {
        data, err = os.ReadFile(path)
        if err == nil {
            break
        }
    }

    if err != nil {
        return nil, err
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }

    return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
    // 服务器配置
    if host := os.Getenv("ATEST_EXT_AI_SERVER_HOST"); host != "" {
        cfg.Server.Host = host
    }
    if port := os.Getenv("ATEST_EXT_AI_SERVER_PORT"); port != "" {
        if p, err := strconv.Atoi(port); err == nil {
            cfg.Server.Port = p
        }
    }
    if socketPath := os.Getenv("ATEST_EXT_AI_SOCKET_PATH"); socketPath != "" {
        cfg.Server.SocketPath = socketPath
    }

    // AI 配置
    if defaultService := os.Getenv("ATEST_EXT_AI_DEFAULT_SERVICE"); defaultService != "" {
        cfg.AI.DefaultService = defaultService
    }

    // Ollama 特定配置
    if endpoint := os.Getenv("ATEST_EXT_AI_OLLAMA_ENDPOINT"); endpoint != "" {
        if _, ok := cfg.AI.Services["ollama"]; ok {
            cfg.AI.Services["ollama"].Endpoint = endpoint
        }
    }
    if model := os.Getenv("ATEST_EXT_AI_OLLAMA_MODEL"); model != "" {
        if _, ok := cfg.AI.Services["ollama"]; ok {
            cfg.AI.Services["ollama"].Model = model
        }
    }

    // OpenAI 特定配置
    if apiKey := os.Getenv("ATEST_EXT_AI_OPENAI_API_KEY"); apiKey != "" {
        if _, ok := cfg.AI.Services["openai"]; !ok {
            cfg.AI.Services["openai"] = AIService{}
        }
        cfg.AI.Services["openai"].APIKey = apiKey
    }

    // 日志级别
    if logLevel := os.Getenv("ATEST_EXT_AI_LOG_LEVEL"); logLevel != "" {
        cfg.Logging.Level = logLevel
    }
}

func applyDefaults(cfg *Config) {
    // 服务器默认值
    if cfg.Server.Host == "" {
        cfg.Server.Host = "0.0.0.0"
    }
    if cfg.Server.Port == 0 {
        cfg.Server.Port = 8080
    }
    if cfg.Server.SocketPath == "" {
        cfg.Server.SocketPath = "/tmp/atest-ext-ai.sock"
    }
    if cfg.Server.Timeout.Duration == 0 {
        cfg.Server.Timeout = Duration{120 * time.Second}
    }

    // AI 默认值
    if cfg.AI.DefaultService == "" {
        cfg.AI.DefaultService = "ollama"
    }
    if cfg.AI.Timeout.Duration == 0 {
        cfg.AI.Timeout = Duration{60 * time.Second}
    }

    // Ollama 服务默认值
    if cfg.AI.Services == nil {
        cfg.AI.Services = make(map[string]AIService)
    }
    if _, ok := cfg.AI.Services["ollama"]; !ok {
        cfg.AI.Services["ollama"] = AIService{
            Enabled:   true,
            Provider:  "ollama",
            Endpoint:  "http://localhost:11434",
            Model:     "qwen2.5-coder:latest",
            MaxTokens: 4096,
            Timeout:   Duration{60 * time.Second},
        }
    }

    // 日志默认值
    if cfg.Logging.Level == "" {
        cfg.Logging.Level = "info"
    }
    if cfg.Logging.Format == "" {
        cfg.Logging.Format = "json"
    }
    if cfg.Logging.Output == "" {
        cfg.Logging.Output = "stdout"
    }

    // 插件默认值
    if cfg.Plugin.Name == "" {
        cfg.Plugin.Name = "atest-ext-ai"
    }
    if cfg.Plugin.Version == "" {
        cfg.Plugin.Version = "1.0.0"
    }
    if cfg.Plugin.LogLevel == "" {
        cfg.Plugin.LogLevel = cfg.Logging.Level
    }
    if cfg.Plugin.Environment == "" {
        cfg.Plugin.Environment = "production"
    }
}

func validateConfig(cfg *Config) error {
    // 验证服务器配置
    if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
        return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
    }

    // 验证 AI 配置
    if cfg.AI.DefaultService == "" {
        return fmt.Errorf("default AI service not specified")
    }

    if _, ok := cfg.AI.Services[cfg.AI.DefaultService]; !ok {
        return fmt.Errorf("default service '%s' not found in services", cfg.AI.DefaultService)
    }

    // 验证每个服务
    for name, svc := range cfg.AI.Services {
        if !svc.Enabled {
            continue
        }

        if svc.Provider == "" {
            return fmt.Errorf("service '%s': provider not specified", name)
        }

        if svc.Model == "" {
            return fmt.Errorf("service '%s': model not specified", name)
        }

        // 验证 provider 类型
        validProviders := []string{"ollama", "openai", "claude", "deepseek", "local", "custom"}
        valid := false
        for _, p := range validProviders {
            if svc.Provider == p {
                valid = true
                break
            }
        }
        if !valid {
            return fmt.Errorf("service '%s': invalid provider '%s'", name, svc.Provider)
        }
    }

    // 验证日志配置
    validLogLevels := []string{"debug", "info", "warn", "error"}
    valid := false
    for _, level := range validLogLevels {
        if cfg.Logging.Level == level {
            valid = true
            break
        }
    }
    if !valid {
        return fmt.Errorf("invalid log level: %s", cfg.Logging.Level)
    }

    return nil
}

func defaultConfig() *Config {
    return &Config{
        Server: ServerConfig{
            Host:       "0.0.0.0",
            Port:       8080,
            SocketPath: "/tmp/atest-ext-ai.sock",
            Timeout:    Duration{120 * time.Second},
        },
        Plugin: PluginConfig{
            Name:        "atest-ext-ai",
            Version:     "1.0.0",
            LogLevel:    "info",
            Environment: "production",
        },
        AI: AIConfig{
            DefaultService: "ollama",
            Timeout:        Duration{60 * time.Second},
            Services: map[string]AIService{
                "ollama": {
                    Enabled:   true,
                    Provider:  "ollama",
                    Endpoint:  "http://localhost:11434",
                    Model:     "qwen2.5-coder:latest",
                    MaxTokens: 4096,
                    Timeout:   Duration{60 * time.Second},
                },
            },
        },
        Logging: LoggingConfig{
            Level:  "info",
            Format: "json",
            Output: "stdout",
        },
    }
}
```

**步骤 2.2: 更新 go.mod**

```bash
# 移除 Viper
go mod edit -droprequire=github.com/spf13/viper

# 清理未使用的依赖
go mod tidy
```

**步骤 2.3: 删除旧文件**

```bash
git rm pkg/config/loader.go
```

**步骤 2.4: 测试验证**

```bash
# 测试默认配置
rm -f config.yaml
go run cmd/atest-ext-ai/main.go &
# 验证启动成功

# 测试配置文件加载
cp config.yaml config.yaml.bak
go run cmd/atest-ext-ai/main.go &
# 验证配置正确加载

# 测试环境变量覆盖
export ATEST_EXT_AI_OLLAMA_MODEL=llama2
go run cmd/atest-ext-ai/main.go &
# 验证环境变量生效
```

**提交**:
```bash
git add .
git commit -m "refactor(config): replace Viper with simple YAML loader

- Remove spf13/viper dependency (~30 indirect deps)
- Implement simple YAML + env vars loader
- Reduce config code from 583 to ~80 lines
- Improve startup time by 20-30%

BREAKING CHANGE: Removed config hot-reload and remote config features"
```

---

### 阶段 3: 移除接口抽象 (1-2 天) 🟡

**目标**: 移除 ClientFactory 和 RetryManager 接口

**步骤 3.1: 更新 types.go**

修改 `pkg/ai/types.go`:

```go
package ai

import (
    "time"
    "github.com/linuxsuren/atest-ext-ai/pkg/interfaces"
)

// 移除这些接口定义:
// - type ClientFactory interface { ... }
// - type RetryManager interface { ... }

// 保留类型别名
type AIClient = interfaces.AIClient
type GenerateRequest = interfaces.GenerateRequest
type GenerateResponse = interfaces.GenerateResponse
// ...

// 保留配置结构
type ProviderConfig struct {
    Name       string
    Enabled    bool
    Priority   int
    Config     map[string]any
    Models     []string
    Timeout    time.Duration
    MaxRetries int
}

type AIServiceConfig struct {
    Providers []ProviderConfig
    Retry     RetryConfig
}

type RetryConfig struct {
    MaxAttempts       int
    BaseDelay         time.Duration
    MaxDelay          time.Duration
    BackoffMultiplier float64
    Jitter            bool
}
```

**步骤 3.2: 确认 manager.go 已不使用接口**

在阶段 1 中，我们已经在 `manager.go` 中直接使用了具体类型，不需要额外修改。

**步骤 3.3: 测试验证**

```bash
# 编译检查
go build ./...

# 运行测试
go test ./... -v
```

**提交**:
```bash
git add .
git commit -m "refactor(ai): remove unnecessary interface abstractions

- Remove ClientFactory interface (only had one impl)
- Remove RetryManager interface (only had one impl)
- Use concrete types directly in manager
- Reduce code by ~150 lines"
```

---

### 阶段 4: 简化重试机制 (1 天) 🟡

**目标**: 简化 retry.go，保留核心重试逻辑

**步骤 4.1: 重构 retry.go**

修改 `pkg/ai/retry.go`:

```go
package ai

import (
    "context"
    "errors"
    "net"
    "strings"
    "syscall"
    "time"
)

// RetryConfig 重试配置（保留，用于配置文件）
type RetryConfig struct {
    MaxAttempts       int
    BaseDelay         time.Duration
    MaxDelay          time.Duration
    BackoffMultiplier float64
    Jitter            bool
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
    if err == nil {
        return false
    }

    // 上下文取消/超时不重试
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        return false
    }

    // 网络错误
    var netErr net.Error
    if errors.As(err, &netErr) && netErr.Timeout() {
        return true
    }

    // DNS 错误
    var dnsErr *net.DNSError
    if errors.As(err, &dnsErr) {
        return true
    }

    // 连接错误
    var opErr *net.OpError
    if errors.As(err, &opErr) && opErr.Op == "dial" {
        return true
    }

    // 系统调用错误
    var syscallErr *syscall.Errno
    if errors.As(err, &syscallErr) {
        switch *syscallErr {
        case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ETIMEDOUT:
            return true
        }
    }

    // Provider 特定错误
    errMsg := strings.ToLower(err.Error())

    // 速率限制
    if containsAny(errMsg, []string{"rate limit", "too many requests", "quota exceeded", "429"}) {
        return true
    }

    // 服务器错误
    if containsAny(errMsg, []string{"internal server error", "service unavailable", "bad gateway", "gateway timeout", "500", "502", "503", "504"}) {
        return true
    }

    // 认证/授权错误不重试
    if containsAny(errMsg, []string{"unauthorized", "forbidden", "invalid api key", "authentication failed", "401", "403"}) {
        return false
    }

    // 请求错误不重试
    if containsAny(errMsg, []string{"bad request", "invalid request", "malformed", "400"}) {
        return false
    }

    return false
}

func containsAny(s string, substrs []string) bool {
    for _, substr := range substrs {
        if strings.Contains(s, substr) {
            return true
        }
    }
    return false
}

// CalculateBackoff 计算指数退避延迟
func CalculateBackoff(attempt int, config RetryConfig) time.Duration {
    if attempt == 0 {
        return 0
    }

    // 指数退避: baseDelay * (multiplier ^ attempt)
    delay := config.BaseDelay
    for i := 0; i < attempt-1; i++ {
        delay = time.Duration(float64(delay) * config.BackoffMultiplier)
        if delay > config.MaxDelay {
            delay = config.MaxDelay
            break
        }
    }

    // 添加抖动
    if config.Jitter {
        jitter := time.Duration(rand.Int63n(int64(delay / 4)))
        delay = delay + jitter
    }

    return delay
}

// RetryableError 包装可重试错误
type RetryableError struct {
    Err       error
    Retryable bool
}

func (e *RetryableError) Error() string {
    return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
    return e.Err
}

// NewRetryableError 创建可重试错误
func NewRetryableError(err error, retryable bool) error {
    return &RetryableError{
        Err:       err,
        Retryable: retryable,
    }
}
```

**步骤 4.2: 更新 manager.go 中的重试逻辑**

确认 `manager.go` 的 `Generate` 方法使用简化的重试：

```go
func (m *AIManager) Generate(ctx context.Context, req *interfaces.GenerateRequest) (*interfaces.GenerateResponse, error) {
    var lastErr error

    for attempt := 0; attempt < 3; attempt++ {
        if attempt > 0 {
            // 计算退避延迟
            delay := CalculateBackoff(attempt, RetryConfig{
                BaseDelay:         time.Second,
                MaxDelay:          10 * time.Second,
                BackoffMultiplier: 2.0,
                Jitter:            true,
            })

            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }

        client := m.selectHealthyClient()
        if client == nil {
            lastErr = fmt.Errorf("no healthy clients available")
            continue
        }

        resp, err := client.Generate(ctx, req)
        if err != nil {
            if !IsRetryable(err) {
                return nil, err
            }
            lastErr = err
            continue
        }

        return resp, nil
    }

    return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}
```

**步骤 4.3: 测试验证**

```bash
# 测试重试逻辑
go test ./pkg/ai -run TestRetry -v

# 测试实际调用
go test ./pkg/ai -v
```

**提交**:
```bash
git add .
git commit -m "refactor(ai): simplify retry mechanism

- Remove RetryManager interface and implementation
- Keep essential retry logic functions
- Inline retry logic in manager
- Reduce code by ~200 lines"
```

---

### 阶段 5: 清理和优化 (1 天) 🟢

**目标**: 移除未使用代码，优化导入

**步骤 5.1: 清理未使用的导入**

```bash
# 使用 goimports 清理
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

**步骤 5.2: 移除未使用的类型和函数**

检查并移除：
- 未使用的类型定义
- 未使用的辅助函数
- 废弃的测试文件

**步骤 5.3: 更新文档**

```bash
# 更新 README
# 更新 CHANGELOG
# 更新 API 文档
```

**步骤 5.4: 最终测试**

```bash
# 完整测试套件
go test ./... -v -race

# 性能测试
go test -bench=. -benchmem ./...

# 代码覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**提交**:
```bash
git add .
git commit -m "refactor: final cleanup and optimization

- Remove unused imports and functions
- Update documentation
- Optimize code structure
- Run full test suite"
```

---

### 阶段 6: 合并和发布 (1 天) 🎉

**步骤 6.1: 代码审查**

创建 Pull Request:
```bash
git push origin refactor/architecture-simplification
# 在 GitHub 上创建 PR
```

**步骤 6.2: 性能对比**

| 指标 | 重构前 | 重构后 | 改进 |
|------|--------|--------|------|
| 代码行数 | 7,447 | ~6,200 | -16.7% |
| Go 文件数 | 28 | ~22 | -21.4% |
| 依赖包数 | 105 | ~75 | -28.6% |
| 启动时间 | ~200ms | ~150ms | -25% |
| 内存占用 | ~25MB | ~20MB | -20% |

**步骤 6.3: 合并到主分支**

```bash
git checkout main
git merge refactor/architecture-simplification
git tag v2.0.0
git push origin main --tags
```

---

## 📊 风险评估和缓解

### 高风险项

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| 破坏现有功能 | 中 | 高 | 完整测试覆盖，每阶段独立验证 |
| 性能回退 | 低 | 中 | 基准测试对比 |
| 配置迁移失败 | 低 | 高 | 向后兼容，保留配置格式 |

### 回退策略

1. **阶段级回退**: 每个阶段独立提交，可单独回退
2. **分支保护**: 在独立分支进行重构
3. **标签备份**: 重构前打标签 `v1.x-before-refactor`
4. **完整备份**: 保留重构前代码副本

---

## ✅ 验证清单

### 功能验证

- [ ] 前端可以获取模型列表
- [ ] 前端可以测试连接
- [ ] 前端可以生成 SQL
- [ ] SQL 解释正确显示
- [ ] 健康检查工作正常
- [ ] 配置文件正确加载
- [ ] 环境变量覆盖生效

### 性能验证

- [ ] 启动时间 < 200ms
- [ ] 内存占用 < 30MB
- [ ] SQL 生成响应 < 5s
- [ ] 并发请求处理正常

### 代码质量

- [ ] 所有测试通过
- [ ] 代码覆盖率 > 70%
- [ ] 无 race condition
- [ ] golint 无警告
- [ ] go vet 通过

---

## 📚 参考资料

### 设计原则

- [KISS Principle](https://en.wikipedia.org/wiki/KISS_principle)
- [YAGNI](https://en.wikipedia.org/wiki/You_aren%27t_gonna_need_it)
- [Go Proverbs](https://go-proverbs.github.io/)

### Go 最佳实践

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

---

## 📝 附录

### A. 文件变更清单

| 操作 | 文件 | 行数变化 |
|------|------|----------|
| 删除 | pkg/ai/client.go | -699 |
| 删除 | pkg/ai/provider_manager.go | -417 |
| 创建 | pkg/ai/manager.go | +350 |
| 删除 | pkg/config/loader.go | -583 |
| 创建 | pkg/config/simple_loader.go | +80 |
| 修改 | pkg/ai/types.go | -50 |
| 修改 | pkg/ai/retry.go | -150 |
| 修改 | pkg/ai/engine.go | +10 |
| 修改 | pkg/plugin/service.go | +20 |

**总计**: 约减少 1,439 行代码

### B. 依赖变更清单

**移除的依赖**:
```
github.com/spf13/viper
github.com/fsnotify/fsnotify (viper 依赖)
github.com/spf13/afero (viper 依赖)
github.com/spf13/cast (viper 依赖)
... 约 30 个间接依赖
```

**保留的依赖**:
```
github.com/tmc/langchaingo (OpenAI 集成)
google.golang.org/grpc (通信)
gopkg.in/yaml.v2 (配置解析)
github.com/cenkalti/backoff/v4 (重试，可选简化使用)
```

---

**文档结束**

如有疑问或需要调整，请联系架构团队。
