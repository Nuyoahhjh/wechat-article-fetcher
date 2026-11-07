<script setup lang="ts">
import { computed } from 'vue'
import {
  AlertTriangle,
  Info,
  ListFilter,
  XCircle,
} from 'lucide-vue-next'
import type { AgentLog } from '@/types'
import { formatTime } from '@/utils/format'

const props = defineProps<{
  logs: AgentLog[]
}>()

const visibleLogs = computed(() => props.logs.slice(0, 8))

const levelLabel = {
  debug: '调试',
  info: '信息',
  warning: '警告',
  error: '错误',
}
</script>

<template>
  <section class="log-panel panel">
    <div class="panel-head">
      <div>
        <span class="panel-kicker">OPERATION FEED</span>
        <h2 class="panel-title">操作日志</h2>
      </div>
      <button class="icon-button" title="筛选日志"><ListFilter :size="14" /></button>
    </div>
    <div class="log-list">
      <div v-for="log in visibleLogs" :key="log.id" class="log-row" :class="log.level">
        <span class="log-icon">
          <AlertTriangle v-if="log.level === 'warning'" :size="13" />
          <XCircle v-else-if="log.level === 'error'" :size="13" />
          <Info v-else :size="13" />
        </span>
        <div>
          <p>{{ log.message }}</p>
          <span>
            <em>{{ levelLabel[log.level] }}</em>
            <span class="mono">{{ formatTime(log.createdAt) }}</span>
          </span>
        </div>
      </div>
      <div v-if="!visibleLogs.length" class="log-empty">当前节点没有操作日志</div>
    </div>
    <button class="all-logs">查看全部操作记录</button>
  </section>
</template>

<style scoped>
.log-panel {
  overflow: hidden;
}

.panel-title {
  gap: 0;
}

.log-list {
  display: flex;
  flex-direction: column;
}

.log-row {
  display: flex;
  gap: 10px;
  padding: 12px 15px;
  border-bottom: 1px solid var(--line);
}

.log-icon {
  display: grid;
  width: 24px;
  height: 24px;
  flex: 0 0 24px;
  place-items: center;
  border-radius: 4px;
  color: var(--green-dark);
  background: rgba(11, 151, 103, 0.08);
}

.log-row.warning .log-icon {
  color: #9a6216;
  background: rgba(213, 141, 40, 0.1);
}

.log-row.error .log-icon {
  color: var(--danger);
  background: rgba(221, 85, 63, 0.08);
}

.log-row > div {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 5px;
}

.log-row p {
  margin: 0;
  color: var(--ink-2);
  font-size: 9px;
  line-height: 1.5;
}

.log-row div > span {
  display: flex;
  align-items: center;
  gap: 7px;
  color: #999e97;
  font-size: 7px;
}

.log-row em {
  padding: 2px 4px;
  border-radius: 3px;
  color: var(--green-dark);
  background: rgba(11, 151, 103, 0.06);
  font-style: normal;
}

.log-row.warning em {
  color: #9a6216;
  background: rgba(213, 141, 40, 0.08);
}

.log-empty {
  padding: 45px 20px;
  color: var(--muted);
  font-size: 9px;
  text-align: center;
}

.all-logs {
  width: 100%;
  padding: 11px;
  border: 0;
  color: var(--green-dark);
  background: transparent;
  font-size: 8px;
  font-weight: 600;
  cursor: pointer;
}

.all-logs:hover {
  background: var(--paper);
}
</style>
