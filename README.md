# 拾刊 V3

微信公众号内容采集控制台。平台侧使用 Go + SQLite，管理端使用 Vue 3 + Element Plus，
Windows Agent 使用 Python、`windows-mcp` 和 OpenAI 兼容模型完成桌面 UI 自动化与
文章元数据/正文提取。

> 当前为工程原型和单节点 MVP。项目不绕过微信登录、验证码、安全提醒、付费内容或隐私限制。
> 历史文章完整性取决于微信桌面端实际展示和本地缓存内容。

## 功能

- Agent 注册、Token 轮换、WebSocket 心跳、断线重连与消息确认
- 任务队列、租约、自动调度、取消、重试和完整事件流
- 微信登录请求、登录画面上传和扫码状态跟踪
- 截图鉴权上传、持久化、保留策略和真实 Agent 日志
- 公众号缓存优先读取：优先使用微信 `bizProfile` 元数据，必要时回退到 UI
- 文章 URL 去重、元数据/正文状态、筛选、详情和 JSON/CSV/Markdown 导出
- Dashboard、用户与角色管理、个人设置、密码修改和操作审计
- 单机 Docker Compose，以及 Linux/Windows 本地开发方式

定时采集、单篇 URL 导入、批量账号、公众号档案、Webhook/OpenAPI 和外部消息队列
属于后续规划，当前版本不会在界面中伪装为已实现。

## 架构

| 组件 | 技术栈 | 职责 |
| --- | --- | --- |
| Server | Go + chi + SQLite | REST API、WebSocket Hub、任务调度、文章入库 |
| Admin | Vue 3 + TypeScript + Element Plus | 登录、Dashboard、Agent/任务/文章管理 |
| Agent | Python + windows-mcp + LLM | 微信桌面自动化、缓存读取、文章提取 |

```text
Browser
  -> Vue Admin
  -> Go REST API / WebSocket
  -> SQLite
  <- Windows Agent (WebSocket)
       -> windows-mcp
       -> WeChat desktop
       -> optional WeChat bizProfile cache CLI
```

## 环境要求

- Go `1.23+`
- Node.js `20+` 和 npm
- Python `3.11+`、[uv](https://docs.astral.sh/uv/)
- Agent 运行机器：Windows 10/11、未锁屏的交互式桌面、已安装并登录的微信桌面端
- `windows-mcp` 可执行文件
- 一个支持结构化 JSON 输出的 OpenAI 兼容模型接口；需要读屏时应使用支持图像输入的模型
- 可选：单独安装兼容的微信 `bizProfile` 缓存 CLI，用于读取公众号文章元数据

## Docker 快速启动

```powershell
git clone https://github.com/Nuyoahhjh/wechat-article-fetcher.git
cd wechat-article-fetcher
Copy-Item .env.example .env
```

编辑 `.env`，至少设置强随机值：

```dotenv
JWT_SECRET=replace-with-a-long-random-secret
AGENT_REGISTRATION_KEY=replace-with-a-random-registration-key
BOOTSTRAP_ADMIN_PASSWORD=replace-with-a-strong-admin-password
```

启动平台和管理端：

```powershell
docker compose up -d --build
```

- 管理端：`http://localhost:3000`
- REST API：`http://localhost:8080/api/v1`
- 健康检查：`http://localhost:8080/healthz`

Docker 只运行 Server 和 Admin。Agent 必须运行在能够访问 Windows 桌面的机器上。

## 本地开发

### Server

```powershell
cd server
$env:JWT_SECRET = "<random-secret-at-least-32-characters>"
$env:AGENT_REGISTRATION_KEY = "<random-agent-registration-key>"
$env:BOOTSTRAP_ADMIN_PASSWORD = "<strong-admin-password>"
$env:CORS_ALLOWED_ORIGINS = "http://localhost:5173"
go run ./cmd/server
```

服务默认监听 `http://127.0.0.1:8080`，SQLite 数据默认写入 `server/data/platform.db`。

### Admin

```powershell
cd frontend
npm ci
npm run dev
```

开发地址默认是 `http://localhost:5173`，Vite 会将 `/api` 和 `/ws` 代理到
`http://127.0.0.1:8080`。

### Agent

```powershell
cd agent
uv sync
Copy-Item .\config.example.yaml .\config.yaml
```

在 `agent/.env.local` 中配置模型、平台和本机工具路径，然后执行：

```powershell
uv run wechat-agent --config .\config.yaml --check-mcp
.\scripts\run.ps1
```

完整配置项和 Windows 注意事项见 [Agent 使用说明](agent/README.md)。`config.yaml`、
`.env.local`、运行数据和日志都属于本机文件，不应提交到仓库。

## 缓存优先模式

Agent 会优先读取微信本地 `bizProfile`，直接获得公众号最近文章的标题、URL、发布时间
和摘要。配置独立缓存工具后：

```powershell
$env:WECHAT_CACHE_CLI_PATH = "<path-to>\wechat-article-fetcher\bin\wechat-fetch.ps1"
```

行为如下：

- 只采列表：缓存命中后直接回传，不逐屏扫描 UI。
- 需要正文：先尝试公开文章 URL，失败后再对缺失正文进入微信 UI。
- 缓存未命中：通过微信 UI 精确打开一次公众号主页，再回读缓存。
- 请求数量大于缓存数量：返回实际可用文章并标记为部分完成。
- 未配置缓存工具：保留原有 UI 自动化路径。

## 验收

协议、任务派发、截图、日志、入库、去重和 Dashboard 聚合可以脱离真实微信运行：

```powershell
uv run --project agent python scripts\e2e_mock_agent.py
```

真实公众号采集还需要在 Agent 所在 Windows 会话中登录微信，并通过管理端或 API
创建任务。

## 安全与合规

- 不要提交 `.env`、`.env.local`、`config.yaml`、Cookie、二维码、截图或数据库。
- 生产环境使用 HTTPS/WSS，并轮换 JWT、注册密钥和管理员密码。
- Agent 只处理当前可见、已授权的微信桌面内容。
- 登录、验证码和微信安全提示需要人工完成。
- 二维码、截图和日志属于敏感数据，应设置访问权限和保留周期。

## 文档

- [产品方案](docs/product-plan.md)
- [实现状态](docs/implementation-status.md)
- [部署与运维](docs/operations.md)
- [REST API](server/docs/api.md)
- [Agent WebSocket 协议](server/docs/websocket-protocol.md)
- [Server 说明](server/README.md)
- [Agent 使用说明](agent/README.md)
- [Admin 使用说明](frontend/README.md)

## License

当前仓库尚未添加开源许可证。对外分发或商用前请先补充合适的 `LICENSE` 文件。
