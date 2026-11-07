# Agent WebSocket 协议

## 1. 连接

```text
GET /api/v1/agent/ws/{agent_id}
Authorization: Bearer <agent_token>
```

要求：

- 必须使用注册接口返回的 Agent token
- token 丢失后只能使用注册密钥重新注册并轮换 token
- 连接建立后第一条业务消息必须是 `agent.hello`
- 单条消息最大 8 MiB
- 服务端使用 WebSocket Ping/Pong 检测死连接
- Agent 应每 15 秒发送 `agent.heartbeat`

浏览器 `Origin` 不为空时，必须命中 `CORS_ALLOWED_ORIGINS`。Python Agent 通常不发送 `Origin`。

## 2. Envelope

所有消息使用同一层信封：

```json
{
  "version": "1.0",
  "id": "8bd77eb3-17da-4dff-9720-1a8531aae120",
  "type": "task.assigned",
  "timestamp": "2026-11-05T14:30:00Z",
  "request_id": "",
  "agent_id": "e53f42e0-741c-4a95-a5b4-f660917c7155",
  "data": {},
  "error": null
}
```

| 字段 | 必填 | 说明 |
|---|---|---|
| `version` | 是 | 当前固定 `1.0` |
| `id` | 是 | 每条消息唯一 UUID |
| `type` | 是 | 消息类型 |
| `timestamp` | 是 | RFC3339，UTC 推荐 |
| `request_id` | 否 | 关联的上游消息 ID |
| `agent_id` | Agent 发送时建议 | 必须与 URL 中的 Agent ID 相同 |
| `data` | 视类型 | 业务 payload |
| `error` | 平台发送时可能 | 结构化错误 |

平台对除 `agent.heartbeat`、`agent.log` 外的 Agent 消息回复：

```json
{
  "version": "1.0",
  "id": "2c99145a-b53d-43e1-af1b-0cdbf7433312",
  "type": "message.ack",
  "timestamp": "2026-11-05T14:30:01Z",
  "data": {
    "reply_to": "8bd77eb3-17da-4dff-9720-1a8531aae120"
  }
}
```

`message.ack` 只表示平台已接收并处理，不代表任务完成。

## 3. 消息类型

### Agent -> Platform

| type | 作用 | 平台响应 |
|---|---|---|
| `agent.hello` | 连接初始化、能力与微信状态 | `server.hello` + `message.ack` |
| `agent.heartbeat` | 保活、负载、微信状态 | 无业务响应 |
| `task.ack` | 确认开始任务 | `message.ack` |
| `task.progress` | 上报阶段进度 | `message.ack` |
| `task.result` | 上传一批文章 | `message.ack` |
| `task.complete` | 告知任务成功或失败 | `message.ack` |
| `wechat.login.qr` | 上传登录二维码 | `message.ack` |
| `wechat.login.status` | 扫码/确认/过期状态 | `message.ack` |
| `agent.log` | 上报运行日志 | 无业务响应 |
| `message.ack` | 确认平台下行消息 | 无 |

### Platform -> Agent

| type | 作用 |
|---|---|
| `server.hello` | 下发当前配置、版本和心跳建议 |
| `agent.configure` | 配置变更 |
| `task.assigned` | 下发或重放任务 |
| `task.cancel` | 取消当前任务 |
| `wechat.login.request` | 请求 Agent 生成并上传二维码 |
| `message.ack` | 平台接收确认 |
| `protocol.error` | 协议、鉴权后业务或消息处理错误 |

## 4. Payload

### 4.1 agent.hello

```json
{
  "version": "0.1.0",
  "machine_id": "WIN-MACHINE-UUID",
  "hostname": "DESKTOP-01",
  "os": "Windows 11",
  "capabilities": {
    "windows_mcp": true,
    "vision_model": "qwen3.8-27b",
    "screenshot": true
  },
  "wechat": {
    "logged_in": true,
    "nickname": "公众号运营"
  }
}
```

### 4.2 server.hello

```json
{
  "agent_id": "e53f42e0-741c-4a95-a5b4-f660917c7155",
  "server_time": "2026-11-05T14:30:00Z",
  "heartbeat_seconds": 15,
  "config": {
    "ui": {
      "startup_timeout_seconds": 30
    }
  },
  "config_version": 3
}
```

### 4.3 agent.heartbeat

```json
{
  "running_tasks": 1,
  "cpu_percent": 23.5,
  "memory_percent": 51.2,
  "wechat": {
    "logged_in": true,
    "nickname": "公众号运营"
  },
  "metadata": {
    "wechat_process_running": true
  }
}
```

平台会在收到心跳时延长该 Agent 当前任务的租约。

### 4.4 agent.configure

```json
{
  "config": {
    "ui": {
      "startup_timeout_seconds": 45
    }
  },
  "config_version": 4,
  "max_concurrency": 1
}
```

Agent 应持久化配置和版本。重连后以 `server.hello.config_version` 为准。

### 4.5 task.assigned

```json
{
  "task": {
    "id": "557e2ce5-ecb7-48f5-937a-b76f0048942d",
    "type": "fetch_articles",
    "priority": 10,
    "attempt": 1,
    "max_attempts": 3,
    "created_at": "2026-11-05T14:31:00Z",
    "requested_agent_id": null
  },
  "payload": {
    "account_name": "目标公众号",
    "max_articles": 20,
    "since": "2026-10-01T00:00:00Z",
    "until": "2026-11-05T00:00:00Z",
    "include_content": true
  },
  "lease_expires_at": "2026-11-05T14:32:30Z"
}
```

Agent 收到后应：

1. 本地按 `task.id` 幂等保存
2. 尽快发送 `task.ack`
3. 执行期间持续发送 `task.progress`
4. 分批发送 `task.result`
5. 最后发送 `task.complete`

### 4.6 task.ack

```json
{
  "task_id": "557e2ce5-ecb7-48f5-937a-b76f0048942d"
}
```

### 4.7 task.progress

```json
{
  "task_id": "557e2ce5-ecb7-48f5-937a-b76f0048942d",
  "stage": "collecting_articles",
  "percent": 55,
  "message": "已读取 11/20 篇文章标题",
  "current": 11,
  "total": 20
}
```

`percent` 会被限制在 0 到 100。

### 4.8 task.result

单次最多 500 篇。推荐每 10 到 50 篇发送一批。

```json
{
  "task_id": "557e2ce5-ecb7-48f5-937a-b76f0048942d",
  "articles": [
    {
      "external_id": "agent-local-id",
      "account_name": "目标公众号",
      "account_id": "gh_xxx",
      "title": "文章标题",
      "url": "https://mp.weixin.qq.com/s/xxxx",
      "author": "作者",
      "digest": "摘要",
      "cover_url": "https://mmbiz.qpic.cn/...",
      "content_html": "<article>...</article>",
      "content_text": "正文纯文本",
      "published_at": "2026-11-04T08:00:00Z",
      "fetched_at": "2026-11-05T14:32:00Z",
      "raw": {
        "llm_confidence": 0.96
      }
    }
  ]
}
```

入库规则：

- `title` 和 `url` 必填
- URL 必须是 `http/https`
- `account_name` 为空时使用任务 payload 中的目标公众号
- URL 已存在时静默跳过
- 其他非法文章会跳过，不会使整批失败

### 4.9 task.complete

成功：

```json
{
  "task_id": "557e2ce5-ecb7-48f5-937a-b76f0048942d",
  "success": true,
  "result_count": 20,
  "complete": true
}
```

失败：

```json
{
  "task_id": "557e2ce5-ecb7-48f5-937a-b76f0048942d",
  "success": false,
  "result_count": 3,
  "error": "微信搜索页连续 3 次无法进入目标公众号",
  "error_code": "wechat_ui_unreachable",
  "retryable": true,
  "complete": false
}
```

平台以数据库实际入库数覆盖 `result_count`。重复 complete 是幂等的。
`complete=false` 表示 Agent 未确认列表完整读取，任务仍可成功但管理端会提示。

### 4.10 wechat.login.request

```json
{
  "session_id": "fc553431-93e6-40b5-a570-31c62c561dc6",
  "expires_at": "2026-11-05T14:40:00Z"
}
```

### 4.11 wechat.login.qr

`image_base64` 必须是标准 base64，不带 `data:image/png;base64,` 前缀。

```json
{
  "session_id": "fc553431-93e6-40b5-a570-31c62c561dc6",
  "image_base64": "iVBORw0KGgo...",
  "mime_type": "image/png",
  "expires_at": "2026-11-05T14:40:00Z"
}
```

限制 2 MiB。超过最大有效期时平台会截断到当前时间后 10 分钟。

### 4.12 wechat.login.status

```json
{
  "session_id": "fc553431-93e6-40b5-a570-31c62c561dc6",
  "status": "scanned",
  "wechat_account": "公众号运营",
  "message": "用户已扫码，等待手机确认"
}
```

允许状态：`scanned / confirmed / expired / failed / canceled`。

`confirmed` 会同时把 Agent 标记为微信已登录。

### 4.13 protocol.error

```json
{
  "version": "1.0",
  "id": "a28d508a-c57d-48a2-b4f7-d9b091ff4b93",
  "type": "protocol.error",
  "timestamp": "2026-11-05T14:30:02Z",
  "request_id": "8bd77eb3-17da-4dff-9720-1a8531aae120",
  "error": {
    "code": "message_failed",
    "message": "task is not assigned to this agent",
    "retryable": false
  }
}
```

常见 code：

- `invalid_json`
- `invalid_envelope`
- `connection_not_initialized`
- `message_failed`

## 5. 时序

### 连接初始化

```text
Agent                          Platform
  | --- WS + Bearer token -------> |
  | <-------------- 101 Switching -|
  | --- agent.hello --------------> |
  | <-------------- server.hello ---|
  | <-------------- message.ack ----|
  | --- agent.heartbeat ----------> |
```

### 任务执行

```text
Platform                       Agent
  | --- task.assigned -----------> |
  | <-------------- task.ack ------|
  | <-------------- task.progress -|
  | <-------------- task.result ---|
  | <-------------- task.result ---|
  | <-------------- task.complete -|
  | --- message.ack -------------> |
```

### 扫码登录

```text
Vue        Platform        Agent
 | POST login |               |
 |----------->|               |
 |            |-- request --->|
 |<-- session-|               |
 | GET latest |               |
 |----------->|               |
 |            |<------ qr -----|
 |<-- base64 -|               |
 |            |<-- scanned ----|
 |            |<-- confirmed --|
 |<-- confirmed               |
```

## 6. 重连与投递语义

- 同一 Agent 只保留一个活动连接；新连接会关闭旧连接
- 旧连接关闭不能把新连接状态改为 offline
- 重连后平台重发该 Agent 当前 `assigned/running` 任务
- `task.assigned` 是至少一次投递，Agent 必须本地幂等
- 租约在进度、心跳、结果处理时续期
- 租约过期后任务可能重新分配给其他 Agent
- 文章 URL 唯一键防止跨重试重复入库
