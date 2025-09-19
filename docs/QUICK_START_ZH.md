# atest-ext-ai 快速开始指南

## 5分钟快速上手

本指南帮助您在5分钟内快速部署和使用 atest-ext-ai AI插件。

## 前置条件

- Go 1.23+ 已安装
- 网络连接正常

## 方法一：使用 Ollama（推荐本地开发）

### 1. 安装 Ollama

```bash
# macOS/Linux
curl -fsSL https://ollama.ai/install.sh | sh

# 或下载安装包
# https://ollama.ai/download
```

### 2. 启动 Ollama 并下载模型

```bash
# 启动 Ollama 服务
ollama serve

# 新开终端，下载模型（推荐使用轻量级模型）
ollama pull llama2           # 7B模型，推荐
# ollama pull codellama      # 代码专用模型
# ollama pull mistral        # 轻量级选择
```

### 3. 快速安装插件

```bash
# 克隆项目
git clone https://github.com/linuxsuren/atest-ext-ai.git
cd atest-ext-ai

# 一键安装
./scripts/install.sh

# 或手动构建
go build -o bin/atest-store-ai ./cmd/atest-store-ai
```

### 4. 创建基础配置

```bash
mkdir -p config
cat > config/config.yaml << 'EOF'
server:
  host: "0.0.0.0"
  port: 8080
  timeout: "30s"

plugin:
  name: "atest-ext-ai"
  version: "1.0.0"
  debug: false
  log_level: "info"

ai:
  default_service: "ollama"
  services:
    ollama:
      enabled: true
      provider: "ollama"
      endpoint: "http://localhost:11434"
      model: "llama2"
      max_tokens: 4096
      temperature: 0.1
      timeout: "60s"

  cache:
    enabled: true
    ttl: "30m"
    max_size: 1000

logging:
  level: "info"
  format: "text"
  output: "stdout"
EOF
```

### 5. 启动服务

```bash
# 启动AI插件
./bin/atest-store-ai --config config/config.yaml

# 看到以下输出表示成功：
# Initializing AI plugin service...
# AI plugin service creation completed
# Server starting on :8080
```

### 6. 测试功能

```bash
# 测试健康状态
curl http://localhost:8080/health

# 测试AI能力
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "key": "capabilities",
    "sql": ""
  }'

# 测试SQL生成
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "key": "查找年龄大于30岁的所有用户",
    "sql": ""
  }'
```

## 方法二：使用 OpenAI（云服务）

### 1. 获取 OpenAI API Key

访问 [OpenAI Platform](https://platform.openai.com/) 获取API密钥

### 2. 设置环境变量

```bash
export OPENAI_API_KEY="your-api-key-here"
```

### 3. 创建配置文件

```bash
cat > config/config.yaml << 'EOF'
server:
  host: "0.0.0.0"
  port: 8080

plugin:
  name: "atest-ext-ai"
  version: "1.0.0"

ai:
  default_service: "openai"
  services:
    openai:
      enabled: true
      provider: "openai"
      model: "gpt-3.5-turbo"
      max_tokens: 4096
      temperature: 0.1
      timeout: "30s"
EOF
```

### 4. 启动并测试

```bash
./bin/atest-store-ai --config config/config.yaml
```

## 常用示例

### SQL 生成示例

```bash
# 基础查询
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "key": "查找所有活跃用户",
    "sql": ""
  }'

# 复杂查询
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "key": "统计每个月的订单数量和总金额",
    "sql": "SELECT * FROM orders"
  }'

# 联表查询
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ai",
    "key": "查找用户及其最新订单信息",
    "sql": ""
  }'
```

### 能力查询示例

```bash
# 查看所有能力
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"type": "ai", "key": "capabilities", "sql": ""}'

# 查看支持的模型
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"type": "ai", "key": "ai.capabilities.models", "sql": ""}'

# 查看插件元数据
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"type": "ai", "key": "ai.capabilities.metadata", "sql": ""}'
```

## Docker 快速部署

### 1. 使用预构建镜像

```bash
# 启动服务（使用Ollama）
docker run -d \
  --name atest-ai-plugin \
  -p 8080:8080 \
  -e OLLAMA_BASE_URL=http://host.docker.internal:11434 \
  atest-ext-ai:latest

# 启动服务（使用OpenAI）
docker run -d \
  --name atest-ai-plugin \
  -p 8080:8080 \
  -e OPENAI_API_KEY=your-api-key \
  atest-ext-ai:latest
```

### 2. 使用 Docker Compose

```bash
# 下载项目
git clone https://github.com/linuxsuren/atest-ext-ai.git
cd atest-ext-ai

# 启动完整环境（包括Ollama）
docker-compose up -d

# 仅启动AI插件
docker-compose up atest-ext-ai
```

## 集成到现有项目

### Go 项目集成

```go
import (
    "github.com/linuxsuren/atest-ext-ai/pkg/plugin"
)

// 创建AI插件客户端
aiService, err := plugin.NewAIPluginService()
if err != nil {
    log.Fatal(err)
}

// 使用AI生成SQL
result, err := aiService.Query(ctx, &server.DataQuery{
    Type: "ai",
    Key:  "查找活跃用户",
})
```

### Python 项目集成

```python
import requests

def generate_sql(query):
    response = requests.post('http://localhost:8080/query', json={
        'type': 'ai',
        'key': query,
        'sql': ''
    })
    return response.json()

# 使用
result = generate_sql("查找年龄大于25岁的用户")
print(result['data'][0]['value'])  # 生成的SQL
```

### JavaScript 项目集成

```javascript
async function generateSQL(query) {
    const response = await fetch('http://localhost:8080/query', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            type: 'ai',
            key: query,
            sql: ''
        })
    });
    return await response.json();
}

// 使用
generateSQL('查找所有用户').then(result => {
    console.log('生成的SQL:', result.data[0].value);
});
```

## 故障排除

### 常见问题速查

| 问题 | 解决方案 |
|------|----------|
| 端口8080被占用 | 修改配置文件中的端口号 |
| Ollama连接失败 | 确保Ollama服务运行：`ollama serve` |
| OpenAI API限制 | 检查API密钥和额度限制 |
| 内存不足 | 使用更小的模型或增加内存 |
| 响应缓慢 | 启用缓存或使用更快的模型 |

### 快速诊断命令

```bash
# 检查服务状态
curl -s http://localhost:8080/health | jq .

# 检查AI服务连接
curl -s http://localhost:11434/api/tags  # Ollama

# 查看实时日志
tail -f /var/log/atest-ext-ai.log

# 测试配置文件
./bin/atest-store-ai --config config/config.yaml --validate
```

## 下一步

现在您已经成功运行了 atest-ext-ai，可以：

1. 📖 阅读完整的[集成指南](AI_PLUGIN_INTEGRATION_GUIDE_ZH.md)
2. 🔧 探索[配置选项](CONFIGURATION.md)
3. 🎯 查看[使用示例](USER_GUIDE.md)
4. 🐛 [报告问题](https://github.com/linuxsuren/atest-ext-ai/issues)
5. 💬 加入[讨论社区](https://github.com/linuxsuren/atest-ext-ai/discussions)

祝您使用愉快！🚀