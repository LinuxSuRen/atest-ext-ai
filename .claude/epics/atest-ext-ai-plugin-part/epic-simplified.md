---
name: atest-ext-ai-plugin-part-simplified
status: backlog
created: 2025-09-15T10:00:00Z
progress: 0%
prd: .claude/prds/atest-ext-ai-plugin-part.md
github: [Will be updated when synced to GitHub]
---

# Epic: atest-ext-ai-plugin-part (简化版)

## Overview

**基于AI_PLUGIN_DEVELOPMENT.md文档**：AI插件采用标准插件系统架构，使用统一的`testing.Loader`接口。本Epic专注于实现`ai.generate`和`ai.capabilities`两个核心方法，插件二进制名称为`atest-store-ai`，通过Unix socket `/tmp/atest-store-ai.sock`与主系统通信。

## Architecture Decisions

### 标准插件系统架构
- ✅ **标准Loader接口**: 使用`testing.Loader.Query(map[string]string)`统一接口
- ✅ **AI插件标识**: 通过`categories: ["ai"]`标识插件类型
- ✅ **JSON消息协议**: 通过map[string]string传递参数和返回结果
- ✅ **Unix Socket通信**: 标准的Unix domain socket通信机制
- ✅ **单一职责原则**: AI插件只负责内容生成，不执行SQL

### 严格技术规范（不可更改）
- **二进制名称**: `atest-store-ai` (精确匹配，插件发现依赖此名称)
- **Socket路径**: `/tmp/atest-store-ai.sock` (Unix domain socket)
- **插件类型**: `categories: ["ai"]`标识
- **方法实现**: 必须实现`ai.generate`和`ai.capabilities`两个方法

## Technical Approach

### 核心实现方法

#### 1. 实现标准Loader接口
```go
type AIPlugin struct {
    client AIClient
}

// 实现testing.Loader接口的Query方法
func (p *AIPlugin) Query(query map[string]string) (result testing.DataResult, err error) {
    result = testing.DataResult{
        Pairs: make(map[string]string),
    }

    method := query["method"]

    switch method {
    case "ai.generate":
        // 处理SQL生成请求
        return p.handleGenerate(query)
    case "ai.capabilities":
        // 返回插件能力
        return p.handleCapabilities(query)
    default:
        result.Pairs["error"] = fmt.Sprintf("不支持的方法: %s", method)
        result.Pairs["success"] = "false"
    }

    return result, nil
}

// 其他必需的Loader接口方法
func (p *AIPlugin) HasMore() bool { return false }
func (p *AIPlugin) Load() ([]byte, error) { return nil, nil }
func (p *AIPlugin) Reset() {}
func (p *AIPlugin) Put([]byte) error { return nil }
```

#### 2. 实现核心AI方法
```go
// ai.generate - 生成SQL内容
func (p *AIPlugin) handleGenerate(query map[string]string) (testing.DataResult, error) {
    result := testing.DataResult{Pairs: make(map[string]string)}

    model := query["model"]
    prompt := query["prompt"]
    configJSON := query["config"]

    // 解析配置
    var config map[string]interface{}
    if configJSON != "" {
        json.Unmarshal([]byte(configJSON), &config)
    }

    // 调用AI服务生成SQL
    content, meta, err := p.client.Generate(model, prompt, config)
    if err != nil {
        result.Pairs["error"] = err.Error()
        result.Pairs["success"] = "false"
        return result, nil
    }

    result.Pairs["content"] = content
    if meta != nil {
        metaJSON, _ := json.Marshal(meta)
        result.Pairs["meta"] = string(metaJSON)
    }
    result.Pairs["success"] = "true"

    return result, nil
}

// ai.capabilities - 返回插件能力
func (p *AIPlugin) handleCapabilities(query map[string]string) (testing.DataResult, error) {
    result := testing.DataResult{Pairs: make(map[string]string)}

    capabilities := map[string]interface{}{
        "models":             []string{"gpt-4", "gpt-3.5-turbo", "llama2"},
        "features":           []string{"text-generation", "sql-generation"},
        "maxTokens":          4096,
        "supportedLanguages": []string{"zh", "en"},
    }

    capJSON, _ := json.Marshal(capabilities)
    result.Pairs["capabilities"] = string(capJSON)
    result.Pairs["success"] = "true"

    return result, nil
}
```

### 配置集成（extension.yaml格式）
```yaml
items:
  - name: atest-store-ai
    categories:
      - ai                    # 必须包含"ai"类别
    dependencies:
      - name: atest-store-ai
    link: https://github.com/yourorg/atest-ext-ai
```

### AI服务配置（多种方式）
```yaml
# ~/.atest/extensions/ai-plugin/config.yaml
ai:
  ollama:
    endpoint: "http://localhost:11434"
  openai:
    api_key: "${OPENAI_API_KEY}"  # 引用环境变量
```

## Implementation Strategy

### 实现策略（基于标准插件架构）
1. **遵循文档规范**: 严格按照AI_PLUGIN_DEVELOPMENT.md文档规范实现
2. **简化架构**: 只做AI内容生成，不做数据库操作
3. **标准接口**: 使用testing.Loader接口，不定义AI特定接口

### 风险缓解（简化架构降低风险）
- ✅ **接口简单**: 标准Loader接口，不需要复杂的protobuf定义
- ✅ **职责单一**: 只做内容生成，不涉及数据库操作
- ✅ **错误处理**: 统一的成功/失败标识，简化错误处理
- 🔄 **AI服务集成**: 需实现多种AI服务集成
- 🔄 **配置管理**: 需实现多级配置管理

### 测试策略（简化测试）
- ✅ **单元测试**: 测试ai.generate和ai.capabilities两个方法
- ✅ **集成测试**: 使用curl测试Unix socket通信
- 🔄 **AI服务测试**: 测试本地和在线AI服务集成

## Task Breakdown

### 极大简化任务列表

#### 🔄 需要实现（核心功能）
- [ ] **实现Loader接口**: 实现testing.Loader.Query(map[string]string)方法
- [ ] **AI方法实现**: 实现ai.generate和ai.capabilities两个方法
- [ ] **AI服务集成**: 集成Ollama和在线AI服务

#### ❌ 不需要实现（简化架构）
- ❌ **SQL执行引擎**: 由主程序处理
- ❌ **数据库连接管理**: 由主程序处理
- ❌ **复杂的gRPC方法**: 只需Query方法
- ❌ **自动降级机制**: 暂不实现
- ❌ **复杂的配置管理**: 使用简单的配置方式

**估计工作量进一步减少**: 从4-6周缩减至**3-4周**。

## Dependencies

### 关键外部依赖（大幅减少）
- ✅ **标准插件架构**: 主项目的标准插件系统已完成
- 🔄 **Ollama本地环境**: 需要本地Ollama服务可访问
- 🔄 **AI服务API**: 在线AI服务的API访问

### 消除的依赖（已由简化架构消除）
- ❌ ~~复杂protobuf定义~~: 使用简单的map[string]string
- ❌ ~~数据库驱动~~: 不在插件中管理数据库
- ❌ ~~SQL执行框架~~: 由主程序处理

## Success Criteria (Technical)

### 核心功能验收
- ✅ **插件发现**: 二进制名称`atest-store-ai`被主项目自动发现
- ✅ **Socket通信**: Unix socket `/tmp/atest-store-ai.sock` 正常建立连接
- ✅ **方法处理**: `method="ai.generate"`和`method="ai.capabilities"`请求正确处理
- ✅ **响应格式**: 返回正确的JSON格式响应
- ✅ **错误处理**: 统一的错误格式和成功标识

### 性能基准
- AI处理响应时间 < 30s
- 插件启动发现时间 < 2s
- 方法响应时间 < 500ms

## Estimated Effort

### 极大缩减的开发周期
**原估计**: 4-6周
**新估计**: **3-4周** (基于标准插件架构简化)

### 精简团队配置保持不变
**需求**: **1名Go工程师**

### 重新定义的阶段里程碑
1. **Week 1-2**: 实现Loader接口和ai.generate/ai.capabilities方法
2. **Week 2-3**: AI服务集成（Ollama和在线服务）
3. **Week 3-4**: 集成测试、配置优化和文档完善

---

*本Epic已基于AI_PLUGIN_DEVELOPMENT.md文档规范重新简化，专注于标准插件接口实现。*