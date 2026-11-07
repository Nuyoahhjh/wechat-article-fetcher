# 拾刊管理端

Vue 3 + TypeScript + Element Plus 管理端，只使用真实 REST API，不包含 Mock 数据。

## 页面

- 登录
- Dashboard
- Agent 列表与详情
- 微信登录、截图和真实操作日志
- 任务创建、列表、详情、事件流和取消
- 文章库、文章详情、筛选和 JSON/CSV/Markdown 导出
- 用户与角色管理（admin）
- 操作审计（admin）
- 个人资料、密码、刷新偏好和服务状态

## 开发

```powershell
npm install
npm run dev
```

默认 Vite 端口 `5173`，代理到 `http://127.0.0.1:8080`。

可通过环境变量覆盖开发代理：

```powershell
$env:VITE_DEV_PORT='5173'
$env:VITE_DEV_API_TARGET='http://127.0.0.1:18081'
$env:VITE_DEV_WS_TARGET='ws://127.0.0.1:18081'
npm run dev
```

## 检查

```powershell
npm run typecheck
npm run build
```

## 数据刷新

管理端使用 REST 短轮询。自动刷新开关和间隔来自用户设置，并持久化到平台。
Agent 详情页单独以较短间隔刷新截图和登录状态。
