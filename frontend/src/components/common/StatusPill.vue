<script setup lang="ts">
import { computed } from 'vue'
import type { AgentStatus, ArticleStatus, TaskStatus } from '@/types'

const props = defineProps<{
  status: AgentStatus | TaskStatus | ArticleStatus
  label?: string
}>()

const statusMap: Record<string, { label: string; tone: string }> = {
  online: { label: '在线', tone: 'success' },
  busy: { label: '运行中', tone: 'success' },
  offline: { label: '离线', tone: 'neutral' },
  disabled: { label: '已禁用', tone: 'neutral' },
  error: { label: '异常', tone: 'danger' },
  pending: { label: '待执行', tone: 'neutral' },
  assigned: { label: '已分配', tone: 'warning' },
  running: { label: '采集中', tone: 'success' },
  succeeded: { label: '已完成', tone: 'success' },
  failed: { label: '失败', tone: 'danger' },
  canceled: { label: '已取消', tone: 'neutral' },
  complete: { label: '内容完整', tone: 'success' },
  metadata_only: { label: '仅元数据', tone: 'warning' },
  content_missing: { label: '正文缺失', tone: 'warning' },
}

const config = computed(() => statusMap[props.status] ?? { label: props.status, tone: 'neutral' })
</script>

<template>
  <span class="status-pill" :class="`tone-${config.tone}`">
    <i />
    {{ label ?? config.label }}
  </span>
</template>

<style scoped>
.status-pill {
  display: inline-flex;
  height: 24px;
  align-items: center;
  gap: 6px;
  padding: 0 9px;
  border: 1px solid transparent;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 650;
  line-height: 1;
  white-space: nowrap;
}

.status-pill i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}

.tone-success {
  border-color: rgba(11, 151, 103, 0.19);
  color: #08714f;
  background: rgba(11, 151, 103, 0.08);
}

.tone-warning {
  border-color: rgba(213, 141, 40, 0.2);
  color: #9a6216;
  background: rgba(213, 141, 40, 0.09);
}

.tone-danger {
  border-color: rgba(221, 85, 63, 0.18);
  color: #b23f2d;
  background: rgba(221, 85, 63, 0.08);
}

.tone-neutral {
  border-color: var(--line);
  color: #737a73;
  background: rgba(21, 34, 27, 0.035);
}
</style>
