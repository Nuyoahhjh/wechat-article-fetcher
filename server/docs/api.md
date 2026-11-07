# 拾刊 REST API

Base URL：`http://127.0.0.1:8080/api/v1`

- 用户接口使用 `Authorization: Bearer <jwt>`
- Agent 注册使用 `X-Agent-Registration-Key`
- Agent 截图上传使用 Agent Bearer Token
- 时间使用 RFC3339
- 分页参数：`page`、`page_size`，默认 20，最大 100

成功：

```json
{ "data": {} }
```

分页：

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 123
  }
}
```

错误：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "human readable message"
  }
}
```

## 权限

| 角色 | 权限 |
| --- | --- |
| viewer | Dashboard、Agent、任务、文章、扫码状态只读 |
| operator | viewer + Agent 配置、扫码、创建/取消任务、导出、设置 |
| admin | operator + 用户管理、Agent 禁用、审计日志 |

## 系统

### GET /healthz

无需认证。数据库可用时返回 `200`。

## 认证

### POST /auth/login

```json
{
  "username": "admin",
  "password": "your-password"
}
```

返回前端所需结构：

```json
{
  "data": {
    "token": "<jwt>",
    "user": {
      "id": "uuid",
      "name": "Administrator",
      "email": "admin",
      "role": "admin"
    }
  }
}
```

### GET /auth/me

返回当前用户。前端用户字段使用 `name/email`。

### POST /auth/password

```json
{
  "current_password": "old-password",
  "new_password": "new-password-at-least-8"
}
```

## 用户

### GET /users

admin。分页返回用户。

### POST /users

```json
{
  "username": "operator01",
  "password": "a-strong-password",
  "display_name": "运营一号",
  "role": "operator"
}
```

### GET /users/{id}

### PATCH /users/{id}

可选字段：`display_name`、`role`、`status`、`password`。

### DELETE /users/{id}

成功返回 `204`，不能删除当前登录用户。

## Agent

### POST /agents/register

请求头：

```text
X-Agent-Registration-Key: <AGENT_REGISTRATION_KEY>
```

```json
{
  "name": "Windows Agent 01",
  "machine_id": "DESKTOP-01",
  "hostname": "DESKTOP-01",
  "os": "Windows 11",
  "version": "0.1.0",
  "capabilities": {
    "wechat_articles": true,
    "screenshot": true
  }
}
```

响应 `201` 包含 `agent` 和仅返回一次的 `agent_token`。相同 `machine_id`
再次注册会复用 Agent ID 并轮换 Token。

### GET /agents

查询参数：`status`、`search`、`page`、`page_size`。

返回真实 `active_tasks`、心跳、微信登录态和配置版本，不返回 Token 哈希。

### GET /agents/{id}

### PATCH /agents/{id}/config

operator/admin。

```json
{
  "config": {
    "ui": {
      "startup_timeout_seconds": 45
    }
  },
  "max_concurrency": 1
}
```

配置写入后，在线 Agent 会收到 `agent.configure`。

### DELETE /agents/{id}

admin。标记 Agent 为 disabled 并断开连接，保留历史数据。

### GET /agents/{id}/screen

operator/admin。返回最近一次持久化截图。

### POST /agents/{id}/screen

Agent Token 鉴权，不使用用户 JWT：

```text
Authorization: Bearer <agent_token>
```

```json
{
  "image_data": "iVBORw0KGgo...",
  "mime_type": "image/png",
  "width": 1920,
  "height": 1080,
  "captured_at": "2026-11-07T04:00:00Z"
}
```

单次最大 8 MiB。每个 Agent 保留最近 20 张，并按
`SCREENSHOT_RETENTION_DAYS` 清理。

### GET /agents/{id}/logs

operator/admin。返回最近 100 条真实 Agent 日志。

## 微信登录

### POST /agents/{id}/wechat/login

operator/admin。要求 Agent 在线。创建扫码会话并发送
`wechat.login.request`，响应 `202`。

### GET /agents/{id}/wechat/login/latest

读取最近一次会话，不会重复创建。viewer 的二维码内容会被服务端清空。

### GET /wechat/login-sessions/{session_id}

按会话 ID 查询状态。

允许状态：`pending`、`qr_ready`、`scanned`、`confirmed`、`expired`、
`failed`、`canceled`、`superseded`。

## 任务

### POST /tasks

operator/admin。

```json
{
  "type": "fetch_articles",
  "priority": 10,
  "agent_id": null,
  "payload": {
    "account_name": "目标公众号",
    "max_articles": 20,
    "since": "2026-10-01T00:00:00Z",
    "until": "2026-11-07T00:00:00Z",
    "include_content": true
  }
}
```

`priority` 为 1-1000，数字越小越优先。`agent_id` 可省略，交给调度器分配。

### GET /tasks

查询参数：`status`、`agent_id`、`type`、`search`、`page`、`page_size`。

### GET /tasks/{id}

终态字段：

- `error_code`：稳定失败分类
- `last_error`：用户可读错误
- `complete`：Agent 是否确认列表完整读取

### GET /tasks/{id}/events

返回按时间正序排列的事件流。`limit` 为 1-500，默认 100。

### POST /tasks/{id}/cancel

```json
{ "reason": "运营手工取消" }
```

非终态任务变为 `canceled`，在线 Agent 收到 `task.cancel`。

## 文章

### GET /articles

查询参数：

- `task_id`
- `agent_id`
- `account_name`
- `status`
- `search`
- `from` / `until`
- `page` / `page_size`

列表不返回正文，避免大响应。

### GET /articles/{id}

返回完整 `content_html`、`content_text`、质量状态和来源。

### GET /articles/accounts

返回文章库中全部去重后的公众号名称，用于筛选下拉框。

文章状态：

- `complete`
- `metadata_only`
- `content_missing`
- `failed`

### GET /articles/export

operator/admin。同步导出当前筛选结果，最多 10000 篇：

```text
GET /articles/export?format=json
GET /articles/export?format=csv&account_name=目标公众号
GET /articles/export?format=markdown&status=complete
```

响应为文件下载，`X-Export-Count` 为数量，并写入操作审计。

## 设置

### GET /settings

返回当前用户显示名称和持久化偏好。

### PATCH /settings

```json
{
  "display_name": "运营一号",
  "settings": {
    "notify_task_failed": true,
    "auto_refresh": true,
    "refresh_interval": 15,
    "timezone": "Asia/Shanghai"
  }
}
```

## 审计

### GET /audit-logs

admin。分页返回登录、任务、Agent、用户、设置和导出操作。

## Dashboard

### GET /dashboard

```json
{
  "data": {
    "users": 3,
    "agents": 2,
    "online_agents": 1,
    "tasks": 10,
    "active_tasks": 2,
    "articles": 100,
    "today_articles": 12,
    "task_success_rate": 87.5,
    "succeeded_tasks": 7,
    "failed_tasks": 1,
    "trends": [
      {
        "date": "2026-11-07",
        "tasks": 2,
        "articles": 12
      }
    ],
    "database": "ok"
  }
}
```
