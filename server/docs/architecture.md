# 平台端架构设计

## 1. 设计边界

平台端负责：

- 管理多个 Windows Agent 实例
- 维护 Agent 在线状态、配置和负载
- 创建任务、调度任务、跟踪状态、接收结果
- 保存微信公众号文章
- 管理平台用户与权限
- 接收 Agent 上传的微信登录二维码并提供前端轮询接口

平台端不负责模拟微信 UI。点击、截图、LLM 判断和微信进程控制仍在 Windows Agent 内完成。

当前实现是一个单节点 MVP：

- API、WebSocket Hub、Scheduler 在同一进程
- SQLite 保存业务数据
- 任务采用数据库租约和至少一次投递
- 前端通过 REST API 和短轮询获取业务状态

## 2. 组件视图

```text
                         +----------------------+
                         |      Vue Admin       |
                         +----------+-----------+
                                    |
                              REST / JWT
                                    |
+-----------------------------------v-----------------------------------+
|                         Golang Platform                               |
|                                                                       |
|  HTTP API  --------+                                                  |
|                    |                                                  |
|  Auth / RBAC       |                                                  |
|  Agent Registry    |                                                  |
|  Task Service      +------ SQLite                                      |
|  Article Service   |                                                  |
|  WeChat Login API  |                                                  |
|                    |                                                  |
|  Scheduler --------+                                                  |
|                                                                       |
|  WebSocket Hub <---- agent.hello / heartbeat / task / qr / logs       |
+-----------------------------------^-----------------------------------+
                                    |
                         WebSocket + Bearer token
                                    |
              +---------------------+---------------------+
              |                     |                     |
       Windows Agent 01      Windows Agent 02      Windows Agent N
       WeChat Account A       WeChat Account B       WeChat Account N
       windows-mcp + LLM      windows-mcp + LLM      windows-mcp + LLM
```

## 3. Golang 包结构

```text
server/
  cmd/server/
    main.go                    配置、HTTP server、信号退出
  internal/app/
    app.go                     依赖装配；解决 Hub 与服务层的构造顺序
  internal/config/
    config.go                  环境变量与约束
  internal/domain/
    constants.go               角色、Agent、任务、登录状态
    models.go                  领域实体
  internal/protocol/
    protocol.go                WS envelope 与 payload
  internal/auth/
    auth.go                    JWT、bcrypt、Agent token hash
  internal/store/
    store.go                   SQLite 打开、迁移、统一错误
    users.go                   users
    agents.go                  agents
    tasks.go                   tasks、task_events、租约
    articles.go                articles
    login_sessions.go          wechat_login_sessions
    stats.go                   Dashboard 聚合
    migration.sql              完整 Schema
  internal/service/
    auth_service.go
    user_service.go
    agent_service.go
    task_service.go
    article_service.go
    wechat_service.go
    notifier.go                服务层对 WS 的窄接口
  internal/ws/
    hub.go                     连接注册、单写协程、Ping/Pong、重连替换
    dispatcher.go              Agent 消息路由
  internal/scheduler/
    scheduler.go               租约回收、离线清理、任务派发
  internal/httpapi/
    router.go
    middleware.go
    *_handler.go
```

分层规则：

- `httpapi` 只做参数绑定、状态码和权限校验
- `service` 实现业务状态机，不依赖 `http`
- `store` 只负责 SQL 与事务
- `ws` 负责连接和消息传输，通过 `AgentNotifier` 向业务层暴露能力
- `protocol` 和 `domain` 不依赖上层包

## 4. 核心实体

### agents

一个 Windows 机器对应一个 Agent，一个 Agent 当前登录一个微信账号。

关键字段：

- `machine_id`：稳定机器标识，用于重复注册时复用同一条 Agent 记录
- `token_hash`：Agent Bearer token 的 SHA-256，不保存明文 token
- `status`：`offline / online / busy / disabled / error`
- `config_json + config_version`：平台下发配置及版本
- `max_concurrency`：该 Agent 可同时执行的任务数
- `last_heartbeat_at`：应用层心跳时间
- `wechat_logged_in / wechat_nickname`：微信登录状态

### tasks

任务同时保存“指定 Agent 偏好”和“当前执行 Agent”：

- `requested_agent_id`：创建任务时的人工指定，租约过期后仍然保留
- `agent_id`：当前租约持有者，任务回收后清空
- `attempts / max_attempts`：至少一次投递的重试边界
- `lease_until`：当前投递租约
- `progress_json`：前端可展示的进度快照
- `payload_json`：`fetch_articles` 参数

### articles

文章以规范化 URL 为幂等键：

- 同一 URL 重复上传不会产生重复文章
- `task_id` 和 `agent_id` 保留首次成功入库的来源
- `raw_json` 保留 Agent 原始结构化结果，便于后续补充解析

### users

角色分为：

- `admin`：用户管理、Agent 禁用、全部业务操作
- `operator`：配置 Agent、创建和取消任务、发起扫码
- `viewer`：只读

### wechat_login_sessions

扫码会话生命周期与 Agent 实际登录状态分离。前端轮询此表即可展示：

- `pending`：已请求 Agent 生成二维码
- `qr_ready`：二维码已上传
- `scanned`：用户已扫码但尚未确认
- `confirmed / expired / failed / canceled / superseded`

## 5. 任务状态机

```text
                 +---------------- cancel ----------------+
                 |                                        v
pending --assign--> assigned --task.ack--> running --complete--> succeeded
   ^                    |                      |                 |
   |                    |                      +-----complete----> failed
   |                    |                                        |
   +---- lease expired -+------------ retry <-------------------+
                                       attempt < max_attempts

任意非终态 --operator cancel--> canceled
租约过期且 attempts >= max_attempts --> failed
```

状态约束：

- 只有 `pending` 或租约过期的 `assigned/running` 可以被重新分配
- 分配时 `attempts + 1`
- Agent 发送 `task.ack` 后进入 `running`
- 进度、结果、完成消息必须来自当前 `agent_id`
- 重复 `task.result` 依靠 URL 唯一键去重
- 重复 `task.complete` 在终态下幂等返回

## 6. 调度算法

Scheduler 每 5 秒运行一次，同时接受“任务创建”和“Agent 重连”的即时触发。

单次调度顺序：

1. 清理心跳超时的 Agent，将 `online/busy` 改为 `offline`
2. 回收租约过期的任务
3. 将过期扫码会话标记为 `expired`
4. 查询可运行任务
5. 查询已登录微信、心跳有效、未禁用的 Agent
6. 按“当前活跃任务数 / max_concurrency”做最低负载选择
7. 数据库条件更新为 `assigned` 并设置租约
8. 通过 WebSocket 发送 `task.assigned`
9. 如果发送失败，立即释放任务并记录 `delivery_failed`

投递语义是“至少一次”。Agent 必须使用 `task_id` 保持本地幂等，平台使用 URL 唯一键保证结果幂等。

Agent 重连后：

1. 重新发送 `agent.hello`
2. 平台更新在线状态
3. 发送 `server.hello`
4. 查询该 Agent 当前 `assigned/running` 任务并重发
5. 触发一次全局调度

## 7. 数据库 Schema

完整 SQL 位于 `internal/store/migration.sql`。按职责分为：

- `users`
- `agents`
- `tasks`
- `task_events`
- `articles`
- `wechat_login_sessions`

关键索引：

- `tasks(status, priority, created_at)`：任务队列扫描
- `tasks(agent_id, status)`：重连任务恢复
- `tasks(status, lease_until)`：租约回收
- `articles(url)`：幂等入库
- `articles(account_name, published_at)`：公众号文章查询
- `wechat_login_sessions(agent_id, created_at)`：最新扫码会话

更详细的字段说明见 [schema.md](schema.md)。

## 8. REST API 与 WebSocket

- REST API 见 [api.md](api.md)
- Agent WebSocket 协议见 [websocket-protocol.md](websocket-protocol.md)

REST 使用统一信封：

```json
{
  "data": {}
}
```

错误统一为：

```json
{
  "error": {
    "code": "invalid_request",
    "message": "human readable message"
  }
}
```

## 9. 故障与恢复

| 故障 | 行为 |
|---|---|
| Agent 网络断开 | WS 连接移除；超过租约后任务回到 `pending` |
| Agent 进程崩溃但网络未断 | Ping/Pong 超时关闭连接；应用心跳超时后标记 offline |
| 任务消息丢失 | 租约过期后重新投递，最多 `max_attempts` 次 |
| Agent 重复上传文章 | `articles.url` 唯一键去重 |
| Agent 重复完成任务 | 终态幂等 |
| 平台重启 | offline 状态可由心跳清理；已分配任务由租约回收 |
| 二维码过期 | 会话转 `expired`，前端重新发起 |

## 10. 扩展路线

进入多实例或更大规模时：

1. 将 SQLite 替换为 PostgreSQL
2. 使用数据库行锁、`SKIP LOCKED` 或独立 leader 保证只有一个调度器
3. WebSocket 网关无状态化，使用 Redis Pub/Sub 或 NATS 路由 Agent 消息
4. QR 图片和大文章正文放对象存储，数据库只保存 key/摘要
5. 登录接口增加速率限制、刷新令牌和审计日志
6. 增加 Prometheus 指标：在线 Agent、队列深度、任务延迟、投递失败、文章入库量
