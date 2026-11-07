<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Bot,
  CalendarClock,
  Clock3,
  ExternalLink,
  FileStack,
  RefreshCw,
  ShieldAlert,
  TimerReset,
  XCircle,
} from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import StatusPill from '@/components/common/StatusPill.vue'
import { useTaskStore } from '@/stores/tasks'
import { formatDateTime } from '@/utils/format'

const taskStore = useTaskStore()
const router = useRouter()
const cancelling = ref(false)
const task = computed(() => taskStore.selectedTask)
const terminal = computed(
  () =>
    task.value?.status === 'succeeded' ||
    task.value?.status === 'failed' ||
    task.value?.status === 'canceled',
)
const eventLabels: Record<string, string> = {
  created: '任务创建',
  assigned: '任务派发',
  delivery_failed: '派发失败',
  running: '开始执行',
  progress: '进度更新',
  articles_received: '文章入库',
  lease_expired: '租约过期',
  succeeded: '执行成功',
  failed: '执行失败',
  canceled: '任务取消',
}

function close() {
  taskStore.drawerOpen = false
}

function openAgent() {
  if (!task.value?.agentId) return
  close()
  router.push({ name: 'agent-detail', params: { id: task.value.agentId } })
}

async function refresh() {
  try {
    await taskStore.refreshSelected()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '任务详情刷新失败')
  }
}

async function cancel() {
  if (!task.value) return
  try {
    await ElMessageBox.confirm('取消后 Agent 会停止当前 UI 操作。', '取消任务', {
      type: 'warning',
      confirmButtonText: '确认取消',
      cancelButtonText: '返回',
    })
  } catch {
    return
  }
  cancelling.value = true
  try {
    await taskStore.cancelTask(task.value)
    ElMessage.success('任务已取消')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '取消任务失败')
  } finally {
    cancelling.value = false
  }
}
</script>

<template>
  <el-drawer
    :model-value="taskStore.drawerOpen"
    append-to-body
    size="min(560px, 94vw)"
    :show-close="false"
    @update:model-value="taskStore.drawerOpen = $event"
  >
    <template #header>
      <div class="drawer-head">
        <div>
          <p class="eyebrow">PIPELINE DETAIL</p>
          <h2>{{ task?.name ?? '任务详情' }}</h2>
        </div>
        <div class="head-actions">
          <button class="icon-button" title="刷新任务" @click="refresh">
            <RefreshCw :size="15" />
          </button>
          <button class="icon-button" title="关闭" @click="close">×</button>
        </div>
      </div>
    </template>

    <div v-if="task" class="detail-body">
      <section class="status-hero">
        <div class="status-line">
          <StatusPill :status="task.status" />
          <span class="mono">{{ task.id }}</span>
        </div>
        <div class="progress-number">
          <strong class="mono">{{ task.progress }}</strong>
          <span>%</span>
        </div>
        <div class="hero-track"><i :style="{ width: `${task.progress}%` }" /></div>
        <p class="stage-message">{{ task.progressMessage || task.stage || '等待 Agent 上报进度' }}</p>
        <div class="hero-stats">
          <div>
            <strong class="mono">{{ task.collected }}</strong>
            <span>已处理</span>
          </div>
          <div>
            <strong class="mono">{{ task.total }}</strong>
            <span>计划数量</span>
          </div>
          <div>
            <strong class="mono">{{ task.attempts }}/{{ task.maxAttempts }}</strong>
            <span>尝试次数</span>
          </div>
        </div>
      </section>

      <section v-if="task.lastError" class="notice-section error">
        <ShieldAlert :size="17" />
        <div>
          <strong>执行失败</strong>
          <small v-if="task.errorCode" class="mono">{{ task.errorCode }}</small>
          <p>{{ task.lastError }}</p>
        </div>
      </section>
      <section v-else-if="task.status === 'succeeded' && !task.complete" class="notice-section">
        <ShieldAlert :size="17" />
        <div>
          <strong>列表可能不完整</strong>
          <p>Agent 已完成当前采集，但未确认已到达列表底部，请结合事件与诊断检查。</p>
        </div>
      </section>

      <section class="detail-section">
        <h3>任务信息</h3>
        <div class="info-grid">
          <div>
            <span><Bot :size="13" />执行 Agent</span>
            <button v-if="task.agentId" @click="openAgent">
              {{ task.agentName }} <ExternalLink :size="11" />
            </button>
            <strong v-else>等待自动调度</strong>
          </div>
          <div>
            <span><FileStack :size="13" />采集类型</span>
            <strong>公众号文章采集</strong>
          </div>
          <div>
            <span><CalendarClock :size="13" />优先级</span>
            <strong class="mono">{{ task.priority }}</strong>
          </div>
          <div>
            <span><Clock3 :size="13" />创建时间</span>
            <strong>{{ formatDateTime(task.createdAt) }}</strong>
          </div>
          <div>
            <span><TimerReset :size="13" />开始时间</span>
            <strong>{{ task.startedAt ? formatDateTime(task.startedAt) : '尚未开始' }}</strong>
          </div>
          <div>
            <span><XCircle :size="13" />结束时间</span>
            <strong>{{ task.finishedAt ? formatDateTime(task.finishedAt) : '尚未结束' }}</strong>
          </div>
        </div>
      </section>

      <section class="detail-section">
        <h3>采集参数</h3>
        <div class="source-card">
          <span class="source-index mono">WECHAT</span>
          <div>
            <strong>{{ task.accountName }}</strong>
            <small>最多 {{ task.maxArticles }} 篇 · {{ task.includeContent ? '抓取正文' : '仅元数据' }}</small>
          </div>
          <i />
        </div>
        <div v-if="task.since || task.until" class="date-window">
          <span>发布时间</span>
          <strong class="mono">
            {{ task.since ? formatDateTime(task.since) : '不限' }}
            →
            {{ task.until ? formatDateTime(task.until) : '不限' }}
          </strong>
        </div>
      </section>

      <section class="timeline-section">
        <div class="timeline-head">
          <h3>任务事件</h3>
          <span>{{ taskStore.events.length }} 条</span>
        </div>
        <div v-loading="taskStore.eventsLoading" class="timeline">
          <div
            v-for="event in taskStore.events"
            :key="event.id"
            class="timeline-item"
            :class="event.type"
          >
            <i />
            <span>
              <strong>{{ eventLabels[event.type] ?? event.type }}</strong>
              <p>{{ event.message || '无附加消息' }}</p>
              <small>{{ formatDateTime(event.createdAt) }}</small>
            </span>
          </div>
          <div v-if="!taskStore.eventsLoading && !taskStore.events.length" class="timeline-empty">
            暂无任务事件
          </div>
        </div>
      </section>

      <section v-if="!terminal" class="actions-section">
        <el-button type="danger" plain :loading="cancelling" @click="cancel">
          <XCircle :size="14" />
          取消任务
        </el-button>
      </section>
    </div>
  </el-drawer>
</template>

<style scoped>
.drawer-head {
  display: flex;
  width: 100%;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.drawer-head h2 {
  max-width: 390px;
  margin: 0;
  overflow: hidden;
  font-size: 20px;
  font-weight: 680;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.head-actions {
  display: flex;
  gap: 4px;
}

.head-actions .icon-button:last-child {
  font-size: 20px;
}

.detail-body {
  padding: 20px 24px 30px;
}

.status-hero {
  padding: 18px;
  border-radius: 7px;
  color: #f2f5ef;
  background: var(--ink);
}

.status-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.status-line > span:last-child {
  overflow: hidden;
  color: rgba(255, 255, 255, 0.28);
  font-size: 8px;
  text-overflow: ellipsis;
}

.progress-number {
  display: flex;
  align-items: baseline;
  gap: 3px;
  margin: 16px 0 10px;
}

.progress-number strong {
  color: var(--acid);
  font-size: 42px;
  font-weight: 500;
  letter-spacing: -0.06em;
  line-height: 1;
}

.progress-number span {
  color: rgba(255, 255, 255, 0.38);
  font-size: 14px;
}

.hero-track {
  height: 4px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.1);
}

.hero-track i {
  display: block;
  height: 100%;
  background: var(--acid);
  transition: width 0.4s ease;
}

.stage-message {
  margin: 9px 0 0;
  color: rgba(255, 255, 255, 0.45);
  font-size: 9px;
}

.hero-stats {
  display: grid;
  gap: 1px;
  margin-top: 18px;
  background: rgba(255, 255, 255, 0.1);
  grid-template-columns: repeat(3, 1fr);
}

.hero-stats div {
  display: flex;
  padding: 10px 11px;
  flex-direction: column;
  gap: 3px;
  background: var(--ink);
}

.hero-stats strong {
  font-size: 16px;
  font-weight: 520;
}

.hero-stats span {
  color: rgba(255, 255, 255, 0.36);
  font-size: 8px;
}

.detail-section,
.timeline-section {
  padding: 22px 0;
  border-bottom: 1px solid var(--line);
}

.detail-section h3,
.timeline-head h3 {
  margin: 0 0 14px;
  font-size: 11px;
  font-weight: 650;
}

.info-grid {
  display: grid;
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--line);
  grid-template-columns: 1fr 1fr;
}

.info-grid > div {
  display: flex;
  min-height: 68px;
  justify-content: center;
  padding: 11px;
  flex-direction: column;
  gap: 7px;
  background: var(--surface);
}

.info-grid span {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--muted);
  font-size: 8px;
}

.info-grid strong,
.info-grid button {
  overflow: hidden;
  padding: 0;
  border: 0;
  color: var(--ink);
  background: transparent;
  font-size: 10px;
  font-weight: 600;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.info-grid button {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--green-dark);
  cursor: pointer;
}

.source-card {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 12px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: #fff;
}

.source-index {
  display: grid;
  width: 47px;
  height: 30px;
  place-items: center;
  border-radius: 4px;
  color: var(--green-dark);
  background: rgba(201, 243, 106, 0.3);
  font-size: 7px;
  font-weight: 700;
  letter-spacing: 0.06em;
}

.source-card div {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.source-card strong {
  font-size: 10px;
}

.source-card small {
  color: var(--muted);
  font-size: 8px;
}

.source-card i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--green);
}

.date-window {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 8px;
  padding: 9px 11px;
  border-radius: 5px;
  color: var(--muted);
  background: var(--paper);
  font-size: 8px;
}

.date-window strong {
  color: var(--ink-2);
  font-weight: 550;
}

.notice-section {
  display: flex;
  gap: 10px;
  margin-top: 18px;
  padding: 12px;
  border: 1px solid rgba(213, 141, 40, 0.2);
  border-radius: 6px;
  color: #9a6216;
  background: rgba(213, 141, 40, 0.07);
}

.notice-section.error {
  border-color: rgba(221, 85, 63, 0.2);
  color: #b23f2d;
  background: rgba(221, 85, 63, 0.07);
}

.notice-section div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.notice-section strong {
  font-size: 9px;
}

.notice-section p {
  margin: 0;
  font-size: 9px;
  line-height: 1.5;
}

.timeline-section {
  border-bottom: 0;
}

.timeline-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.timeline-head span {
  color: var(--muted);
  font-size: 8px;
}

.timeline {
  min-height: 52px;
  padding-left: 4px;
}

.timeline-item {
  position: relative;
  display: flex;
  min-height: 62px;
  gap: 12px;
  padding-left: 12px;
  border-left: 1px solid var(--line);
}

.timeline-item:last-child {
  border-left-color: transparent;
}

.timeline-item > i {
  position: absolute;
  top: 2px;
  left: -4px;
  width: 7px;
  height: 7px;
  border: 1px solid var(--green);
  border-radius: 50%;
  background: var(--green);
  box-shadow: 0 0 0 3px var(--surface);
}

.timeline-item.failed > i,
.timeline-item.delivery_failed > i,
.timeline-item.lease_expired > i {
  border-color: var(--orange);
  background: var(--orange);
}

.timeline-item span {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.timeline-item strong {
  font-size: 9px;
  font-weight: 650;
}

.timeline-item p {
  margin: 0;
  color: var(--ink-2);
  font-size: 9px;
  line-height: 1.45;
}

.timeline-item small {
  color: var(--muted);
  font-size: 8px;
}

.timeline-empty {
  padding: 20px 0;
  color: var(--muted);
  font-size: 9px;
}

.actions-section {
  display: flex;
  justify-content: flex-end;
  padding-top: 10px;
}
</style>
