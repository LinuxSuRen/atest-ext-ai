# API Testing AI插件开发指南

## 概述

API Testing项目支持AI插件扩展，AI插件与其他插件使用相同的扩展系统架构。本文档说明如何开发一个符合标准的AI插件。

## 架构说明

AI插件使用API Testing统一的插件系统架构：
- 通过Unix Socket与主程序通信
- 使用标准的`testing.Loader.Query(map[string]string)`接口
- 通过`categories: ["ai"]`标识为AI插件
- 采用JSON消息协议进行通信

## 插件配置

### 插件描述文件 (extension.yaml)

```yaml
items:
  - name: your-ai-plugin
    categories: 
      - ai                    # 必须包含"ai"类别
    dependencies:
      - name: your-ai-plugin
    link: https://github.com/yourorg/your-ai-plugin
```

## 接口规范

AI插件必须实现以下两个方法：

### 1. ai.generate - 生成内容

**请求参数** (通过`Query(map[string]string)`传入):
```go
map[string]string{
    "method": "ai.generate",              // 必需：方法名
    "model":  "gpt-4",                    // 必需：模型标识符
    "prompt": "生成测试用例...",           // 必需：提示词或指令
    "config": `{"temperature": 0.7}`      // 可选：JSON配置字符串
}
```

**响应格式** (通过`DataResult.Pairs`返回):
```go
map[string]string{
    "content": "生成的内容...",           // 必需：生成的内容
    "meta":    `{"tokens": 100}`,        // 可选：JSON元数据
    "success": "true"                     // 必需：是否成功
}
```

### 2. ai.capabilities - 获取插件能力

**请求参数**:
```go
map[string]string{
    "method": "ai.capabilities"           // 必需：方法名
}
```

**响应格式**:
```go
map[string]string{
    "capabilities": `{                    // 推荐：完整能力描述JSON
        "models": ["gpt-4", "gpt-3.5"],
        "features": ["text-generation", "code-generation"],
        "maxTokens": 4096,
        "supportedLanguages": ["zh", "en"]
    }`,
    "models":      `["gpt-4", "gpt-3.5"]`,   // 备选：支持的模型列表
    "features":    `["text-generation"]`,     // 备选：支持的功能列表
    "description": "AI代码生成插件",           // 备选：插件描述
    "version":     "1.0.0",                   // 备选：插件版本
    "success":     "true"                     // 必需：是否成功
}
```

### 错误处理

当发生错误时，返回格式：
```go
map[string]string{
    "error":   "具体的错误信息",
    "success": "false"
}
```

## 实现示例

### Go语言实现示例

```go
package main

import (
    "encoding/json"
    "fmt"
    "github.com/linuxsuren/api-testing/pkg/testing"
)

type AIPlugin struct {
    // 你的AI客户端
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
        // 处理生成请求
        model := query["model"]
        prompt := query["prompt"]
        configJSON := query["config"]
        
        // 解析配置
        var config map[string]interface{}
        if configJSON != "" {
            json.Unmarshal([]byte(configJSON), &config)
        }
        
        // 调用AI服务
        content, meta, err := p.client.Generate(model, prompt, config)
        if err != nil {
            result.Pairs["error"] = err.Error()
            result.Pairs["success"] = "false"
            return result, nil
        }
        
        // 返回成功结果
        result.Pairs["content"] = content
        if meta != nil {
            metaJSON, _ := json.Marshal(meta)
            result.Pairs["meta"] = string(metaJSON)
        }
        result.Pairs["success"] = "true"
        
    case "ai.capabilities":
        // 返回插件能力
        capabilities := map[string]interface{}{
            "models":             []string{"gpt-4", "gpt-3.5-turbo"},
            "features":           []string{"text-generation", "code-generation"},
            "maxTokens":          4096,
            "supportedLanguages": []string{"zh", "en"},
        }
        
        capJSON, _ := json.Marshal(capabilities)
        result.Pairs["capabilities"] = string(capJSON)
        result.Pairs["success"] = "true"
        
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

## 主程序调用方式

主程序通过以下方式调用AI插件：

### 1. HTTP API调用

```bash
# 获取AI插件能力
GET /api/v1/ai/capabilities/{plugin_name}

# 生成内容
POST /api/v1/ai/generate
{
    "plugin_name": "your-ai-plugin",
    "model": "gpt-4",
    "prompt": "生成一个用户登录的测试用例",
    "config": "{\"temperature\": 0.7, \"max_tokens\": 1000}"
}
```

### 2. gRPC调用

```protobuf
// 调用AI生成
rpc CallAI(AIRequest) returns (AIResponse);

// 获取AI能力
rpc GetAICapabilities(AICapabilitiesRequest) returns (AICapabilitiesResponse);
```

### 3. 内部代码调用

```go
// 获取AI插件
stores, err := server.GetStores(ctx, &SimpleQuery{Kind: "ai"})

// 调用AI插件
loader, err := server.getLoaderByStoreName("your-ai-plugin")
result, err := loader.Query(map[string]string{
    "method": "ai.generate",
    "model":  "gpt-4",
    "prompt": "生成测试用例",
    "config": `{"temperature": 0.7}`,
})
content := result.Pairs["content"]
```

## 配置参数说明

### config字段（JSON格式）

常用配置参数示例：
```json
{
    "temperature": 0.7,        // 生成随机性 (0-1)
    "max_tokens": 1000,        // 最大生成长度
    "top_p": 0.9,             // 核采样参数
    "frequency_penalty": 0,    // 频率惩罚
    "presence_penalty": 0,     // 存在惩罚
    "stop": ["\n\n"],         // 停止序列
    "system": "你是一个测试专家" // 系统提示词
}
```

## 最佳实践

1. **错误处理**：始终返回清晰的错误信息，帮助调试
2. **配置验证**：验证必需参数，提供合理的默认值
3. **异步处理**：对于长时间运行的任务，考虑异步实现
4. **日志记录**：记录关键操作和错误，便于问题定位
5. **资源管理**：正确管理AI客户端连接和资源释放
6. **版本兼容**：在capabilities中声明版本，保证向后兼容

## 测试插件

### 1. 单元测试

```go
func TestAIPlugin_Generate(t *testing.T) {
    plugin := &AIPlugin{}
    result, err := plugin.Query(map[string]string{
        "method": "ai.generate",
        "model":  "gpt-4",
        "prompt": "测试提示词",
    })
    
    assert.NoError(t, err)
    assert.Equal(t, "true", result.Pairs["success"])
    assert.NotEmpty(t, result.Pairs["content"])
}
```

### 2. 集成测试

使用curl测试HTTP接口：
```bash
# 测试能力获取
curl -X GET "http://localhost:8080/api/v1/ai/capabilities/your-ai-plugin"

# 测试内容生成
curl -X POST "http://localhost:8080/api/v1/ai/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_name": "your-ai-plugin",
    "model": "gpt-4",
    "prompt": "Hello"
  }'
```

## 常见问题

### Q1: 如何处理流式响应？
A: 当前接口设计为同步调用，如需流式响应，可以：
1. 在meta中返回任务ID
2. 提供额外的查询接口获取流式结果
3. 或等待后续版本支持

### Q2: 如何支持多模态输入？
A: 可以在prompt中使用特殊格式，如：
```json
{
    "prompt": "[IMAGE:base64_data] 描述这张图片",
    "config": "{\"input_type\": \"multimodal\"}"
}
```

### Q3: 如何处理上下文对话？
A: 在config中传递会话ID和历史消息：
```json
{
    "config": "{\"session_id\": \"xxx\", \"history\": [...]}"
}
```

## 参考资源

- [API Testing扩展开发文档](https://github.com/LinuxSuRen/api-testing/docs)
- [示例AI插件实现](https://github.com/LinuxSuRen/atest-ext-ai-example)
- [主项目源码](https://github.com/LinuxSuRen/api-testing)

## 联系方式

如有问题，请通过以下方式联系：
- GitHub Issues: https://github.com/LinuxSuRen/api-testing/issues
- 邮件列表: api-testing-dev@googlegroups.com