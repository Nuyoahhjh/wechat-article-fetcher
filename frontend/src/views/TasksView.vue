<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowUpRight,
  Clock3,
  Plus,
  RefreshCw,
  Search,
  SlidersHorizontal,
} from 'lucide-vue-next'
import { useAgentStore } from '@/stores/agents'
import { useTaskStore } from '@/stores/tasks'
import type { TaskStatus } from '@/types'
import { formatDateTime, formatRelative } from '@/utils/format'
import StatusPill from '@/components/common/StatusPill.vue'
import TaskCreateDrawer from '@/components/tasks/TaskCreateDrawer.vue'
import TaskDetailDrawer from '@/components/tasks/TaskDetailDrawer.vue'
import { api } from '@/services/api'
import { usePolling } from '@/composables/usePolling'

const route = useRoute()
const router = useRouter()
const tasks = useTaskStore()
const agents = useAgentStore()
const keyword = ref('')
const activeStatus = ref<TaskStatus | 'all'>('all')
const selectedAgent = ref('')
const createOpen = ref(false)
const initialAgentId = ref('')

const successRate = computed(() => {
  const completed = tasks.items.filter((task) => task.status === 'succeeded').length
  const failed = tasks.items.filter((task) => task.status === 'failed').length
  return completed + failed ? ((completed / (completed + failed)) * 100).toFixed(1) : '--'
})

const filters: Array<{ value: TaskStatus | 'all'; label: string }> = [
  { value: 'all', label: '全部任务' },
  { value: 'pending', label: '待调度' },
  { value: 'assigned', label: '已分配' },
  { value: 'running', label: '采集中' },
  { value: 'succeeded', label: '已完成' },
  { value: 'failed', label: '失败' },
  { value: 'canceled', label: '已取消' },
]

const statusCount = (status: TaskStatus) =>
  tasks.items.filter((task) => task.status === status).length

const statusMix = computed(() => {
  const total = tasks.items.length
  const parts = [
    { status: 'running', count: statusCount('running') + statusCount('assigned') },
    { status: 'pending', count: statusCount('pending') },
    { status: 'succeeded', count: statusCount('succeeded') },
    { status: 'failed', count: statusCount('failed') + statusCount('canceled') },
  ]
  return parts.map((part) => ({
    ...part,
    width: total ? `${(part.count / total) * 100}%` : '0%',
  }))
})

async function load() {
  await Promise.all([
    tasks.fetchTasks({
      page: 1,
      keyword: keyword.value,
      status: activeStatus.value,
      agentId: selectedAgent.value || undefined,
    }),
    agents.fetchAgents({ page: 1, pageSize: 100 }),
  ])
}

async function selectStatus(status: TaskStatus | 'all') {
  activeStatus.value = status
  await load()
}

async function openCreate() {
  initialAgentId.value = selectedAgent.value
  createOpen.value = true
}

async function handleCreated(taskId: string) {
  await load()
  const task = tasks.items.find((item) => item.id === taskId)
  if (task) await tasks.openTask(task)
}

async function handleQueryState() {
  if (route.query.create === '1') {
    initialAgentId.value = typeof route.query.agentId === 'string' ? route.query.agentId : ''
    createOpen.value = true
    router.replace({ query: { ...route.query, create: undefined } })
  }
  if (typeof route.query.agentId === 'string' && route.query.create !== '1') {
    selectedAgent.value = route.query.agentId
    await load()
  }
  if (typeof route.query.task === 'string') {
    const task = await api.tasks.get(route.query.task)
    await tasks.openTask(task)
    router.replace({ query: { ...route.query, task: undefined } })
  }
}

watch(() => route.query, handleQueryState)
onMounted(async () => {
  await load()
  await handleQueryState()
})

usePolling(() => tasks.fetchTasks())
</script>

<template>
  <div class="page-shell tasks-page">
    <header class="page-header">
      <div>
        <p class="eyebrow">COLLECTION PIPELINES</p>
        <h1 class="page-title">采集任务</h1>
        <p class="page-lead">创建、分配并跟踪所有公众号采集任务，实时进度由 Agent 持续回传。</p>
      </div>
      <div class="header-actions">
        <el-button :loading="tasks.loading" @click="load">
          <RefreshCw v-if="!tasks.loading" :size="15" />
          刷新
        </el-button>
        <el-button type="primary" @click="openCreate">
          <Plus :size="15" />
          创建任务
        </el-button>
      </div>
    </header>

    <section class="task-overview">
      <div class="overview-primary">
        <span class="panel-kicker">TODAY'S PIPELINE</span>
        <strong class="mono">{{ tasks.total }}</strong>
        <p>任务记录</p>
        <div class="overview-signal">
          <i />
          轮询同步
        </div>
      </div>
      <div class="overview-metric">
        <span>当前采集中</span>
        <strong class="mono">{{ statusCount('running') + statusCount('assigned') }}</strong>
        <small>多节点并行</small>
      </div>
      <div class="overview-metric">
        <span>等待调度</span>
        <strong class="mono">{{ statusCount('pending') }}</strong>
        <small>按优先级调度</small>
      </div>
      <div class="overview-metric">
        <span>已完成</span>
        <strong class="mono">{{ statusCount('succeeded') }}</strong>
        <small>已完成样本成功率 {{ successRate }}%</small>
      </div>
      <div class="overview-chart">
        <span class="panel-kicker">STATUS MIX</span>
        <div class="status-mix">
          <span
            v-for="part in statusMix"
            :key="part.status"
            :class="`mix-${part.status}`"
            :style="{ width: part.width }"
          />
        </div>
        <div class="mix-legend">
          <span><i class="running" />运行</span>
          <span><i class="pending" />等待</span>
          <span><i class="succeeded" />完成</span>
          <span><i class="failed" />失败</span>
        </div>
      </div>
    </section>

    <section class="task-panel panel">
      <div class="task-toolbar">
        <div class="status-tabs">
          <button
            v-for="filter in filters"
            :key="filter.value"
            :class="{ active: activeStatus === filter.value }"
            @click="selectStatus(filter.value)"
          >
            {{ filter.label }}
          </button>
        </div>
        <div class="toolbar-spacer" />
        <el-select
          v-model="selectedAgent"
          class="agent-filter"
          placeholder="全部 Agent"
          clearable
          @change="load"
        >
          <template #prefix><SlidersHorizontal :size="13" /></template>
          <el-option
            v-for="agent in agents.items"
            :key="agent.id"
            :label="agent.name"
            :value="agent.id"
          />
        </el-select>
        <el-input
          v-model="keyword"
          class="task-search"
          placeholder="搜索任务或公众号"
          clearable
          @keyup.enter="load"
          @clear="load"
        >
          <template #prefix><Search :size="14" /></template>
        </el-input>
      </div>

      <el-table
        v-loading="tasks.loading"
        :data="tasks.items"
        row-key="id"
        @row-click="tasks.openTask"
      >
        <el-table-column label="任务" min-width="245">
          <template #default="{ row }">
            <div class="task-name">
              <i :class="row.status" />
              <span>
                <strong>{{ row.name }}</strong>
                <small><span class="mono">{{ row.id }}</span> · {{ formatDateTime(row.createdAt) }}</small>
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="105">
          <template #default="{ row }"><StatusPill :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="Agent" min-width="135">
          <template #default="{ row }">
            <div class="agent-cell">
              <strong>{{ row.agentName }}</strong>
              <small>{{ row.accountName }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="采集进度" min-width="200">
          <template #default="{ row }">
            <div class="progress-cell">
              <div class="progress-meta">
                <span class="mono">{{ row.collected.toLocaleString() }} / {{ row.total.toLocaleString() }}</span>
                <strong class="mono">{{ row.progress }}%</strong>
              </div>
              <div class="progress-track">
                <i :class="row.status" :style="{ width: `${row.progress}%` }" />
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          label="重试"
          width="75"
          align="center"
          class-name="task-retry-column"
          label-class-name="task-retry-column"
        >
          <template #default="{ row }">
            <span class="fail-count mono" :class="{ hasError: row.attempts > 1 }">{{ row.attempts }}/{{ row.maxAttempts }}</span>
          </template>
        </el-table-column>
        <el-table-column
          label="阶段"
          width="105"
          class-name="task-stage-column"
          label-class-name="task-stage-column"
        >
          <template #default="{ row }">
            <span class="eta-cell"><Clock3 :size="12" />{{ row.stage || '等待中' }}</span>
          </template>
        </el-table-column>
        <el-table-column
          label="更新"
          width="100"
          class-name="task-updated-column"
          label-class-name="task-updated-column"
        >
          <template #default="{ row }">
            <span class="updated-cell">{{ formatRelative(row.updatedAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="" width="48">
          <template #default="{ row }">
            <button class="row-link" title="查看任务" @click.stop="tasks.openTask(row)">
              <ArrowUpRight :size="15" />
            </button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-row">
        <span>显示 {{ tasks.items.length }} / {{ tasks.total }} 条任务</span>
        <el-pagination
          size="small"
          background
          layout="prev, pager, next"
          :total="tasks.total"
          :page-size="tasks.query.pageSize"
          @current-change="(page: number) => tasks.fetchTasks({ page })"
        />
      </div>
    </section>

    <TaskCreateDrawer
      v-model="createOpen"
      :initial-agent-id="initialAgentId"
      @created="handleCreated"
    />
    <TaskDetailDrawer />
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
}

.header-actions {
  display: flex;
  gap: 9px;
}

.task-overview {
  display: grid;
  min-height: 118px;
  margin-bottom: 18px;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--surface);
  grid-template-columns: 1.25fr repeat(3, 0.8fr) 1.25fr;
  animation: rise-in 0.45s 0.05s ease both;
}

.overview-primary {
  position: relative;
  padding: 18px 20px;
  color: #f4f6f1;
  background: var(--ink);
}

.overview-primary .panel-kicker {
  color: rgba(255, 255, 255, 0.32);
}

.overview-primary strong {
  display: block;
  margin-top: 8px;
  color: var(--acid);
  font-size: 34px;
  font-weight: 500;
  letter-spacing: -0.05em;
  line-height: 1;
}

.overview-primary p {
  margin: 4px 0 0;
  color: rgba(255, 255, 255, 0.4);
  font-size: 9px;
}

.overview-signal {
  position: absolute;
  right: 18px;
  bottom: 17px;
  display: flex;
  align-items: center;
  gap: 6px;
  color: rgba(255, 255, 255, 0.38);
  font-size: 8px;
}

.overview-signal i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--acid);
}

.overview-metric {
  display: flex;
  justify-content: center;
  padding: 18px;
  border-right: 1px solid var(--line);
  flex-direction: column;
  gap: 4px;
}

.overview-metric span,
.overview-metric small {
  color: var(--muted);
  font-size: 8px;
}

.overview-metric strong {
  font-size: 25px;
  font-weight: 550;
  line-height: 1;
}

.overview-metric small {
  color: var(--green-dark);
}

.overview-chart {
  display: flex;
  justify-content: center;
  padding: 18px;
  flex-direction: column;
}

.status-mix {
  display: flex;
  height: 8px;
  gap: 2px;
  margin: 13px 0 8px;
}

.status-mix span {
  height: 100%;
  border-radius: 2px;
}

.mix-running {
  background: var(--green);
}

.mix-pending {
  background: #c9c6bc;
}

.mix-assigned {
  background: var(--amber);
}

.mix-succeeded {
  background: #7eb1a0;
}

.mix-failed {
  background: var(--orange);
}

.mix-legend {
  display: flex;
  gap: 9px;
  color: var(--muted);
  font-size: 7px;
}

.mix-legend span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.mix-legend i {
  width: 4px;
  height: 4px;
  border-radius: 50%;
}

.mix-legend i.running {
  background: var(--green);
}

.mix-legend i.pending {
  background: #c9c6bc;
}

.mix-legend i.succeeded {
  background: #7eb1a0;
}

.mix-legend i.failed {
  background: var(--orange);
}

.task-panel {
  overflow: hidden;
  animation: rise-in 0.45s 0.1s ease both;
}

.task-toolbar {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 62px;
  padding: 11px 16px;
  border-bottom: 1px solid var(--line);
}

.status-tabs {
  display: flex;
  gap: 2px;
}

.status-tabs button {
  height: 30px;
  padding: 0 9px;
  border: 1px solid transparent;
  border-radius: 5px;
  color: var(--muted);
  background: transparent;
  font-size: 10px;
  cursor: pointer;
}

.status-tabs button:hover {
  background: var(--paper);
}

.status-tabs button.active {
  border-color: var(--line);
  color: var(--ink);
  background: #fff;
  font-weight: 650;
}

.agent-filter {
  width: 150px;
}

.task-search {
  width: 210px;
}

.task-name {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.task-name > i {
  width: 3px;
  height: 30px;
  flex: 0 0 3px;
  border-radius: 2px;
  background: var(--green);
}

.task-name > i.pending {
  background: #aaa9a2;
}

.task-name > i.assigned {
  background: var(--amber);
}

.task-name > i.succeeded {
  background: #7eb1a0;
}

.task-name > i.failed {
  background: var(--orange);
}

.task-name > span,
.agent-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.task-name strong,
.agent-cell strong {
  overflow: hidden;
  font-size: 11px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-name small,
.agent-cell small {
  overflow: hidden;
  color: var(--muted);
  font-size: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-name small .mono {
  color: #939891;
}

.agent-cell small {
  font-size: 9px;
}

.progress-cell {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.progress-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--muted);
  font-size: 8px;
}

.progress-meta strong {
  color: var(--ink-2);
  font-size: 8px;
}

.progress-track {
  height: 4px;
  overflow: hidden;
  background: #e7e4db;
}

.progress-track i {
  display: block;
  height: 100%;
  background: var(--green);
  transition: width 0.5s ease;
}

.progress-track i.pending {
  background: #aaa9a2;
}

.progress-track i.assigned {
  background: var(--amber);
}

.progress-track i.failed {
  background: var(--orange);
}

.fail-count {
  color: var(--muted);
  font-size: 11px;
}

.fail-count.hasError {
  color: var(--orange);
}

.eta-cell {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--muted);
  font-size: 8px;
}

.updated-cell {
  color: var(--muted);
  font-size: 8px;
}

.row-link {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border: 0;
  border-radius: 4px;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.row-link:hover {
  color: var(--ink);
  background: var(--paper-deep);
}

.task-panel :deep(.el-table__row) {
  cursor: pointer;
}

.pagination-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
}

.pagination-row > span {
  color: var(--muted);
  font-size: 9px;
}

@media (max-width: 1200px) {
  .task-overview {
    grid-template-columns: 1.2fr repeat(3, 0.8fr);
  }

  .overview-chart {
    display: none;
  }

  .status-tabs button {
    padding: 0 7px;
  }
}

@media (max-width: 900px) {
  .task-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .status-tabs {
    overflow-x: auto;
  }

  .agent-filter,
  .task-search {
    width: 100%;
  }
}

@media (max-width: 760px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .header-actions {
    width: 100%;
    justify-content: flex-end;
  }

  .task-overview {
    grid-template-columns: 1fr 1fr;
  }

  .overview-primary {
    grid-column: span 2;
  }

  .overview-metric {
    border-bottom: 1px solid var(--line);
  }

  .overview-metric:nth-child(3) {
    border-right: 0;
  }

  .overview-metric:nth-child(4) {
    grid-column: span 2;
    border-right: 0;
    border-bottom: 0;
  }
}

@media (max-width: 980px) {
  :deep(.task-retry-column),
  :deep(.task-stage-column),
  :deep(.task-updated-column) {
    display: none;
  }
}
</style>
