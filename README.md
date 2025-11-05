AI plugin for API-Testing[https://github.com/LinuxSuRen/api-testing].

## 功能
It can(now):
1. convert natural language to SQL.
2. generate test examples.(To-do)

目前支持的ai提供商:
云端:DeepSeek, OpenAI
本地:Ollama的任意模型(实现了本地模型自动发现)

## 配置方式

插件由 API Testing 的 GUI 负责下发所有 AI 服务配置。在桌面端选择提供商、终端地址与模型后，插件会自动加载最新设置并刷新连接状态。无需在终端内设置任何环境变量；如需高级调试，可使用 CLI 环境变量，但未来版本可能移除该能力。

### 跨平台监听地址

- macOS / Linux 默认仍然使用 `unix:///tmp/atest-ext-ai.sock`，保持原有的安全隔离。
- Windows 会自动改用本地 TCP 地址 `127.0.0.1:38081`，无需手工处理 Unix 套接字。
- 如需手动覆盖，可设置：
  - `AI_PLUGIN_LISTEN_ADDR`：统一入口，支持 `unix:///path` 或 `tcp://host:port`；
  - 或在 Windows 下使用 `AI_PLUGIN_TCP_ADDR`，在类 Unix 系统使用 `AI_PLUGIN_SOCKET_PATH`。
- 主应用（API Testing）需要读取同样的地址后再去连接，建议在扩展配置里加一个“Windows 默认 TCP”说明。

## 开发命令

项目通过 `make` 提供常用的开发流程:
- `make build` 编译后端插件
- `make build-frontend` 构建前端资源
- `make test` 运行完整测试套件
- `make install-local` 重新打包并安装插件到 `~/.config/atest/bin`

使用 `make help` 可以查看全部可用的目标。

## 配置后端地址
默认情况下，插件会尝试连接 `http://127.0.0.1:8080`。如果后端运行在不同地址，可通过环境变量 `VITE_API_URL` 覆盖。例如：

```bash
export VITE_API_URL=http://localhost:8081
cd ./frontend
npm run dev
```
