import { http } from './http'
import type {
  Agent,
  AgentConfig,
  AgentListQuery,
  AgentLog,
  Article,
  AuditLog,
  ArticleListQuery,
  ArticleStatus,
  CreateTaskPayload,
  DashboardStats,
  LoginPayload,
  LoginResult,
  ManagedUser,
  PageResult,
  QrCodePayload,
  ScreenSnapshot,
  SettingsResponse,
  Task,
  TaskEvent,
  TaskListQuery,
  TaskProgress,
  TaskStatus,
  UserRole,
  UserSettings,
} from '@/types'

type Envelope<T> = T | { data: T }

const unwrap = <T>(value: Envelope<T>): T => {
  if (value && typeof value === 'object' && 'data' in value) {
    return value.data as T
  }
  return value as T
}

function unwrapPage<T>(value: unknown): PageResult<T> {
  const raw = value as {
    data?: unknown
    meta?: { page?: number; page_size?: number; total?: number }
  }
  const items = Array.isArray(raw?.data) ? raw.data : []
  const meta = raw?.meta ?? {}
  return {
    items: items as T[],
    total: meta.total ?? items.length,
    page: meta.page ?? 1,
    pageSize: meta.page_size ?? 20,
  }
}

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {}
}

function stringValue(value: unknown, fallback = ''): string {
  return typeof value === 'string' ? value : fallback
}

function numberValue(value: unknown, fallback = 0): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function optionalString(value: unknown): string | undefined {
  return typeof value === 'string' && value ? value : undefined
}

function adaptAgent(rawValue: unknown): Agent {
  const raw = asRecord(rawValue)
  const capabilitiesValue = raw.capabilities
  const capabilities = Array.isArray(capabilitiesValue)
    ? capabilitiesValue.map(String)
    : Object.keys(asRecord(capabilitiesValue))
  const rawConfig = asRecord(raw.config)
  const config: AgentConfig = {
    heartbeatInterval: numberValue(rawConfig.heartbeat_seconds, 20),
    maxConcurrency: numberValue(raw.max_concurrency, 1),
    screenshotEnabled: rawConfig.screenshot_enabled !== false,
    autoReconnect: true,
    browserMode: 'desktop',
  }
  const machineId = stringValue(raw.machine_id)
  return {
    id: stringValue(raw.id),
    code: machineId.slice(-12).toUpperCase() || stringValue(raw.id).slice(-8).toUpperCase(),
    name: stringValue(raw.name, '未命名 Agent'),
    status: stringValue(raw.status, 'offline') as Agent['status'],
    hostname: stringValue(raw.hostname),
    version: stringValue(raw.version),
    os: stringValue(raw.os),
    activeTasks: numberValue(raw.active_tasks),
    lastHeartbeat: stringValue(raw.last_heartbeat_at, stringValue(raw.updated_at)),
    capabilities,
    metrics: {
      cpu: null,
      memory: null,
      disk: null,
      latency: null,
    },
    wechat: {
      nickname: stringValue(raw.wechat_nickname),
      accountId: machineId,
      loginStatus: raw.wechat_logged_in ? 'logged_in' : 'logged_out',
      lastSeenAt: stringValue(raw.last_heartbeat_at, stringValue(raw.updated_at)),
    },
    config,
    configVersion: numberValue(raw.config_version, 1),
    rawConfig,
  }
}

function adaptTask(rawValue: unknown): Task {
  const raw = asRecord(rawValue)
  const payload = asRecord(raw.payload)
  const progress = asRecord(raw.progress) as unknown as TaskProgress
  const agentId = stringValue(raw.agent_id)
  const agentName = stringValue(raw.agent_name)
  const requestedAgentId = optionalString(raw.requested_agent_id)
  const accountName = stringValue(payload.account_name, '未命名公众号')
  const maxArticles = numberValue(payload.max_articles, 20)
  return {
    id: stringValue(raw.id),
    name: `${accountName} · 文章采集`,
    type: 'fetch_articles',
    status: stringValue(raw.status, 'pending') as TaskStatus,
    priority: numberValue(raw.priority, 100),
    requestedAgentId,
    agentId,
    agentName: agentName || (agentId ? agentId.slice(-8).toUpperCase() : '自动调度'),
    accountName,
    maxArticles,
    since: optionalString(payload.since),
    until: optionalString(payload.until),
    includeContent: payload.include_content === true,
    progress: numberValue(progress.percent),
    stage: stringValue(progress.stage),
    progressMessage: stringValue(progress.message),
    collected: numberValue(progress.current, numberValue(raw.result_count)),
    total: numberValue(progress.total, maxArticles),
    attempts: numberValue(raw.attempts),
    maxAttempts: numberValue(raw.max_attempts, 3),
    lastError: optionalString(raw.last_error),
    errorCode: optionalString(raw.error_code),
    complete: raw.complete === true,
    createdAt: stringValue(raw.created_at),
    startedAt: optionalString(raw.started_at),
    finishedAt: optionalString(raw.finished_at),
    updatedAt: stringValue(raw.updated_at),
  }
}

function adaptTaskEvent(rawValue: unknown): TaskEvent {
  const raw = asRecord(rawValue)
  return {
    id: numberValue(raw.id),
    taskId: stringValue(raw.task_id),
    agentId: optionalString(raw.agent_id),
    type: stringValue(raw.type),
    message: stringValue(raw.message),
    data: asRecord(raw.data),
    createdAt: stringValue(raw.created_at),
  }
}

function adaptArticle(rawValue: unknown): Article {
  const raw = asRecord(rawValue)
  return {
    id: stringValue(raw.id),
    taskId: stringValue(raw.task_id),
    agentId: stringValue(raw.agent_id),
    title: stringValue(raw.title),
    accountName: stringValue(raw.account_name),
    accountId: optionalString(raw.account_id),
    author: stringValue(raw.author),
    digest: stringValue(raw.digest),
    contentHtml: stringValue(raw.content_html),
    contentText: stringValue(raw.content_text),
    coverUrl: stringValue(raw.cover_url),
    url: stringValue(raw.url),
    publishedAt: optionalString(raw.published_at),
    fetchedAt: stringValue(raw.fetched_at),
    collectedAt: stringValue(raw.created_at),
    status: stringValue(raw.status, 'metadata_only') as ArticleStatus,
    complete: raw.complete === true,
    source: stringValue(raw.source),
    quality: asRecord(raw.quality),
  }
}

function adaptQr(rawValue: unknown): QrCodePayload {
  const raw = asRecord(rawValue)
  const image = stringValue(raw.qr_image)
  const mimeType = stringValue(raw.qr_mime_type, 'image/png')
  return {
    id: stringValue(raw.id),
    agentId: stringValue(raw.agent_id),
    imageUrl: image ? `data:${mimeType};base64,${image}` : '',
    mimeType,
    expiresAt: stringValue(raw.expires_at),
    status: stringValue(raw.status, 'pending') as QrCodePayload['status'],
    message: optionalString(raw.failure_reason),
  }
}

function adaptScreen(rawValue: unknown): ScreenSnapshot {
  const raw = asRecord(rawValue)
  const image = stringValue(raw.image_data)
  const mimeType = stringValue(raw.mime_type, 'image/png')
  return {
    imageUrl: image ? `data:${mimeType};base64,${image}` : '',
    capturedAt: stringValue(raw.captured_at),
    width: numberValue(raw.width),
    height: numberValue(raw.height),
    online: Boolean(image),
    message: optionalString(raw.message),
  }
}

function adaptLog(rawValue: unknown): AgentLog {
  const raw = asRecord(rawValue)
  const level = stringValue(raw.level, 'info')
  return {
    id: String(raw.id ?? ''),
    agentId: stringValue(raw.agent_id),
    level: level === 'warn' ? 'warning' : (level as AgentLog['level']),
    message: stringValue(raw.message),
    createdAt: stringValue(raw.created_at),
    fields: asRecord(raw.fields),
  }
}

function adaptUser(rawValue: unknown): ManagedUser {
  const raw = asRecord(rawValue)
  return {
    id: stringValue(raw.id),
    username: stringValue(raw.username),
    displayName: stringValue(raw.display_name, stringValue(raw.username)),
    role: stringValue(raw.role, 'viewer') as UserRole,
    status: stringValue(raw.status, 'active') as ManagedUser['status'],
    lastLoginAt: optionalString(raw.last_login_at),
    createdAt: stringValue(raw.created_at),
    updatedAt: stringValue(raw.updated_at),
  }
}

function adaptAuditLog(rawValue: unknown): AuditLog {
  const raw = asRecord(rawValue)
  return {
    id: numberValue(raw.id),
    actorUserId: optionalString(raw.actor_user_id),
    action: stringValue(raw.action),
    resourceType: stringValue(raw.resource_type),
    resourceId: stringValue(raw.resource_id),
    details: asRecord(raw.details),
    createdAt: stringValue(raw.created_at),
  }
}

function adaptDashboard(rawValue: unknown): DashboardStats {
  const raw = asRecord(rawValue)
  return {
    onlineAgents: numberValue(raw.online_agents),
    totalAgents: numberValue(raw.agents),
    activeTasks: numberValue(raw.active_tasks),
    totalTasks: numberValue(raw.tasks),
    totalArticles: numberValue(raw.articles),
    todayArticles: numberValue(raw.today_articles),
    successRate:
      typeof raw.task_success_rate === 'number' ? raw.task_success_rate : null,
    succeededTasks: numberValue(raw.succeeded_tasks),
    failedTasks: numberValue(raw.failed_tasks),
    trends: Array.isArray(raw.trends)
      ? raw.trends.map((item) => {
          const point = asRecord(item)
          return {
            date: stringValue(point.date),
            tasks: numberValue(point.tasks),
            articles: numberValue(point.articles),
          }
        })
      : [],
    database: stringValue(raw.database, 'unknown'),
  }
}

function pageParams(query: {
  page?: number
  pageSize?: number
  keyword?: string
}) {
  return {
    page: query.page ?? 1,
    page_size: query.pageSize ?? 20,
    search: query.keyword || undefined,
  }
}

export const api = {
  auth: {
    async login(payload: LoginPayload): Promise<LoginResult> {
      const response = await http.post<Envelope<LoginResult>>('/auth/login', {
        username: payload.email,
        password: payload.password,
      })
      return unwrap(response.data)
    },
    async me() {
      const response = await http.get<Envelope<LoginResult['user']>>('/auth/me')
      return unwrap(response.data)
    },
    async password(currentPassword: string, newPassword: string) {
      await http.post('/auth/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
    },
  },

  agents: {
    async list(query: AgentListQuery = {}): Promise<PageResult<Agent>> {
      const response = await http.get('/agents', {
        params: {
          ...pageParams(query),
          status:
            query.status && query.status !== 'all' ? query.status : undefined,
        },
      })
      const page = unwrapPage<unknown>(response.data)
      return { ...page, items: page.items.map(adaptAgent) }
    },

    async get(id: string): Promise<Agent> {
      const response = await http.get<Envelope<unknown>>(`/agents/${id}`)
      return adaptAgent(unwrap(response.data))
    },

    async latestLogin(id: string): Promise<QrCodePayload | null> {
      try {
        const response = await http.get<Envelope<unknown>>(
          `/agents/${id}/wechat/login/latest`,
        )
        return adaptQr(unwrap(response.data))
      } catch (error) {
        const status = (error as { response?: { status?: number } }).response?.status
        if (status === 404) return null
        throw error
      }
    },

    async requestLogin(id: string): Promise<QrCodePayload> {
      const response = await http.post<Envelope<unknown>>(
        `/agents/${id}/wechat/login`,
      )
      return adaptQr(unwrap(response.data))
    },

    async screen(id: string): Promise<ScreenSnapshot> {
      const response = await http.get<Envelope<unknown>>(`/agents/${id}/screen`)
      return adaptScreen(unwrap(response.data))
    },

    async logs(id: string): Promise<AgentLog[]> {
      const response = await http.get<Envelope<unknown[]>>(`/agents/${id}/logs`)
      return unwrap(response.data).map(adaptLog)
    },

    async updateConfig(
      id: string,
      input: { config?: Record<string, unknown>; maxConcurrency?: number },
    ) {
      const response = await http.patch<Envelope<unknown>>(
        `/agents/${id}/config`,
        {
          config: input.config,
          max_concurrency: input.maxConcurrency,
        },
      )
      return adaptAgent(unwrap(response.data))
    },

    async disable(id: string) {
      await http.delete(`/agents/${id}`)
    },
  },

  tasks: {
    async list(query: TaskListQuery = {}): Promise<PageResult<Task>> {
      const response = await http.get('/tasks', {
        params: {
          ...pageParams(query),
          status:
            query.status && query.status !== 'all' ? query.status : undefined,
          agent_id: query.agentId || undefined,
        },
      })
      const page = unwrapPage<unknown>(response.data)
      return { ...page, items: page.items.map(adaptTask) }
    },

    async get(id: string): Promise<Task> {
      const response = await http.get<Envelope<unknown>>(`/tasks/${id}`)
      return adaptTask(unwrap(response.data))
    },

    async create(payload: CreateTaskPayload, agentId?: string): Promise<Task> {
      const response = await http.post<Envelope<unknown>>('/tasks', {
        type: 'fetch_articles',
        priority: payload.priority,
        agent_id: agentId || undefined,
        payload: {
          account_name: payload.accountName,
          max_articles: payload.maxArticles,
          include_content: payload.includeContent,
          since: payload.since || undefined,
          until: payload.until || undefined,
        },
      })
      return adaptTask(unwrap(response.data))
    },

    async cancel(id: string, reason = '运营手工取消') {
      const response = await http.post<Envelope<unknown>>(
        `/tasks/${id}/cancel`,
        { reason },
      )
      return adaptTask(unwrap(response.data))
    },

    async events(id: string): Promise<TaskEvent[]> {
      const response = await http.get<Envelope<unknown[]>>(
        `/tasks/${id}/events`,
      )
      return unwrap(response.data).map(adaptTaskEvent)
    },
  },

  articles: {
    async list(query: ArticleListQuery = {}): Promise<PageResult<Article>> {
      const response = await http.get('/articles', {
        params: {
          ...pageParams(query),
          account_name: query.accountName || undefined,
          status:
            query.status && query.status !== 'all' ? query.status : undefined,
          task_id: query.taskId || undefined,
          agent_id: query.agentId || undefined,
          from: query.dateRange?.[0] || undefined,
          until: query.dateRange?.[1] || undefined,
        },
      })
      const page = unwrapPage<unknown>(response.data)
      return { ...page, items: page.items.map(adaptArticle) }
    },

    async get(id: string): Promise<Article> {
      const response = await http.get<Envelope<unknown>>(`/articles/${id}`)
      return adaptArticle(unwrap(response.data))
    },

    async accounts(): Promise<string[]> {
      const response = await http.get<Envelope<string[]>>('/articles/accounts')
      return unwrap(response.data)
    },

    async export(
      format: 'json' | 'csv' | 'markdown',
      query: ArticleListQuery = {},
    ) {
      const response = await http.get<Blob>('/articles/export', {
        params: {
          format,
          search: query.keyword || undefined,
          account_name: query.accountName || undefined,
          status:
            query.status && query.status !== 'all' ? query.status : undefined,
          task_id: query.taskId || undefined,
          agent_id: query.agentId || undefined,
          from: query.dateRange?.[0] || undefined,
          until: query.dateRange?.[1] || undefined,
        },
        responseType: 'blob',
      })
      return response
    },
  },

  dashboard: {
    async get(): Promise<DashboardStats> {
      const response = await http.get<Envelope<unknown>>('/dashboard')
      return adaptDashboard(unwrap(response.data))
    },
  },

  settings: {
    async get(): Promise<SettingsResponse> {
      const response = await http.get<Envelope<unknown>>('/settings')
      const raw = asRecord(unwrap(response.data))
      const settings = asRecord(raw.settings)
      return {
        displayName: stringValue(raw.display_name),
        settings: {
          notifyTaskFailed:
            settings.notify_task_failed === undefined
              ? undefined
              : Boolean(settings.notify_task_failed),
          notifyAgentOffline:
            settings.notify_agent_offline === undefined
              ? undefined
              : Boolean(settings.notify_agent_offline),
          notifyDailyDigest:
            settings.notify_daily_digest === undefined
              ? undefined
              : Boolean(settings.notify_daily_digest),
          autoRefresh:
            settings.auto_refresh === undefined
              ? undefined
              : Boolean(settings.auto_refresh),
          refreshInterval:
            typeof settings.refresh_interval === 'number'
              ? settings.refresh_interval
              : undefined,
          retentionDays:
            typeof settings.article_retention_days === 'number'
              ? settings.article_retention_days
              : undefined,
          screenshotQuality:
            settings.screenshot_quality === 'high' ? 'high' : undefined,
          timezone:
            typeof settings.timezone === 'string' ? settings.timezone : undefined,
        },
      }
    },

    async update(displayName: string, settings: Partial<UserSettings>) {
      const response = await http.patch<Envelope<unknown>>('/settings', {
        display_name: displayName,
        settings: {
          notify_task_failed: settings.notifyTaskFailed,
          notify_agent_offline: settings.notifyAgentOffline,
          notify_daily_digest: settings.notifyDailyDigest,
          auto_refresh: settings.autoRefresh,
          refresh_interval: settings.refreshInterval,
          article_retention_days: settings.retentionDays,
          screenshot_quality: settings.screenshotQuality,
          timezone: settings.timezone,
        },
      })
      const raw = asRecord(unwrap(response.data))
      return {
        displayName: stringValue(raw.display_name),
        settings: asRecord(raw.settings),
      }
    },
  },

  users: {
    async list(page = 1, pageSize = 20): Promise<PageResult<ManagedUser>> {
      const response = await http.get('/users', {
        params: { page, page_size: pageSize },
      })
      const result = unwrapPage<unknown>(response.data)
      return { ...result, items: result.items.map(adaptUser) }
    },
    async create(input: {
      username: string
      password: string
      displayName: string
      role: UserRole
    }) {
      const response = await http.post<Envelope<unknown>>('/users', {
        username: input.username,
        password: input.password,
        display_name: input.displayName,
        role: input.role,
      })
      return adaptUser(unwrap(response.data))
    },
    async update(
      id: string,
      input: {
        displayName?: string
        role?: UserRole
        status?: ManagedUser['status']
        password?: string
      },
    ) {
      const response = await http.patch<Envelope<unknown>>(`/users/${id}`, {
        display_name: input.displayName,
        role: input.role,
        status: input.status,
        password: input.password,
      })
      return adaptUser(unwrap(response.data))
    },
    async remove(id: string) {
      await http.delete(`/users/${id}`)
    },
  },
  auditLogs: {
    async list(page = 1, pageSize = 20): Promise<PageResult<AuditLog>> {
      const response = await http.get('/audit-logs', {
        params: { page, page_size: pageSize },
      })
      const result = unwrapPage<unknown>(response.data)
      return { ...result, items: result.items.map(adaptAuditLog) }
    },
  },
}
