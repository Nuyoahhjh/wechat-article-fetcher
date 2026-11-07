# WeChat Article Fetcher Platform

微信公众号文章自动化获取系统的 Golang 平台端。当前版本是一个可直接运行的单节点 MVP，包含 Agent 管理、WebSocket 通信、任务调度、文章存储、用户权限和微信扫码登录会话。

## 技术选型

- HTTP：`net/http` + `chi`
- WebSocket：`gorilla/websocket`
- 数据库：SQLite（默认，纯 Go 驱动，部署无 CGO 依赖）
- 鉴权：JWT + bcrypt
- 日志：标准库 `log/slog`
- Go 版本：`1.23+`

SQLite 适合单节点起步和联调。`store` 包已把 SQL 访问集中隔离，生产环境多实例部署时应替换为 PostgreSQL，并将调度器改为数据库抢占锁或独立 leader。

## 快速启动

在 PowerShell 中设置开发环境变量：

```powershell
$env:JWT_SECRET = "local-dev-jwt-secret-change-me-32chars"
$env:AGENT_REGISTRATION_KEY = "local-agent-registration-key"
$env:BOOTSTRAP_ADMIN_PASSWORD = "change-this-password"
$env:DB_PATH = "./data/platform.db"
go run ./cmd/server
```

服务默认监听 `http://127.0.0.1:8080`，健康检查为 `GET /healthz`。

首次启动且数据库中没有用户时，会创建管理员：

- 用户名：`BOOTSTRAP_ADMIN_USERNAME`，默认 `admin`
- 密码：`BOOTSTRAP_ADMIN_PASSWORD`，默认 `admin123!`

默认凭据仅用于开发。部署前必须修改密码和两个密钥。

## 登录与调用

```powershell
$loginBody = @{ username = "admin"; password = "change-this-password" } | ConvertTo-Json
$login = Invoke-RestMethod `
  -Uri "http://127.0.0.1:8080/api/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body $loginBody

$headers = @{ Authorization = "Bearer $($login.data.token)" }
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/v1/dashboard" -Headers $headers
```

## Agent 接入

Agent 首次注册：

```powershell
$body = @{
  name = "Windows Agent 01"
  machine_id = "WIN-MACHINE-UUID"
  hostname = "DESKTOP-01"
  os = "Windows 11"
  version = "0.1.0"
  capabilities = @{
    windows_mcp = $true
    vision_model = "qwen3.8-27b"
  }
} | ConvertTo-Json -Depth 5

$registration = Invoke-RestMethod `
  -Uri "http://127.0.0.1:8080/api/v1/agents/register" `
  -Method Post `
  -Headers @{ "X-Agent-Registration-Key" = "local-agent-registration-key" } `
  -ContentType "application/json" `
  -Body $body
```

返回的 `agent_token` 只展示一次。Agent 使用以下方式连接：

```text
GET /api/v1/agent/ws/{agent_id}
Authorization: Bearer <agent_token>
```

WebSocket 建立后，Agent 必须先发送 `agent.hello`。协议详见 [WebSocket 协议](docs/websocket-protocol.md)。

## 文档

- [平台架构设计](docs/architecture.md)
- [WebSocket 通信协议](docs/websocket-protocol.md)
- [REST API](docs/api.md)
- [数据库 Schema](docs/schema.md)

## 项目结构

```text
cmd/server/                进程入口
internal/app/              依赖装配
internal/config/           环境变量
internal/domain/           领域模型与状态
internal/protocol/         Agent/Platform 消息类型
internal/store/            SQLite 持久化与 migration
internal/service/          用户、Agent、任务、文章、登录服务
internal/ws/               WebSocket Hub 与消息分发
internal/scheduler/        周期调度与租约回收
internal/httpapi/          REST 路由、中间件、Handler
migrations source           store/migration.sql
docs/                      架构、协议、API、Schema
```

## 构建

```powershell
go build -o .\bin\server.exe .\cmd\server
```

## 生产注意事项

- 将 `JWT_SECRET`、`AGENT_REGISTRATION_KEY` 和初始管理员密码放入密钥管理系统。
- 在反向代理层启用 TLS、访问日志、请求体限制和登录限流。
- 多平台实例部署前切换 PostgreSQL，并为调度增加数据库 advisory lock 或独立 scheduler leader。
- 若 Agent 数量进入数百台，拆分连接网关和业务 API，并把 WebSocket 路由键写入共享消息总线。
- 二维码是敏感登录材料，API 只应通过 TLS 暴露；当前 MVP 在数据库中保存短期 QR data URL 内容。
