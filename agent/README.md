# 拾刊 Windows Agent

通过 `windows-mcp` 操作已登录的微信桌面端，用视觉模型观察屏幕、选择受限 UI 动作
并提取公众号文章。每台 Windows 机器运行一个 Agent，同一时刻只执行一个 UI 任务。

## 边界

- 只处理当前 Windows 交互式桌面，不支持锁屏或不可见 Session。
- 公众号名称必须精确匹配，不接受相似账号。
- 不绕过扫码、验证码、安全提醒、付费内容或隐私限制。
- 视觉模型只能返回白名单 UI 动作，不能生成任意系统命令。
- 历史列表完整性取决于微信客户端实际暴露的内容。

## 安装

```powershell
cd D:\work\wechat-article-fetcher-v3\agent
uv sync
Copy-Item .\config.example.yaml .\config.yaml
```

配置环境变量：

```powershell
$env:WECHAT_AGENT_LLM_API_KEY='<your-model-api-key>'
$env:WECHAT_AGENT_LLM_BASE_URL='https://your-openai-compatible-endpoint/v1'
$env:WECHAT_AGENT_LLM_MODEL='your-vision-model'
$env:WECHAT_AGENT_REGISTRATION_KEY='<platform-registration-key>'
$env:WECHAT_PLATFORM_WS_URL='ws://127.0.0.1:8080/api/v1/agent/ws'
$env:WECHAT_CACHE_CLI_PATH='D:\work\wechat-article-fetcher\bin\wechat-fetch.ps1'
```

同时确认：

- `wechat.executable_path` 指向本机微信可执行文件。
- `mcp.command` 指向 `windows-mcp.exe`。
- Windows 会话已登录微信。

检查 MCP：

```powershell
uv run wechat-agent --config .\config.yaml --check-mcp
```

启动：

```powershell
.\scripts\run.ps1
```

首次注册成功后，Agent 会把平台分配的 Agent ID 和 Token 写入
`runtime.data_dir\agent-credentials.json`。后续启动优先复用该凭据。

## 缓存优先采集

Agent 先读取微信 `bizProfile` 缓存中的公众号文章元数据。缓存命中时直接使用
标题、URL、发布时间和摘要，不再逐屏扫描微信历史列表。

- 不抓正文：缓存命中后直接回传，通常几秒内完成。
- 抓正文：公开文章 URL 可用时直接解析；被微信环境校验拦截时，仅对正文缺失的文章进入 UI。
- 缓存未命中：通过微信 UI 精确打开一次公众号主页，再重新读取缓存。
- 请求数量超过缓存数量：返回可用的缓存文章，并将任务标记为不完整，不伪造缺失数据。

缓存工具路径由 `WECHAT_CACHE_CLI_PATH` 配置，默认使用同机
`wechat-article-fetcher` 的 `wechat-fetch.ps1`。

## 消息协议

协议以 `server/docs/websocket-protocol.md` 为唯一事实来源。信封：

```json
{
  "version": "1.0",
  "id": "uuid",
  "type": "task.assigned",
  "timestamp": "2026-11-07T00:00:00Z",
  "request_id": null,
  "agent_id": "agent-uuid",
  "data": {}
}
```

Agent 上行：

- `agent.hello`
- `agent.heartbeat`
- `task.ack`
- `task.progress`
- `task.result`
- `task.complete`
- `wechat.login.qr`
- `wechat.login.status`
- `agent.log`
- `message.ack`

平台下行：

- `server.hello`
- `agent.configure`
- `task.assigned`
- `task.cancel`
- `wechat.login.request`
- `message.ack`
- `protocol.error`

重要业务消息会等待平台 `message.ack`；断线重连后会重发未确认消息。已完成任务写入
`task-state.json`，服务端同时使用规范化 URL 唯一索引兜底去重。

## 采集流程

1. 启动或激活微信，检查登录态。
2. 搜索完整公众号名称。
3. 只打开精确匹配的公众号历史列表。
4. 逐屏提取标题、日期、摘要和可见链接。
5. 必要时打开文章页解析真实 URL；正文开关打开时继续抓取正文。
6. 达到数量、到达底部、连续无新增或达到滚动上限后停止。
7. 分批回传文章并发送最终 `task.complete`。

单篇正文失败不会丢弃已经拿到的列表元数据；结果中会带 `detail_error` 和
`content_missing` 状态。`complete=false` 表示列表可能未完整读取。

## 扫码登录

管理端发起登录后，Agent：

1. 激活微信窗口。
2. 抓取当前桌面画面。
3. 通过 `wechat.login.qr` 上传图像。
4. 轮询视觉登录态；确认登录后发送 `wechat.login.status=confirmed`。

登录画面是敏感数据。管理端仅允许 operator/admin 获取，viewer 会被服务端脱敏。

## 日志与截图

- Agent 日志通过 `agent.log` 上报，服务端持久化并按保留策略清理。
- 截图通过带 Agent Bearer Token 的 `POST /api/v1/agents/{id}/screen` 上传。
- 管理端只向有权限的角色返回截图。

## 测试

```powershell
uv run pytest tests -q
```

平台协议全链路可在仓库根目录执行：

```powershell
uv run --project agent python scripts\e2e_mock_agent.py
```
