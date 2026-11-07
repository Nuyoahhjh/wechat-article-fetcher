# 拾刊 V3

微信公众号内容采集控制台。平台侧使用 Go + SQLite，管理端使用 Vue 3 + Element Plus，
Windows Agent 使用 Python + windows-mcp + 视觉模型。

## 当前范围

已实现 P0/P1 的真实闭环和 P2 的主要运营能力：

- Agent 注册、Token 轮换、WebSocket 心跳与断线重连
- 任务队列、租约、自动调度、取消、重试与事件流
- `task.assigned` / `task.ack` / `task.progress` / `task.result` / `task.complete` 协议
- 微信登录请求、登录画面截图、扫码状态轮询
- 截图上传鉴权、截图持久化和真实 Agent 日志
- 文章 URL 规范化去重、正文/元数据状态、任务与 Agent 筛选
- 真实 Dashboard、文章详情、JSON/CSV/Markdown 导出
- 用户与角色管理、个人设置、密码修改、操作审计
- Docker Compose 部署配置

定时监控、单篇 URL 导入、批量账号任务、公众号档案、Webhook/OpenAPI、
PostgreSQL/Redis/对象存储仍属于产品规划中的 P3 扩展，不在当前 MVP 界面中伪装为已实现。

## 目录

```text
server/    Go API、WebSocket Hub、调度器、SQLite
agent/     Windows 微信桌面自动化 Agent
frontend/  Vue 管理端
scripts/   E2E 验收脚本
docs/      产品与技术文档
```

## 本地启动

### 1. 服务端

```powershell
cd D:\work\wechat-article-fetcher-v3\server
$env:JWT_SECRET='local-only-change-this-secret'
$env:AGENT_REGISTRATION_KEY='local-registration-key'
$env:BOOTSTRAP_ADMIN_PASSWORD='local-admin-password'
$env:CORS_ALLOWED_ORIGINS='http://localhost:5173'
go run ./cmd/server
```

默认监听 `http://127.0.0.1:8080`，数据库写入 `server/data/platform.db`。

### 2. 管理端

```powershell
cd D:\work\wechat-article-fetcher-v3\frontend
npm install
npm run dev
```

打开 `http://localhost:5173`，使用管理员账号登录。

### 3. Agent

```powershell
cd D:\work\wechat-article-fetcher-v3\agent
uv sync
Copy-Item .\config.example.yaml .\config.yaml
uv run wechat-agent --config .\config.yaml --check-mcp
.\scripts\run.ps1
```

至少配置：

- `WECHAT_AGENT_LLM_API_KEY`
- `WECHAT_AGENT_REGISTRATION_KEY`
- `WECHAT_PLATFORM_WS_URL`
- 本机微信路径 `wechat.executable_path`

Agent 首次注册后会将会话凭据写入本地数据目录；不要把 `.env`、`config.yaml` 中的密钥提交到仓库。

## 验收

平台协议、任务派发、扫码、截图、日志、入库、去重和 Dashboard 聚合可以脱离真实微信运行：

```powershell
uv run --project agent python scripts\e2e_mock_agent.py
```

真实公众号采集还需要在 Windows 桌面完成微信登录，并通过管理端或 API 创建任务。

## Docker

```powershell
Copy-Item .env.example .env
# 修改 JWT_SECRET、AGENT_REGISTRATION_KEY、BOOTSTRAP_ADMIN_PASSWORD
docker compose up -d --build
```

- 管理端：`http://localhost:3000`
- 服务端：`http://localhost:8080`

Docker 只部署平台和管理端。Windows Agent 必须运行在能够看到微信桌面窗口的 Windows 会话中。

## 文档

- [产品方案](docs/product-plan.md)
- [实现状态](docs/implementation-status.md)
- [REST API](server/docs/api.md)
- [Agent WebSocket 协议](server/docs/websocket-protocol.md)
- [Agent 使用说明](agent/README.md)
