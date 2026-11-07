# 拾刊 V3 技术约束

产品范围和阶段以 [product-plan.md](product-plan.md) 为唯一事实来源。

## 运行约束

- 平台：Go 1.23+、SQLite，单实例控制台
- 管理端：Vue 3 + TypeScript + Element Plus
- Agent：Python 3.11+、windows-mcp、OpenAI 兼容视觉模型
- Agent 必须在 Windows 交互式桌面运行，并保持微信已登录
- 每台 Windows 机器对应一个 Agent 和一个微信登录态
- 同一 Agent 的 UI 任务串行执行

## 数据与协议

- 用户/Agent API 使用 JWT
- Agent 注册使用注册密钥，后续使用 Agent Token
- Agent WebSocket 协议以 `server/docs/websocket-protocol.md` 为准
- 文章以规范化 URL 为幂等键
- Agent Token 仅保存哈希
- 二维码、截图和日志按权限访问并受保留策略控制

## 模型配置

- Endpoint 和模型名由 `agent/config.yaml` 配置
- API Key 只能通过 `WECHAT_AGENT_LLM_API_KEY` 注入
- Key 不得提交到仓库、写入示例配置或输出到日志

## 合规边界

不绕过微信登录、验证码、安全提醒、付费墙、隐私限制或已删除内容。
历史列表完整性取决于微信桌面端实际暴露的数据。
