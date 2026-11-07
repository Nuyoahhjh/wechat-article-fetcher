<script setup lang="ts">
import { Inbox } from 'lucide-vue-next'

defineProps<{
  title?: string
  description?: string
}>()
</script>

<template>
  <div class="empty-state">
    <span class="empty-icon"><Inbox :size="20" :stroke-width="1.7" /></span>
    <strong>{{ title ?? '暂无数据' }}</strong>
    <p>{{ description ?? '调整筛选条件后再试一次。' }}</p>
  </div>
</template>

<style scoped>
.empty-state {
  display: flex;
  min-height: 250px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 32px;
  text-align: center;
}

.empty-icon {
  display: grid;
  width: 44px;
  height: 44px;
  margin-bottom: 14px;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 8px;
  color: var(--muted);
  background: var(--paper);
}

strong {
  font-size: 14px;
}

p {
  margin: 7px 0 0;
  color: var(--muted);
  font-size: 12px;
}
</style>
