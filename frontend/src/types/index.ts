export type AgentStatus = 'online' | 'busy' | 'offline' | 'disabled' | 'error'
export type LoginStatus = 'logged_in' | 'logged_out' | 'unknown'
export type TaskType = 'fetch_articles'
export type TaskStatus =
  | 'pending'
  | 'assigned'
  | 'running'
  | 'succeeded'
  | 'failed'
  | 'canceled'
export type ArticleStatus = 'complete' | 'metadata_only' | 'content_missing' | 'failed'
export type LogLevel = 'debug' | 'info' | 'warning' | 'error'
export type UserRole = 'admin' | 'operator' | 'viewer'

export interface User {
  id: string
  name: string
  email: string
  role: UserRole
}

export interface LoginPayload {
  email: string
  password: string
}

export interface LoginResult {
  token: string
  user: User
}

export interface AgentMetrics {
  cpu: number | null
  memory: number | null
  disk: number | null
  latency: number | null
}

export interface AgentWechat {
  nickname: string
  accountId: string
  loginStatus: LoginStatus
  lastSeenAt: string
}

export interface AgentConfig {
  heartbeatInterval: number
  maxConcurrency: number
  screenshotEnabled: boolean
  autoReconnect: boolean
  browserMode: 'desktop'
}

export interface Agent {
  id: string
  code: string
  name: string
  status: AgentStatus
  hostname: string
  version: string
  os: string
  activeTasks: number
  lastHeartbeat: string
  capabilities: string[]
  metrics: AgentMetrics
  wechat: AgentWechat
  config: AgentConfig
  configVersion: number
  rawConfig: Record<string, unknown>
}

export interface AgentLog {
  id: string
  agentId: string
  level: LogLevel
  message: string
  createdAt: string
  fields?: Record<string, unknown>
}

export type QrStatus =
  | 'pending'
  | 'qr_ready'
  | 'scanned'
  | 'confirmed'
  | 'expired'
  | 'failed'
  | 'canceled'

export interface QrCodePayload {
  id: string
  agentId: string
  imageUrl: string
  mimeType: string
  expiresAt: string
  status: QrStatus
  message?: string
}

export interface ScreenSnapshot {
  imageUrl?: string
  capturedAt: string
  width: number
  height: number
  online: boolean
  message?: string
}

export interface CreateTaskPayload {
  accountName: string
  maxArticles: number
  priority: number
  since?: string
  until?: string
  includeContent: boolean
}

export interface TaskProgress {
  stage: string
  percent: number
  message?: string
  current?: number
  total?: number
}

export interface Task {
  id: string
  name: string
  type: TaskType
  status: TaskStatus
  priority: number
  requestedAgentId?: string
  agentId: string
  agentName: string
  accountName: string
  maxArticles: number
  since?: string
  until?: string
  includeContent: boolean
  progress: number
  stage: string
  progressMessage: string
  collected: number
  total: number
  attempts: number
  maxAttempts: number
  lastError?: string
  errorCode?: string
  complete: boolean
  createdAt: string
  startedAt?: string
  finishedAt?: string
  updatedAt: string
}

export interface TaskEvent {
  id: number
  taskId: string
  agentId?: string
  type: string
  message: string
  data?: Record<string, unknown>
  createdAt: string
}

export interface Article {
  id: string
  taskId: string
  agentId: string
  title: string
  accountName: string
  accountId?: string
  author: string
  digest: string
  contentHtml: string
  contentText: string
  coverUrl: string
  url: string
  publishedAt?: string
  fetchedAt: string
  collectedAt: string
  status: ArticleStatus
  complete: boolean
  source: string
  quality: Record<string, unknown>
}

export interface ListQuery {
  page?: number
  pageSize?: number
  keyword?: string
}

export interface AgentListQuery extends ListQuery {
  status?: AgentStatus | 'all'
}

export interface TaskListQuery extends ListQuery {
  status?: TaskStatus | 'all'
  agentId?: string
}

export interface ArticleListQuery extends ListQuery {
  accountName?: string
  status?: ArticleStatus | 'all'
  taskId?: string
  agentId?: string
  dateRange?: [string, string]
}

export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export interface TrendPoint {
  date: string
  tasks: number
  articles: number
}

export interface DashboardStats {
  onlineAgents: number
  totalAgents: number
  activeTasks: number
  totalTasks: number
  totalArticles: number
  todayArticles: number
  successRate: number | null
  succeededTasks: number
  failedTasks: number
  trends: TrendPoint[]
  database: string
}

export interface UserSettings {
  displayName: string
  notifyTaskFailed: boolean
  notifyAgentOffline: boolean
  notifyDailyDigest: boolean
  autoRefresh: boolean
  refreshInterval: number
  retentionDays: number
  screenshotQuality: 'standard' | 'high'
  timezone: string
}

export interface SettingsResponse {
  displayName: string
  settings: Partial<UserSettings>
}

export interface ManagedUser {
  id: string
  username: string
  displayName: string
  role: UserRole
  status: 'active' | 'disabled'
  lastLoginAt?: string
  createdAt: string
  updatedAt: string
}

export interface AuditLog {
  id: number
  actorUserId?: string
  action: string
  resourceType: string
  resourceId: string
  details: Record<string, unknown>
  createdAt: string
}

export interface RealtimeEventMap {
  'agent.status': { agentId: string; status: AgentStatus; lastHeartbeat: string }
  'task.progress': {
    taskId: string
    progress: number
    collected: number
    status: TaskStatus
  }
  'article.created': { articleId: string }
  'agent.log': AgentLog
  'agent.screen': { agentId: string; capturedAt: string }
  'agent.qrcode': { agentId: string; status: QrStatus }
}
