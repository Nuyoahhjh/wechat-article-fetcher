# 部署与运维

## 1. 部署拓扑

```text
浏览器
  │ HTTPS
  ▼
Nginx / Frontend
  │ /api/
  ▼
Go Platform ── SQLite
  ▲
  │ WSS
  │
Windows Agent ── Windows 微信桌面端
```

Agent 必须运行在有交互式桌面的 Windows 会话中。锁屏、断开远程桌面或不可见的
Windows Session 会导致微信 UI 自动化失败。

## 2. 生产环境变量

至少设置：

```text
APP_ENV=production
JWT_SECRET=<random-secret-at-least-24-chars>
AGENT_REGISTRATION_KEY=<random-registration-key>
BOOTSTRAP_ADMIN_PASSWORD=<strong-initial-password>
CORS_ALLOWED_ORIGINS=https://your-console.example.com
```

可选保留策略：

```text
ARTICLE_RETENTION_DAYS=0       # 0 表示永久保留
LOG_RETENTION_DAYS=30
SCREENSHOT_RETENTION_DAYS=7
```

生产环境必须使用 HTTPS/WSS。Agent Token、JWT Secret、注册密钥和模型 API Key
不得写入仓库或聊天记录。

## 3. Docker 部署

```powershell
Copy-Item .env.example .env
# 修改 .env
docker compose up -d --build
docker compose ps
```

健康检查：

```powershell
Invoke-RestMethod http://localhost:8080/healthz
```

只升级平台和管理端：

```powershell
docker compose build server frontend
docker compose up -d server frontend
```

Windows Agent 独立升级：

```powershell
cd agent
uv sync
uv run wechat-agent --config .\config.yaml --check-mcp
.\scripts\run.ps1
```

## 4. SQLite 备份

SQLite 使用 WAL。最稳妥的离线备份流程：

```powershell
docker compose stop server
docker run --rm -v wechat-article-fetcher-v3_server_data:/data `
  -v ${PWD}:/backup alpine `
  sh -c "cp /data/platform.db /backup/platform-$(Get-Date -Format yyyyMMdd-HHmmss).db"
docker compose start server
```

如果使用本机二进制而非 Docker，先停止服务，再复制 `platform.db`、`platform.db-wal`
和 `platform.db-shm`。恢复时同时停止服务并替换这三个文件。

## 5. 恢复与升级

1. 备份数据库。
2. 拉取新版本代码。
3. 执行服务端、Agent、管理端检查。
4. 启动新版本，服务端启动时会自动执行幂等 schema migration。
5. 登录管理端检查 Agent、任务、文章和审计日志。
6. 在目标 Windows 机器执行一次真实公众号采集回归。

回滚时同时回滚二进制与前端的版本，并使用升级前数据库备份。

## 6. 常见故障

| 现象 | 优先检查 |
| --- | --- |
| Agent 一直离线 | Registration Key、Agent Token、WSS 地址、证书和防火墙 |
| Agent 在线但任务不派发 | 微信登录状态、心跳时间、并发槽位、指定 Agent 是否禁用 |
| 二维码不显示 | Agent 是否能截图微信窗口，管理端是否使用 operator/admin |
| 任务没有终态 | WebSocket 断线、`task.complete` 消息、租约与调度日志 |
| 文章只有元数据 | Agent 未解析 URL 或未开启正文抓取，查看文章 `status` |
| 重复入库 | 检查 URL 规范化规则和数据库唯一索引 |
| 截图/日志消失 | `SCREENSHOT_RETENTION_DAYS`、`LOG_RETENTION_DAYS` |
