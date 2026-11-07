<script setup lang="ts">
import { Camera, RefreshCw } from 'lucide-vue-next'
import type { Agent, ScreenSnapshot } from '@/types'
import { formatDateTime } from '@/utils/format'

defineProps<{
  agent: Agent
  screen: ScreenSnapshot | null
  loading?: boolean
}>()

const emit = defineEmits<{
  refresh: []
}>()
</script>

<template>
  <section class="screen-panel panel">
    <div class="panel-head">
      <div>
        <span class="panel-kicker">LATEST SCREEN</span>
        <h2 class="panel-title"><Camera :size="15" /> 最近截图</h2>
      </div>
      <div class="screen-actions">
        <span class="capture-time">
          <i :class="{ offline: !screen?.online }" />
          {{ screen?.online ? '截图可用' : '暂无截图' }}
          <span v-if="screen?.capturedAt" class="mono">{{ formatDateTime(screen.capturedAt) }}</span>
        </span>
        <button class="icon-button" title="刷新截图" @click="emit('refresh')">
          <RefreshCw :size="14" :class="{ spinning: loading }" />
        </button>
      </div>
    </div>

    <div class="screen-stage">
      <img v-if="screen?.imageUrl" :src="screen.imageUrl" alt="Agent 最近屏幕截图" />
      <div v-else class="screen-empty">
        <Camera :size="30" :stroke-width="1.3" />
        <strong>暂无实时画面</strong>
        <span>{{ screen?.message || 'Agent 上传截图后会显示在这里' }}</span>
      </div>
      <div v-if="screen?.imageUrl && screen.width && screen.height" class="screen-meta mono">
        {{ screen.width }} × {{ screen.height }}
      </div>
    </div>
  </section>
</template>

<style scoped>
.screen-panel {
  overflow: hidden;
}

.panel-title {
  gap: 7px;
}

.screen-actions {
  display: flex;
  align-items: center;
  gap: 7px;
}

.capture-time {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-right: 4px;
  color: var(--muted);
  font-size: 8px;
}

.capture-time > i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--green);
}

.capture-time > i.offline {
  background: var(--orange);
}

.capture-time > span {
  color: #999e97;
}

.screen-stage {
  position: relative;
  display: grid;
  min-height: 280px;
  overflow: hidden;
  place-items: center;
  background: #1d2621;
  aspect-ratio: 16 / 9;
}

.screen-stage > img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.screen-empty {
  display: flex;
  align-items: center;
  padding: 30px;
  flex-direction: column;
  color: rgba(255, 255, 255, 0.42);
  text-align: center;
}

.screen-empty strong {
  margin-top: 10px;
  color: rgba(255, 255, 255, 0.72);
  font-size: 11px;
}

.screen-empty span {
  margin-top: 5px;
  font-size: 8px;
}

.screen-meta {
  position: absolute;
  right: 12px;
  bottom: 12px;
  padding: 5px 7px;
  border: 1px solid rgba(255, 255, 255, 0.13);
  border-radius: 4px;
  color: rgba(255, 255, 255, 0.68);
  background: rgba(17, 24, 20, 0.72);
  font-size: 7px;
}

.spinning {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 780px) {
  .capture-time > span:last-child {
    display: none;
  }
}
</style>
