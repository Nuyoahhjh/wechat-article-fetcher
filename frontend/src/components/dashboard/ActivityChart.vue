<script setup lang="ts">
import { computed } from 'vue'
import type { TrendPoint } from '@/types'

const props = defineProps<{
  trends: TrendPoint[]
}>()

const hasData = computed(() =>
  props.trends.some((point) => point.tasks > 0 || point.articles > 0),
)
const maxValue = computed(() =>
  Math.max(1, ...props.trends.map((point) => Math.max(point.tasks, point.articles))),
)
const totalArticles = computed(() =>
  props.trends.reduce((sum, point) => sum + point.articles, 0),
)
const totalTasks = computed(() =>
  props.trends.reduce((sum, point) => sum + point.tasks, 0),
)
const change = computed(() => {
  if (props.trends.length < 2) return null
  const current = props.trends[props.trends.length - 1]?.articles ?? 0
  const previous = props.trends[props.trends.length - 2]?.articles ?? 0
  if (previous === 0) return current > 0 ? null : 0
  return ((current - previous) / previous) * 100
})
const labels = computed(() => {
  if (!props.trends.length) return []
  const indexes = [0, 3, 7, 10, props.trends.length - 1]
  return [...new Set(indexes)].map((index) => props.trends[index]?.date.slice(5))
})
</script>

<template>
  <section class="activity-panel panel">
    <div class="panel-head chart-head">
      <div>
        <span class="panel-kicker">14 DAY THROUGHPUT</span>
        <h2 class="panel-title">采集趋势</h2>
      </div>
      <div class="chart-legend">
        <span><i class="article" /> 文章</span>
        <span><i class="task" /> 任务</span>
      </div>
    </div>
    <div class="chart-summary">
      <div>
        <strong class="mono">{{ totalArticles.toLocaleString() }}</strong>
        <span>14 天文章</span>
      </div>
      <div>
        <strong class="mono">{{ totalTasks }}</strong>
        <span>14 天任务</span>
      </div>
      <div class="trend">
        <span class="delta mono">{{ change === null ? '--' : `${change >= 0 ? '+' : ''}${change.toFixed(1)}%` }}</span>
        <small>较前一日</small>
      </div>
    </div>
    <div v-if="!hasData" class="chart-empty">暂无趋势数据</div>
    <div v-else class="chart-body">
      <div class="axis-labels">
        <span>{{ maxValue }}</span>
        <span>{{ Math.round(maxValue * 0.75) }}</span>
        <span>{{ Math.round(maxValue * 0.5) }}</span>
        <span>{{ Math.round(maxValue * 0.25) }}</span>
        <span>0</span>
      </div>
      <div class="plot">
        <div class="grid-lines"><i v-for="line in 5" :key="line" /></div>
        <div class="bars">
          <div v-for="point in trends" :key="point.date" class="bar-wrap">
            <span
              class="bar task"
              :style="{ height: `${(point.tasks / maxValue) * 100}%` }"
              :title="`${point.date} 任务 ${point.tasks}`"
            />
            <span
              class="bar article"
              :style="{ height: `${(point.articles / maxValue) * 100}%` }"
              :title="`${point.date} 文章 ${point.articles}`"
            />
          </div>
        </div>
        <div class="x-labels">
          <span v-for="label in labels" :key="label">{{ label }}</span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.chart-head {
  align-items: flex-start;
  padding-top: 16px;
  flex-direction: column;
}

.chart-head > div:first-child {
  display: flex;
  gap: 4px;
  flex-direction: column;
}

.chart-legend {
  position: absolute;
  top: 20px;
  right: 18px;
  display: flex;
  gap: 14px;
  color: var(--muted);
  font-size: 9px;
}

.chart-legend span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.chart-legend i {
  width: 6px;
  height: 6px;
  border-radius: 2px;
}

.chart-legend i.article {
  background: var(--green);
}

.chart-legend i.task {
  background: var(--amber);
}

.chart-summary {
  display: flex;
  align-items: flex-end;
  gap: 30px;
  padding: 18px 18px 8px;
}

.chart-summary > div:not(.trend) {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.chart-summary strong {
  font-size: 26px;
  font-weight: 560;
  letter-spacing: -0.04em;
  line-height: 1;
}

.chart-summary span,
.chart-summary small {
  color: var(--muted);
  font-size: 9px;
}

.trend {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
  padding-bottom: 3px;
}

.delta {
  color: var(--green-dark);
  font-size: 10px;
  font-weight: 650;
}

.chart-empty {
  display: grid;
  min-height: 210px;
  place-items: center;
  color: var(--muted);
  font-size: 10px;
}

.chart-body {
  display: flex;
  height: 175px;
  padding: 10px 18px 16px;
}

.axis-labels {
  display: flex;
  width: 27px;
  padding: 0 0 24px;
  justify-content: space-between;
  flex-direction: column;
  color: #9b9f98;
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 7px;
}

.plot {
  position: relative;
  min-width: 0;
  flex: 1;
}

.grid-lines {
  position: absolute;
  inset: 0 0 24px;
  display: flex;
  justify-content: space-between;
  flex-direction: column;
}

.grid-lines i {
  width: 100%;
  height: 1px;
  background: rgba(21, 34, 27, 0.065);
}

.bars {
  position: absolute;
  inset: 0 0 24px;
  display: flex;
  align-items: flex-end;
  gap: 7px;
}

.bar-wrap {
  display: flex;
  height: 100%;
  flex: 1;
  align-items: flex-end;
  justify-content: center;
  gap: 2px;
}

.bar {
  width: 42%;
  min-height: 2px;
  border-radius: 2px 2px 0 0;
  transition: opacity 0.18s ease;
}

.bar.article {
  background: var(--green);
  opacity: 0.72;
}

.bar.task {
  background: var(--amber);
  opacity: 0.68;
}

.bar:hover {
  opacity: 1;
}

.x-labels {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  justify-content: space-between;
  color: #9b9f98;
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 7px;
}

.panel {
  overflow: hidden;
}

@media (max-width: 760px) {
  .axis-labels,
  .x-labels {
    font-size: 8px;
  }

  .chart-legend {
    font-size: 9px;
  }
}
</style>
