AI plugin for API-Testing[https://github.com/LinuxSuRen/api-testing].

## 功能
It can(now):
1. convert natural language to SQL.
2. generate test examples.(To-do)

目前支持的ai提供商:
云端:DeepSeek, OpenAI
本地:Ollama的任意模型(实现了本地模型自动发现)

## 开发命令

项目通过 `make` 提供常用的开发流程:
- `make build` 编译后端插件
- `make build-frontend` 构建前端资源
- `make test` 运行完整测试套件
- `make install-local` 重新打包并安装插件到 `~/.config/atest/bin`

使用 `make help` 可以查看全部可用的目标。
