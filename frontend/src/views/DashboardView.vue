<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import {
  ArrowRight,
  Clock3,
  Database,
  FileText,
  Plus,
  RadioTower,
  RefreshCw,
  TrendingUp,
} from 'lucide-vue-next'
import { api } from '@/services/api'
import { useAgentStore } from '@/stores/agents'
import { useAuthStore } from '@/stores/auth'
import { useTaskStore } from '@/stores/tasks'
import type { Article, DashboardStats, Task } from '@/types'
import { formatDateTime, formatRelative, formatTime } from '@/utils/format'
import StatusPill from '@/components/common/StatusPill.vue'
import ActivityChart from '@/components/dashboard/ActivityChart.vue'
import FleetNetwork from '@/components/dashboard/FleetNetwork.vue'
import { usePolling } from '@/composables/usePolling'

const router = useRouter()
const auth = useAuthStore()
const agents = useAgentStore()
const taskStore = useTaskStore()
const tasks = ref<Task[]>([])
const articles = ref<Article[]>([])
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const lastUpdated = ref(dayjs())

const greeting = computed(() => {
  const hour = dayjs().hour()
  const prefix = hour < 12 ? '早上好' : hour < 18 ? '下午好' : '晚上好'
  return `${prefix}，${auth.user?.name || '运营同事'}。`
})

const attentionAgents = computed(() =>
  agents.items
    .filter((agent) => agent.status === 'offline' || agent.status === 'error')
    .slice(0, 3),
)

const activeTasks = computed(() =>
  tasks.value.filter(
    (task) =>
      task.status === 'pending' ||
      task.status === 'assigned' ||
      task.status === 'running',
  ),
)

async function loadDashboard() {
  loading.value = true
  try {
    const [dashboard, agentResult, taskResult, articleResult] = await Promise.all([
      api.dashboard.get(),
      api.agents.list({ page: 1, pageSize: 100 }),
      api.tasks.list({ page: 1, pageSize: 8 }),
      api.articles.list({ page: 1, pageSize: 6 }),
    ])
    stats.value = dashboard
    agents.items = agentResult.items
    agents.total = agentResult.total
    tasks.value = taskResult.items
    taskStore.total = taskResult.total
    articles.value = articleResult.items
    lastUpdated.value = dayjs()
  } finally {
    loading.value = false
  }
}

function createTask() {
  router.push({ name: 'tasks', query: { create: '1' } })
}

function openAgent(agentId: string) {
  router.push({ name: 'agent-detail', params: { id: agentId } })
}

onMounted(async () => {
  await loadDashboard()
})

usePolling(loadDashboard)
</script>

<template>
  <div class="page-shell dashboard">
    <header class="dashboard-header">
      <div>
        <p class="eyebrow">LIVE OPERATIONS</p>
        <h1 class="page-title">{{ greeting }}</h1>
        <p class="page-lead">这里展示平台服务端返回的真实节点、任务与文章状态。</p>
      </div>
      <div class="header-actions">
        <span class="updated-at">更新于 {{ formatTime(lastUpdated.toISOString()) }}</span>
        <button class="icon-button" title="刷新数据" :disabled="loading" @click="loadDashboard">
          <RefreshCw :size="16" :class="{ spinning: loading }" />
        </button>
        <el-button type="primary" @click="createTask">
          <Plus :size="15" />
          创建采集任务
        </el-button>
      </div>
    </header>

    <section v-if="stats" class="metrics-band">
      <div class="metric-intro">
        <span class="panel-kicker">NETWORK SNAPSHOT</span>
        <div>
          <span class="big-number mono">{{ stats.onlineAgents }}</span>
          <span class="number-denom mono">/ {{ stats.totalAgents }}</span>
        </div>
        <p><span class="signal-dot" /> Agent 在线</p>
      </div>
      <div class="metric-cell">
        <span class="metric-icon"><RadioTower :size="16" /></span>
        <div>
          <strong class="mono">{{ stats.activeTasks }}</strong>
          <span>活跃任务</span>
        </div>
        <small>共 {{ stats.totalTasks }} 个任务</small>
      </div>
      <div class="metric-cell">
        <span class="metric-icon"><FileText :size="16" /></span>
        <div>
          <strong class="mono">{{ stats.totalArticles.toLocaleString() }}</strong>
          <span>文章资产</span>
        </div>
        <small>今日入库 {{ stats.todayArticles }}</small>
      </div>
      <div class="metric-cell">
        <span class="metric-icon"><TrendingUp :size="16" /></span>
        <div>
          <strong class="mono">{{ stats.successRate === null ? '--' : stats.successRate.toFixed(1) }}%</strong>
          <span>任务成功率</span>
        </div>
        <small>{{ stats.succeededTasks }} 成功 / {{ stats.failedTasks }} 失败</small>
      </div>
      <div class="metric-note">
        <Database :size="15" />
        <p>{{ stats.database === 'ok' ? '数据库连接正常' : '数据库状态未知' }}</p>
        <span>REST 轮询 {{ formatTime(lastUpdated.toISOString()) }}</span>
      </div>
    </section>

    <div class="dashboard-grid">
      <div class="dashboard-main">
        <ActivityChart :trends="stats?.trends ?? []" />

        <section class="task-queue panel">
          <div class="panel-head">
            <div>
              <span class="panel-kicker">ACTIVE PIPELINES</span>
              <h2 class="panel-title">活跃任务</h2>
            </div>
            <button class="text-button" @click="router.push({ name: 'tasks' })">
              查看全部 <ArrowRight :size="13" />
            </button>
          </div>
          <div v-if="activeTasks.length" class="task-list">
            <button
              v-for="task in activeTasks"
              :key="task.id"
              class="task-row"
              @click="router.push({ name: 'tasks', query: { task: task.id } })"
            >
              <span class="task-agent">
                <i :class="task.status" />
                <span>
                  <strong>{{ task.accountName }}</strong>
                  <small>{{ task.agentName }} · {{ task.stage || '等待调度' }}</small>
                </span>
              </span>
              <span class="task-progress">
                <span class="progress-meta">
                  <small class="mono">{{ task.collected }} / {{ task.total }}</small>
                  <strong class="mono">{{ task.progress }}%</strong>
                </span>
                <span class="progress-track">
                  <i :style="{ width: `${task.progress}%` }" />
                </span>
              </span>
              <span class="task-eta">
                <Clock3 :size="12" />
                {{ task.updatedAt ? formatRelative(task.updatedAt) : '刚创建' }}
              </span>
              <StatusPill :status="task.status" />
            </button>
          </div>
          <div v-else class="queue-empty">当前没有活跃采集任务</div>
        </section>

        <section class="recent-articles panel">
          <div class="panel-head">
            <div>
              <span class="panel-kicker">LATEST INGEST</span>
              <h2 class="panel-title">最新入库</h2>
            </div>
            <button class="text-button" @click="router.push({ name: 'articles' })">
              打开资料库 <ArrowRight :size="13" />
            </button>
          </div>
          <div v-if="articles.length" class="article-list">
            <button
              v-for="(article, index) in articles"
              :key="article.id"
              class="article-row"
              @click="router.push({ name: 'articles', query: { article: article.id } })"
            >
              <span class="article-index mono">{{ String(index + 1).padStart(2, '0') }}</span>
              <span class="article-cover">
                <img v-if="article.coverUrl" :src="article.coverUrl" alt="" loading="lazy" />
                <span v-else>{{ article.accountName.slice(0, 1) }}</span>
              </span>
              <span class="article-copy">
                <strong>{{ article.title }}</strong>
                <small>{{ article.accountName }} · {{ article.author || '未署名' }}</small>
              </span>
              <span class="article-read mono"><StatusPill :status="article.status" /></span>
              <span class="article-time">{{ formatRelative(article.collectedAt) }}</span>
              <ArrowRight :size="14" class="row-arrow" />
            </button>
          </div>
          <div v-else class="queue-empty">暂无文章入库</div>
        </section>
      </div>

      <aside class="dashboard-side">
        <FleetNetwork :agents="agents.items" />

        <section class="attention-panel panel">
          <div class="panel-head">
            <h2 class="panel-title"><span class="attention-mark">!</span> 需要关注</h2>
            <span class="count-badge mono">{{ attentionAgents.length }}</span>
          </div>
          <button
            v-for="agent in attentionAgents"
            :key="agent.id"
            class="attention-row"
            @click="openAgent(agent.id)"
          >
            <span class="attention-code mono">{{ agent.code }}</span>
            <span>
              <strong>{{ agent.status === 'offline' ? '节点离线' : '节点异常' }}</strong>
              <small>{{ agent.name }} · {{ formatRelative(agent.lastHeartbeat) }}</small>
            </span>
            <ArrowRight :size="13" />
          </button>
          <div v-if="!attentionAgents.length" class="attention-empty">所有已注册 Agent 状态正常</div>
        </section>

        <section class="system-card">
          <div class="system-top">
            <span class="panel-kicker">SYSTEM STATUS</span>
            <span class="signal-dot" />
          </div>
          <strong class="mono">{{ stats?.database === 'ok' ? 'ONLINE' : 'UNKNOWN' }}</strong>
          <p>SQLite 数据库</p>
          <div class="system-list">
            <div>
              <span>REST API</span>
              <strong>已连接</strong>
            </div>
            <div>
              <span>实时方式</span>
              <strong>短轮询</strong>
            </div>
            <div>
              <span>最近刷新</span>
              <strong>{{ formatDateTime(lastUpdated.toISOString()) }}</strong>
            </div>
          </div>
        </section>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.dashboard-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 9px;
}

.updated-at {
  margin-right: 8px;
  color: var(--muted);
  font-size: 9px;
}

.spinning {
  animation: spin 0.8s linear infinite;
}

.metrics-band {
  display: grid;
  min-height: 146px;
  margin-bottom: 18px;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--surface);
  grid-template-columns: 1.25fr repeat(3, 1fr) 0.92fr;
  animation: rise-in 0.5s 0.05s ease both;
}

.metric-intro,
.metric-cell,
.metric-note {
  position: relative;
  padding: 20px 21px;
  border-right: 1px solid var(--line);
}

.metric-intro {
  color: #f4f6f1;
  background: var(--ink);
  border-radius: 7px 0 0 7px;
}

.metric-intro .panel-kicker {
  color: rgba(255, 255, 255, 0.34);
}

.metric-intro > div {
  display: flex;
  align-items: baseline;
  margin-top: 8px;
}

.big-number {
  font-size: 42px;
  font-weight: 520;
  letter-spacing: -0.06em;
  line-height: 1;
}

.number-denom {
  margin-left: 4px;
  color: rgba(255, 255, 255, 0.34);
  font-size: 15px;
}

.metric-intro p {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 10px 0 0;
  color: rgba(255, 255, 255, 0.52);
  font-size: 10px;
}

.metric-cell {
  display: flex;
  justify-content: space-between;
  flex-direction: column;
}

.metric-icon {
  display: grid;
  width: 29px;
  height: 29px;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 5px;
  color: var(--green-dark);
  background: var(--paper);
}

.metric-cell > div {
  display: flex;
  align-items: baseline;
  gap: 7px;
}

.metric-cell strong {
  font-size: 27px;
  font-weight: 560;
  letter-spacing: -0.045em;
}

.metric-cell span {
  color: var(--muted);
  font-size: 9px;
}

.metric-cell small {
  color: var(--green-dark);
  font-size: 9px;
}

.metric-note {
  display: flex;
  justify-content: center;
  flex-direction: column;
  border-right: 0;
  border-radius: 0 7px 7px 0;
  color: var(--green-dark);
  background: rgba(201, 243, 106, 0.18);
}

.metric-note p {
  margin: 10px 0 4px;
  color: var(--ink-2);
  font-size: 11px;
  font-weight: 650;
}

.metric-note span {
  color: var(--muted);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 8px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.75fr) minmax(320px, 0.82fr);
  gap: 18px;
}

.dashboard-main,
.dashboard-side {
  display: flex;
  min-width: 0;
  gap: 18px;
  flex-direction: column;
}

.dashboard-main > *,
.dashboard-side > * {
  animation: rise-in 0.5s ease both;
}

.task-queue,
.recent-articles,
.attention-panel {
  overflow: hidden;
}

.panel-head > div {
  display: flex;
  gap: 3px;
  flex-direction: column;
}

.task-list,
.article-list {
  display: flex;
  flex-direction: column;
}

.task-row {
  display: grid;
  min-height: 70px;
  align-items: center;
  gap: 20px;
  padding: 11px 18px;
  border: 0;
  border-bottom: 1px solid var(--line);
  background: transparent;
  text-align: left;
  cursor: pointer;
  grid-template-columns: minmax(165px, 1.1fr) minmax(160px, 1.35fr) 92px 78px;
}

.task-row:last-child,
.article-row:last-child,
.attention-row:last-child {
  border-bottom: 0;
}

.task-row:hover,
.article-row:hover,
.attention-row:hover {
  background: #f5f3ec;
}

.task-agent {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.task-agent > i {
  width: 5px;
  height: 30px;
  flex: 0 0 5px;
  border-radius: 3px;
  background: var(--green);
}

.task-agent > i.pending {
  background: var(--line-dark);
}

.task-agent > i.assigned {
  background: var(--amber);
}

.task-agent span,
.article-copy,
.attention-row > span:nth-child(2) {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.task-agent strong,
.article-copy strong,
.attention-row strong {
  overflow: hidden;
  font-size: 11px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-agent small,
.article-copy small,
.attention-row small {
  overflow: hidden;
  color: var(--muted);
  font-size: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-progress {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.progress-meta {
  display: flex;
  justify-content: space-between;
  color: var(--muted);
  font-size: 8px;
}

.progress-meta strong {
  color: var(--ink-2);
  font-size: 8px;
}

.progress-track {
  height: 3px;
  overflow: hidden;
  background: #e3e0d7;
}

.progress-track i {
  display: block;
  height: 100%;
  background: var(--green);
  transition: width 0.5s ease;
}

.task-eta {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--muted);
  font-size: 8px;
}

.queue-empty,
.attention-empty {
  padding: 38px 20px;
  color: var(--muted);
  font-size: 10px;
  text-align: center;
}

.article-row {
  display: grid;
  min-height: 68px;
  align-items: center;
  gap: 13px;
  padding: 10px 18px;
  border: 0;
  border-bottom: 1px solid var(--line);
  background: transparent;
  text-align: left;
  cursor: pointer;
  grid-template-columns: 24px 42px minmax(0, 1fr) 100px 60px 15px;
}

.article-index {
  color: #a2a69f;
  font-size: 8px;
}

.article-cover {
  display: grid;
  width: 42px;
  height: 42px;
  overflow: hidden;
  place-items: center;
  border-radius: 5px;
  color: rgba(255, 255, 255, 0.8);
  background: var(--ink);
  font-size: 13px;
  font-weight: 700;
}

.article-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.article-read {
  display: flex;
}

.article-time {
  color: var(--muted);
  font-size: 8px;
}

.row-arrow {
  color: var(--muted);
}

.attention-mark {
  display: grid;
  width: 18px;
  height: 18px;
  place-items: center;
  border-radius: 50%;
  color: var(--ink);
  background: #f1af63;
  font-family: serif;
  font-size: 11px;
  font-weight: 800;
}

.count-badge {
  display: grid;
  min-width: 28px;
  height: 21px;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 4px;
  color: var(--muted);
  font-size: 8px;
}

.attention-row {
  display: grid;
  width: 100%;
  min-height: 62px;
  align-items: center;
  gap: 11px;
  padding: 10px 16px;
  border: 0;
  border-bottom: 1px solid var(--line);
  background: transparent;
  text-align: left;
  cursor: pointer;
  grid-template-columns: 50px minmax(0, 1fr) 14px;
}

.attention-code {
  color: var(--orange);
  font-size: 8px;
  font-weight: 650;
}

.system-card {
  padding: 18px;
  border-radius: 8px;
  color: #f1f4ed;
  background: var(--ink);
}

.system-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.system-top .panel-kicker {
  color: rgba(255, 255, 255, 0.32);
}

.system-card > strong {
  display: block;
  margin-top: 15px;
  color: var(--acid);
  font-size: 24px;
  font-weight: 520;
}

.system-card > p {
  margin: 4px 0 14px;
  color: rgba(255, 255, 255, 0.4);
  font-size: 9px;
}

.system-list {
  display: flex;
  flex-direction: column;
}

.system-list div {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.38);
  font-size: 8px;
}

.system-list strong {
  color: rgba(255, 255, 255, 0.68);
  font-weight: 550;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1250px) {
  .metrics-band {
    grid-template-columns: 1.15fr repeat(3, 1fr);
  }

  .metric-note {
    display: none;
  }

  .dashboard-grid {
    grid-template-columns: minmax(0, 1.5fr) minmax(300px, 0.8fr);
  }
}

@media (max-width: 980px) {
  .metrics-band {
    grid-template-columns: 1fr 1fr;
  }

  .metric-intro {
    border-radius: 7px 0 0 0;
  }

  .metric-cell:nth-child(2) {
    border-right: 0;
  }

  .metric-cell:nth-child(3) {
    border-bottom: 1px solid var(--line);
  }

  .metric-cell:nth-child(4) {
    border-right: 0;
    border-bottom: 0;
  }
}

@media (max-width: 1080px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .dashboard-side {
    display: grid;
    align-items: start;
    grid-template-columns: 1fr 1fr;
  }

  .dashboard-side > :first-child {
    grid-row: span 2;
  }
}

@media (max-width: 760px) {
  .dashboard-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .header-actions {
    width: 100%;
  }

  .updated-at {
    display: none;
  }

  .header-actions .el-button {
    margin-left: auto;
  }

  .metrics-band {
    grid-template-columns: 1fr 1fr;
  }

  .metric-intro {
    border-radius: 7px 0 0 0;
  }

  .metric-cell {
    border-bottom: 1px solid var(--line);
  }

  .metric-cell:nth-child(3) {
    border-right: 0;
    border-radius: 0 7px 0 0;
  }

  .metric-cell:nth-child(4) {
    border-bottom: 0;
  }

  .dashboard-side {
    display: flex;
    align-items: stretch;
  }

  .dashboard-side > * {
    width: 100%;
  }

  .task-row {
    gap: 12px;
    grid-template-columns: minmax(130px, 1fr) minmax(110px, 1fr) 70px;
  }

  .task-eta {
    display: none;
  }

  .article-row {
    grid-template-columns: 36px minmax(0, 1fr) 90px 15px;
  }

  .article-index,
  .article-time {
    display: none;
  }
}

@media (max-width: 520px) {
  .metrics-band {
    grid-template-columns: 1fr 1fr;
  }

  .metric-intro,
  .metric-cell {
    padding: 16px;
  }

  .metric-intro {
    min-height: 130px;
  }

  .big-number {
    font-size: 34px;
  }

  .metric-cell > div {
    align-items: flex-start;
    flex-direction: column;
    gap: 3px;
  }

  .metric-cell strong {
    font-size: 23px;
  }

  .task-row {
    grid-template-columns: 1fr 70px;
  }

  .task-progress {
    display: none;
  }

  .article-row {
    grid-template-columns: 36px minmax(0, 1fr) 15px;
  }

  .article-read {
    display: none;
  }
}
</style>
