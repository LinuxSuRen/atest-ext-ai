---
name: atest-ext-ai-plugin-part
status: backlog
created: 2025-09-15T07:57:30Z
progress: 0%
prd: .claude/prds/atest-ext-ai-plugin-part.md
github: https://github.com/KariHall619/atest-ext-ai/issues/1
---

# Epic: atest-ext-ai-plugin-part

## Overview

基于AI_PLUGIN_DEVELOPMENT.md文档标准，实现符合标准插件系统架构的AI内容生成插件。插件采用`testing.Loader.Query(map[string]string)`接口，通过`ai.generate`和`ai.capabilities`两个核心方法提供AI服务。遵循单一职责原则：专注于内容生成，SQL执行由主程序处理。支持本地AI模型(Ollama)和在线AI服务，通过Unix socket通信。

## Architecture Decisions

### 标准插件架构选择
- **接口标准化**: 使用`testing.Loader`接口而非AI特定接口，简化集成
- **通信协议**: JSON消息通过map[string]string传递，避免复杂protobuf定义
- **职责分离**: AI插件只负责内容生成，数据库操作由主程序处理
- **插件标识**: 通过`categories: ["ai"]`标识插件类型
- **二进制命名**: 严格遵循`atest-store-ai`命名规范

### AI服务集成策略
- **本地优先**: 支持Ollama本地模型，降低成本和延迟
- **在线备选**: 集成OpenAI、Claude等在线服务作为补充
- **配置灵活**: 支持环境变量、配置文件和动态参数多种配置方式
- **错误处理**: 统一的成功/失败标识和错误信息格式

### 技术栈选择
- **Go 1.19+**: 与主项目保持一致的Go版本
- **Unix Socket**: 标准的本地进程间通信方式
- **JSON协议**: 简单、可读性强的数据交换格式
- **模块化设计**: AI引擎、配置管理、插件服务分离

## Technical Approach

### Backend Services

#### 1. 标准Loader接口实现
```go
type AIPlugin struct {
    aiEngine *ai.Engine
    config   *config.Config
}

// 实现testing.Loader接口
func (p *AIPlugin) Query(query map[string]string) (testing.DataResult, error) {
    method := query["method"]
    switch method {
    case "ai.generate":
        return p.handleGenerate(query)
    case "ai.capabilities":
        return p.handleCapabilities(query)
    default:
        return p.errorResponse(fmt.Sprintf("不支持的方法: %s", method))
    }
}
```

#### 2. AI内容生成引擎
- **自然语言处理**: 解析用户输入的自然语言描述
- **SQL生成**: 基于数据库schema和用户需求生成SQL
- **多数据库支持**: 针对MySQL、PostgreSQL、SQLite生成适配的SQL
- **质量保证**: 语法检查、性能建议、详细注释

#### 3. AI服务抽象层
```go
type AIClient interface {
    Generate(model, prompt string, config map[string]interface{}) (string, map[string]interface{}, error)
    GetCapabilities() (*Capabilities, error)
}

// 支持多种AI服务
type OllamaClient struct { endpoint string }
type OpenAIClient struct { apiKey string }
type ClaudeClient struct { apiKey string }
```

### Infrastructure

#### 1. 插件服务架构
```go
// Unix socket服务器
func main() {
    socketPath := "/tmp/atest-store-ai.sock"
    os.Remove(socketPath)

    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        log.Fatalf("Failed to listen on socket: %v", err)
    }
    defer listener.Close()

    // 启动HTTP服务处理Loader请求
    server := &http.Server{
        Handler: NewAIPluginHandler(),
    }
    server.Serve(listener)
}
```

#### 2. 配置管理系统
- **层级配置**: 环境变量 > 配置文件 > 默认值
- **热重载**: 支持配置文件变更时自动重载
- **敏感信息**: API密钥等通过环境变量管理
- **验证机制**: 配置项合法性检查

#### 3. 部署和监控
- **容器化**: Docker支持，便于部署和扩展
- **健康检查**: 提供健康状态API供主程序监控
- **日志管理**: 结构化日志，支持不同级别输出
- **性能监控**: AI服务调用耗时、成功率统计

## Implementation Strategy

### 开发阶段规划
1. **Phase 1 (2周)**: 实现标准Loader接口和ai.generate基础功能
2. **Phase 2 (2周)**: 完善AI服务集成、配置管理和ai.capabilities
3. **Phase 3 (1周)**: 集成测试、性能优化、文档完善

### 风险缓解策略
- **接口简化**: 使用标准接口降低集成复杂度
- **职责单一**: 避免复杂的数据库操作逻辑
- **渐进开发**: 先实现核心功能，再逐步完善
- **充分测试**: 单元测试、集成测试、性能测试全覆盖

### 质量保证
- **代码规范**: 严格遵循Go代码规范和项目约定
- **错误处理**: 完善的错误处理和用户友好提示
- **性能优化**: AI调用优化、内存使用控制
- **安全考虑**: 输入验证、权限控制、敏感信息保护

## Task Breakdown Preview

基于标准插件架构，将实现任务简化为**5个核心类别**：

### 1. 插件接口实现
- [ ] 实现testing.Loader接口的所有方法
- [ ] 实现ai.generate方法处理SQL生成请求
- [ ] 实现ai.capabilities方法返回插件能力
- [ ] Unix socket通信服务器搭建

### 2. AI服务集成
- [ ] Ollama本地模型客户端实现
- [ ] OpenAI在线服务客户端实现
- [ ] AI服务抽象层设计
- [ ] 多数据库SQL生成逻辑

### 3. 配置管理
- [ ] 配置文件结构设计和解析
- [ ] 环境变量支持和优先级
- [ ] AI服务配置管理
- [ ] 配置验证和热重载

### 4. 集成测试
- [ ] 插件接口单元测试
- [ ] AI服务集成测试
- [ ] Unix socket通信测试
- [ ] 性能基准测试

### 5. 部署优化
- [ ] Dockerfile和构建脚本
- [ ] 日志和监控集成
- [ ] 错误处理和恢复
- [ ] 文档和使用指南

## Dependencies

### 关键外部依赖
- **主项目插件系统**: 主项目的标准插件发现和管理机制
- **Ollama服务**: 本地AI模型运行环境
- **在线AI服务**: OpenAI、Claude等API服务可用性
- **数据库驱动**: Go标准数据库驱动包

### 消除的依赖(通过架构简化)
- ~~复杂protobuf定义~~: 使用简单JSON协议
- ~~数据库连接管理~~: 由主程序处理
- ~~HTTP API开发~~: 使用标准插件接口
- ~~前端组件开发~~: 主项目已提供

### 前置条件
- 开发环境安装Go 1.19+
- 本地Ollama环境搭建和模型配置
- 在线AI服务API密钥获取
- 主项目测试环境准备

## Success Criteria (Technical)

### 核心功能验收
- **插件发现**: 二进制`atest-store-ai`被主项目自动发现并加载
- **接口合规**: 完全实现testing.Loader接口规范
- **方法响应**: ai.generate和ai.capabilities方法正确处理请求
- **通信稳定**: Unix socket通信稳定可靠
- **错误处理**: 统一的错误格式和成功标识

### 性能基准
- AI处理响应时间 < 30s (包含网络延迟)
- 本地模型响应时间 < 10s (优先目标)
- 插件启动发现时间 < 2s
- 内存使用增量 < 100MB
- 并发请求支持 >= 10个

### 质量指标
- SQL生成准确率 >= 85%
- 单元测试覆盖率 >= 90%
- 集成测试通过率 = 100%
- 错误率 < 1%
- 可用性 >= 99.5%

## Tasks Created
- [ ] #2 - 实现标准Loader接口架构 (parallel: true)
- [ ] #3 - 实现ai.generate方法 (parallel: false)
- [ ] #4 - 实现ai.capabilities方法 (parallel: false)  
- [ ] #5 - AI服务抽象层设计 (parallel: true)
- [ ] #6 - Ollama和在线AI服务集成 (parallel: false)
- [ ] #7 - 配置管理系统实现 (parallel: true)
- [ ] #8 - 集成测试套件 (parallel: false)
- [ ] #9 - 部署和文档完善 (parallel: false)

Total tasks: 8
Parallel tasks: 3
Sequential tasks: 5
Estimated total effort: ~21 days (1 XL + 1 XL + 2 L + 2 M + 1 S)
## Estimated Effort

### 开发周期: 5周
- **Week 1-2**: 插件接口实现 + AI服务集成
- **Week 3-4**: 配置管理 + 集成测试
- **Week 5**: 部署优化 + 文档完善

### 资源需求: 1名Go工程师
- 熟悉Go开发和gRPC/HTTP服务
- 了解AI服务API调用
- 具备插件架构设计经验

### 关键里程碑
- **Week 2**: ai.generate基础功能可用
- **Week 4**: 完整插件功能集成测试通过
- **Week 5**: 生产环境部署就绪

### 风险评估: 低风险
- 架构简化降低技术复杂度
- 职责单一减少集成问题
- 标准接口提高兼容性
- 充分测试保证质量
