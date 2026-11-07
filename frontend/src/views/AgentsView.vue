<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowUpRight,
  RadioTower,
  RefreshCw,
  Search,
  Server,
  Wifi,
} from 'lucide-vue-next'
import { useAgentStore } from '@/stores/agents'
import type { Agent, AgentStatus } from '@/types'
import { formatRelative } from '@/utils/format'
import AgentAvatar from '@/components/common/AgentAvatar.vue'
import StatusPill from '@/components/common/StatusPill.vue'
import { usePolling } from '@/composables/usePolling'

const router = useRouter()
const agents = useAgentStore()
const keyword = ref('')
const activeStatus = ref<AgentStatus | 'all'>('all')

const filters: Array<{ value: AgentStatus | 'all'; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'online', label: '在线' },
  { value: 'busy', label: '运行中' },
  { value: 'error', label: '异常' },
  { value: 'offline', label: '离线' },
  { value: 'disabled', label: '已禁用' },
]

const counts = computed(() => ({
  all: agents.items.length,
  online: agents.items.filter((agent) => agent.status === 'online').length,
  busy: agents.items.filter((agent) => agent.status === 'busy').length,
  error: agents.items.filter((agent) => agent.status === 'error').length,
  offline: agents.items.filter((agent) => agent.status === 'offline').length,
  disabled: agents.items.filter((agent) => agent.status === 'disabled').length,
}))

const availability = computed(() =>
  counts.value.all
    ? (((counts.value.online + counts.value.busy) / counts.value.all) * 100).toFixed(1)
    : '--',
)

async function refresh() {
  await agents.fetchAgents({ page: 1, keyword: keyword.value, status: activeStatus.value })
}

async function selectStatus(status: AgentStatus | 'all') {
  activeStatus.value = status
  await agents.fetchAgents({ page: 1, status })
}

function search() {
  agents.fetchAgents({ page: 1, keyword: keyword.value })
}

function openAgent(id: string) {
  router.push({ name: 'agent-detail', params: { id } })
}

function openAgentRow(row: Agent) {
  openAgent(row.id)
}

function openTasks(id: string) {
  router.push({ name: 'tasks', query: { agentId: id } })
}

onMounted(async () => {
  await refresh()
})

usePolling(() => agents.fetchAgents())
</script>

<template>
  <div class="page-shell agents-page">
    <header class="page-header">
      <div>
        <p class="eyebrow">WINDOWS AGENT FLEET</p>
        <h1 class="page-title">Agent 网络</h1>
        <p class="page-lead">监控每台 Windows 机器的在线状态、微信账号、负载与任务执行情况。</p>
      </div>
      <div class="header-actions">
        <span class="last-sync">状态来自 REST 实时查询</span>
        <el-button :loading="agents.loading" @click="refresh">
          <RefreshCw v-if="!agents.loading" :size="15" />
          刷新状态
        </el-button>
      </div>
    </header>

    <section class="fleet-summary">
      <div class="summary-main">
        <div class="summary-icon"><RadioTower :size="19" /></div>
        <div>
          <span class="panel-kicker">FLEET AVAILABILITY</span>
          <strong><span class="mono">{{ counts.online + counts.busy }}</span> / {{ counts.all }}</strong>
          <p>个节点当前可接受调度</p>
        </div>
        <span class="availability mono">{{ availability }}%</span>
      </div>
      <div class="summary-cell">
        <span><i class="online" />在线</span>
        <strong class="mono">{{ counts.online }}</strong>
      </div>
      <div class="summary-cell">
        <span><i class="busy" />运行中</span>
        <strong class="mono">{{ counts.busy }}</strong>
      </div>
      <div class="summary-cell">
        <span><i class="error" />异常</span>
        <strong class="mono">{{ counts.error }}</strong>
      </div>
      <div class="summary-cell">
        <span><i class="offline" />离线</span>
        <strong class="mono">{{ counts.offline }}</strong>
      </div>
      <div class="summary-health">
        <span class="panel-kicker">DISABLED NODES</span>
        <strong class="mono">{{ counts.disabled }}</strong>
        <small>管理员已禁用</small>
      </div>
    </section>

    <section class="agent-table-panel panel">
      <div class="table-toolbar">
        <div class="status-tabs">
          <button
            v-for="filter in filters"
            :key="filter.value"
            :class="{ active: activeStatus === filter.value }"
            @click="selectStatus(filter.value)"
          >
            {{ filter.label }}
            <span class="mono">{{ counts[filter.value] }}</span>
          </button>
        </div>
        <div class="toolbar-spacer" />
        <el-input
          v-model="keyword"
          class="agent-search"
          placeholder="搜索名称、编号或微信"
          clearable
          @keyup.enter="search"
          @clear="search"
        >
          <template #prefix><Search :size="14" /></template>
        </el-input>
      </div>

      <el-table
        v-loading="agents.loading"
        :data="agents.items"
        class="agent-table"
        row-key="id"
        @row-click="openAgentRow"
      >
        <el-table-column label="Agent" min-width="220">
          <template #default="{ row }">
            <div class="agent-identity">
              <span class="status-indicator" :class="row.status" />
              <AgentAvatar :name="row.name" :size="37" />
              <span>
                <strong>{{ row.name }}</strong>
                <small class="mono">{{ row.code }} · {{ row.hostname }}</small>
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }"><StatusPill :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="微信账号" min-width="180">
          <template #default="{ row }">
            <div class="wechat-cell">
              <strong>{{ row.wechat.nickname }}</strong>
              <span>
                <i :class="row.wechat.loginStatus" />
                {{ row.wechat.loginStatus === 'logged_in' ? '已登录' : '未登录' }}
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          label="主机"
          width="130"
          class-name="agent-host-column"
          label-class-name="agent-host-column"
        >
          <template #default="{ row }">
            <span class="location-cell"><Server :size="12" />{{ row.hostname || '未知' }}</span>
          </template>
        </el-table-column>
        <el-table-column
          label="并发槽位"
          width="110"
          align="center"
          class-name="agent-slots-column"
          label-class-name="agent-slots-column"
        >
          <template #default="{ row }">
            <span class="mono">{{ row.activeTasks }} / {{ row.config.maxConcurrency }}</span>
          </template>
        </el-table-column>
        <el-table-column label="当前任务" width="100" align="center">
          <template #default="{ row }">
            <button class="task-count" :disabled="!row.activeTasks" @click.stop="openTasks(row.id)">
              <span class="mono">{{ row.activeTasks }}</span>
              <small>运行中</small>
            </button>
          </template>
        </el-table-column>
        <el-table-column
          label="心跳"
          width="115"
          class-name="agent-heartbeat-column"
          label-class-name="agent-heartbeat-column"
        >
          <template #default="{ row }">
            <div class="heartbeat-cell">
              <Wifi :size="12" />
              <span>{{ formatRelative(row.lastHeartbeat) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="" width="52" align="right">
          <template #default="{ row }">
            <button class="row-link" title="查看详情" @click.stop="openAgent(row.id)">
              <ArrowUpRight :size="15" />
            </button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <footer class="table-footer">
      <span>共 {{ agents.total }} 个 Agent 节点</span>
      <span class="mono">FLEET ID / WECHAT-AGENT-SHA</span>
    </footer>
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
  align-items: center;
  gap: 12px;
}

.last-sync {
  color: var(--muted);
  font-size: 9px;
}

.fleet-summary {
  display: grid;
  min-height: 112px;
  margin-bottom: 18px;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--surface);
  grid-template-columns: minmax(260px, 1.5fr) repeat(4, 0.65fr) minmax(180px, 0.85fr);
  animation: rise-in 0.45s 0.05s ease both;
}

.summary-main {
  display: flex;
  align-items: center;
  gap: 15px;
  padding: 18px 20px;
  border-right: 1px solid var(--line);
  color: #f4f6f1;
  background: var(--ink);
}

.summary-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 38px;
  place-items: center;
  border: 1px solid rgba(201, 243, 106, 0.24);
  border-radius: 6px;
  color: var(--acid);
  background: rgba(201, 243, 106, 0.06);
}

.summary-main > div:nth-child(2) {
  display: flex;
  flex-direction: column;
}

.summary-main .panel-kicker {
  color: rgba(255, 255, 255, 0.32);
}

.summary-main strong {
  margin-top: 4px;
  font-size: 26px;
  font-weight: 520;
  line-height: 1;
}

.summary-main strong .mono {
  color: var(--acid);
}

.summary-main p {
  margin: 5px 0 0;
  color: rgba(255, 255, 255, 0.43);
  font-size: 9px;
}

.availability {
  margin-left: auto;
  color: var(--acid);
  font-size: 12px;
}

.summary-cell {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 15px;
  border-right: 1px solid var(--line);
  flex-direction: column;
}

.summary-cell span {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  font-size: 9px;
}

.summary-cell i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--green);
}

.summary-cell i.busy {
  background: var(--acid);
}

.summary-cell i.error {
  background: var(--amber);
}

.summary-cell i.offline {
  background: #8d938e;
}

.summary-cell strong {
  font-size: 22px;
  font-weight: 550;
}

.summary-health {
  display: flex;
  justify-content: center;
  padding: 17px;
  flex-direction: column;
}

.summary-health strong {
  margin: 8px 0 4px;
  color: var(--ink);
  font-size: 22px;
  font-weight: 550;
}

.summary-health small {
  color: var(--muted);
  font-size: 8px;
}

.agent-table-panel {
  overflow: hidden;
  animation: rise-in 0.45s 0.1s ease both;
}

.table-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 62px;
  padding: 11px 16px;
  border-bottom: 1px solid var(--line);
}

.status-tabs {
  display: flex;
  align-items: center;
  gap: 3px;
}

.status-tabs button {
  display: flex;
  height: 30px;
  align-items: center;
  gap: 5px;
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

.status-tabs span {
  color: #9ca099;
  font-size: 8px;
}

.agent-search {
  width: 240px;
}

.agent-identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.status-indicator {
  width: 3px;
  height: 28px;
  flex: 0 0 3px;
  border-radius: 2px;
  background: var(--green);
}

.status-indicator.busy {
  background: var(--acid);
}

.status-indicator.error {
  background: var(--amber);
}

.status-indicator.offline,
.status-indicator.disabled {
  background: #9b9e98;
}

.agent-identity > span:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.agent-identity strong {
  overflow: hidden;
  font-size: 12px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-identity small {
  color: var(--muted);
  font-size: 8px;
}

.wechat-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.wechat-cell strong {
  overflow: hidden;
  font-size: 11px;
  font-weight: 560;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wechat-cell span {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--muted);
  font-size: 8px;
}

.wechat-cell i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--green);
}

.wechat-cell i.expired,
.wechat-cell i.logged_out {
  background: var(--orange);
}

.location-cell,
.heartbeat-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  font-size: 9px;
}

.metrics-cell {
  display: flex;
  align-items: center;
  gap: 13px;
}

.metrics-cell span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #687169;
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 9px;
}

.task-count {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 7px;
  border: 1px solid var(--line);
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
}

.task-count:disabled {
  cursor: default;
  opacity: 0.55;
}

.task-count span {
  font-size: 12px;
  font-weight: 600;
}

.task-count small {
  color: var(--muted);
  font-size: 7px;
}

.row-link {
  display: inline-grid;
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

.agent-table :deep(.el-table__row) {
  cursor: pointer;
}

.table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 13px 4px;
  color: var(--muted);
  font-size: 9px;
}

.table-footer .mono {
  color: #9a9e97;
  font-size: 7px;
  letter-spacing: 0.1em;
}

@media (max-width: 1250px) {
  .fleet-summary {
    grid-template-columns: minmax(240px, 1.5fr) repeat(4, 0.6fr);
  }

  .summary-health {
    display: none;
  }
}

@media (max-width: 760px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .header-actions {
    width: 100%;
  }

  .last-sync {
    display: none;
  }

  .header-actions .el-button {
    margin-left: auto;
  }

  .fleet-summary {
    grid-template-columns: 1fr 1fr;
  }

  .summary-main {
    grid-column: span 2;
    border-right: 0;
    border-bottom: 1px solid var(--line);
  }

  .summary-cell {
    border-bottom: 1px solid var(--line);
  }

  .summary-cell:nth-child(3),
  .summary-cell:nth-child(5) {
    border-right: 0;
  }

  .table-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .status-tabs {
    overflow-x: auto;
  }

  .agent-search {
    width: 100%;
  }
}

@media (max-width: 980px) {
  :deep(.agent-host-column),
  :deep(.agent-slots-column),
  :deep(.agent-heartbeat-column) {
    display: none;
  }
}
</style>
