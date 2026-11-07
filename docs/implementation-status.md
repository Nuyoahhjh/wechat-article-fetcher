# 拾刊 V3 实现状态

更新日期：2026-11-07

本文件用于区分“已落地”“受外部环境约束”和“后续版本范围”，避免管理端展示未实现能力。

## P0：真实端到端链路

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| Agent/Platform WebSocket 信封 | 已实现 | 统一为 `version/id/type/timestamp/request_id/agent_id/data` |
| Agent 接收 `task.assigned` | 已实现 | 支持服务端任务与 payload 映射 |
| ACK、进度、结果、完成 | 已实现 | Agent 发送 `task.ack/progress/result/complete` |
| 文章入库模型转换 | 已实现 | 支持正文、元数据、状态、诊断和错误 |
| 管理端创建任务 | 已实现 | 使用 `POST /tasks` 和服务端 snake_case payload |
| 任务/文章/Dashboard 适配 | 已实现 | 页面只使用真实 REST 数据 |
| 分页总数 | 已实现 | `meta.total` 来自数据库 COUNT |
| 实时状态 | 已实现 | 使用短轮询；不再展示假 WebSocket 连接状态 |
| P0 自动验收 | 已实现 | `scripts/e2e_mock_agent.py` |
| `bizProfile` 缓存优先 | 已实现 | 先读本地公众号缓存；仅在缓存未命中或正文缺失时调用 UI |

## P1：单节点 MVP

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| WebSocket 心跳与重连 | 已实现 | 指数退避；重要消息等待 ACK 并重发 |
| 任务幂等 | 已实现 | Agent 本地持久化已完成任务，服务端按 URL 去重 |
| 微信登录请求 | 已实现 | Agent 捕获桌面登录画面并上传 |
| 扫码状态 | 已实现 | 支持 pending / qr_ready / confirmed / expired / failed |
| 截图 | 已实现 | Agent Token 鉴权上传，SQLite 持久化，支持保留策略 |
| Agent 日志 | 已实现 | `agent.log` 持久化，管理端展示真实日志 |
| 错误码 | 已实现 | 任务记录 `error_code`、重试次数与可读错误 |
| 任务时间线 | 已实现 | 创建、派发、ACK、进度、入库、终态均可查询 |
| 登录页清理 | 已实现 | 无 Mock 文案、无预填管理员密码 |
| 设置页 | 已实现 | 资料、密码、刷新策略、偏好设置调用真实 API |

真实公众号连续采集仍需在目标 Windows 机器上使用已登录微信桌面端验证，属于外部桌面环境验收。

## P2：小团队运营

| 项目 | 状态 | 说明 |
| --- | --- | --- |
| 用户管理界面 | 已实现 | admin 可创建、修改、停用和删除用户 |
| 多 Agent 调度 | 已实现 | 按在线、微信登录、心跳、并发和优先级调度 |
| 失败原因分类 | 已实现 | 保存 `error_code`；详情页显示错误码与消息 |
| 日志/截图保留 | 已实现 | 服务端周期清理，保留天数可配置 |
| 文章导出 | 已实现 | JSON / CSV / Markdown |
| 文章详情 | 已实现 | 正文、元数据、来源、原始链接、质量状态 |
| 操作审计 | 已实现 | 登录、任务、Agent、用户、设置和导出记录 |
| Docker 部署 | 已实现 | Server/Frontend Compose、健康检查、反向代理 |
| 备份与升级 | 已写入运维文档 | SQLite 文件备份与升级流程 |

## P3：规模化扩展

以下能力属于后续版本，当前界面不提供假入口：

- 公众号档案和别名管理
- 定时采集与增量监控
- 单篇 URL 导入
- 批量账号任务
- Webhook / OpenAPI / SDK
- PostgreSQL / Redis / 对象存储
- Prometheus 指标
- 更细粒度任务模板和权限模型
- 内容质量与置信度评分体系

## 验证命令

```powershell
cd server
go test ./...

cd ..\agent
uv run pytest tests -q

cd ..\frontend
npm run typecheck

cd ..
uv run --project agent python scripts\e2e_mock_agent.py
```
