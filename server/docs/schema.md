# 数据库 Schema

权威 SQL 为 `internal/store/migration.sql`。本文说明表职责与关键约束。

## users

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | TEXT PK | UUID |
| `username` | TEXT UNIQUE | 登录名 |
| `password_hash` | TEXT | bcrypt |
| `display_name` | TEXT | 展示名 |
| `role` | TEXT | `admin / operator / viewer` |
| `status` | TEXT | `active / disabled` |
| `last_login_at` | DATETIME | 最近登录 |
| `created_at / updated_at` | DATETIME | 审计时间 |

## agents

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | TEXT PK | UUID |
| `name` | TEXT | 平台展示名 |
| `machine_id` | TEXT UNIQUE | Agent 稳定机器标识 |
| `token_hash` | TEXT | SHA-256 Agent token |
| `status` | TEXT | `offline / online / busy / disabled / error` |
| `version / os / hostname` | TEXT | Agent 运行信息 |
| `capabilities_json` | TEXT | 能力声明 |
| `config_json` | TEXT | 平台下发配置 |
| `config_version` | INTEGER | 单调递增配置版本 |
| `max_concurrency` | INTEGER | 最大并行任务数 |
| `wechat_logged_in` | INTEGER | 0/1 |
| `wechat_nickname` | TEXT | 当前微信昵称 |
| `last_heartbeat_at` | DATETIME | 应用心跳 |
| `connected_at` | DATETIME | WS 最近连接时间 |
| `created_at / updated_at` | DATETIME | 审计时间 |

## tasks

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | TEXT PK | UUID |
| `type` | TEXT | 当前为 `fetch_articles` |
| `status` | TEXT | `pending / assigned / running / succeeded / failed / canceled` |
| `priority` | INTEGER | 越小越优先 |
| `requested_agent_id` | TEXT FK nullable | 人工指定偏好 |
| `agent_id` | TEXT FK nullable | 当前租约持有者 |
| `payload_json` | TEXT | 任务参数 |
| `progress_json` | TEXT | 最新进度 |
| `result_count` | INTEGER | 本任务实际入库文章数 |
| `last_error` | TEXT | 最近错误 |
| `error_code` | TEXT | 稳定失败分类 |
| `complete` | INTEGER | Agent 是否确认列表完整读取 |
| `attempts` | INTEGER | 已投递次数 |
| `max_attempts` | INTEGER | 最大投递次数 |
| `lease_until` | DATETIME | 执行租约 |
| `created_by` | TEXT FK nullable | 创建用户 |
| `started_at / finished_at` | DATETIME | 执行时间 |
| `created_at / updated_at` | DATETIME | 审计时间 |

## task_events

任务事件流，用于运营排障和前端时间线。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | INTEGER PK AUTOINCREMENT | 事件序号 |
| `task_id` | TEXT FK | 任务 |
| `agent_id` | TEXT FK nullable | 相关 Agent |
| `event_type` | TEXT | `created / assigned / running / progress / ...` |
| `message` | TEXT | 可读说明 |
| `data_json` | TEXT | 结构化附加数据 |
| `created_at` | DATETIME | 发生时间 |

## articles

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | TEXT PK | UUID |
| `task_id` | TEXT FK | 来源任务 |
| `agent_id` | TEXT FK | 来源 Agent |
| `account_name` | TEXT | 公众号名称 |
| `account_id` | TEXT | 可选公众号 ID |
| `title` | TEXT | 标题 |
| `url` | TEXT UNIQUE | 规范化文章 URL，幂等键 |
| `author / digest / cover_url` | TEXT | 元数据 |
| `content_html / content_text` | TEXT | 正文 |
| `published_at` | DATETIME | 发布时间 |
| `fetched_at` | DATETIME | Agent 获取时间 |
| `raw_json` | TEXT | 原始结构化结果 |
| `status` | TEXT | `complete / metadata_only / content_missing / failed` |
| `complete` | INTEGER | 列表结果是否完整 |
| `source` | TEXT | 采集来源，默认 `wechat_desktop` |
| `quality_json` | TEXT | 诊断、详情错误与质量信息 |
| `created_at` | DATETIME | 入库时间 |

## wechat_login_sessions

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | TEXT PK | 会话 UUID |
| `agent_id` | TEXT FK | 目标 Agent |
| `status` | TEXT | `pending / qr_ready / scanned / confirmed / expired / failed / canceled / superseded` |
| `qr_image` | TEXT | 纯 base64 图片，不含 data URL 前缀 |
| `qr_mime_type` | TEXT | 通常为 `image/png` |
| `wechat_account` | TEXT | 确认后的微信账号/昵称 |
| `failure_reason` | TEXT | 失败原因 |
| `expires_at` | DATETIME | 二维码过期时间 |
| `created_at / updated_at` | DATETIME | 审计时间 |

## agent_logs

Agent 上报的结构化运行日志。`fields_json` 保存上下文；敏感字段键会被脱敏。

## agent_screenshots

最近屏幕截图。保存纯 base64、MIME、尺寸和采集时间；每个 Agent 保留最近 20 张。

## user_preferences

按用户保存刷新、通知偏好和时区等设置。`settings_json` 为结构化 JSON。

## audit_logs

登录、任务、Agent、用户、设置和导出等操作审计。`actor_user_id` 可为空以保留已删除用户的历史记录。
