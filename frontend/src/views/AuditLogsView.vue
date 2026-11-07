<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { RefreshCw, ScrollText } from 'lucide-vue-next'
import { api } from '@/services/api'
import type { AuditLog } from '@/types'
import { formatDateTime } from '@/utils/format'

const logs = ref<AuditLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)

const actionLabels: Record<string, string> = {
  'auth.login': '登录平台',
  'task.create': '创建任务',
  'task.cancel': '取消任务',
  'agent.config.update': '更新 Agent 配置',
  'agent.disable': '禁用 Agent',
  'wechat.login.request': '请求微信登录',
  'user.create': '创建用户',
  'user.update': '更新用户',
  'user.delete': '删除用户',
  'settings.update': '更新个人设置',
  'articles.export': '导出文章',
}

async function load(nextPage = page.value) {
  page.value = nextPage
  loading.value = true
  try {
    const result = await api.auditLogs.list(page.value, pageSize)
    logs.value = result.items
    total.value = result.total
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '审计日志加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => load(1))
</script>

<template>
  <div class="page-shell">
    <header class="page-header">
      <div>
        <p class="eyebrow">PLATFORM AUDIT</p>
        <h1 class="page-title">操作审计</h1>
        <p class="page-lead">记录登录、任务、Agent、用户、设置和导出等关键平台操作。</p>
      </div>
      <el-button :loading="loading" @click="load()">
        <RefreshCw v-if="!loading" :size="15" />
        刷新
      </el-button>
    </header>

    <section class="audit-summary panel">
      <span class="summary-icon"><ScrollText :size="20" /></span>
      <div>
        <span class="panel-kicker">AUDIT EVENTS</span>
        <strong class="mono">{{ total }}</strong>
        <p>条平台操作记录</p>
      </div>
    </section>

    <section class="panel table-panel">
      <el-table v-loading="loading" :data="logs" row-key="id">
        <el-table-column label="操作" min-width="170">
          <template #default="{ row }">
            <div class="action-cell">
              <strong>{{ actionLabels[row.action] ?? row.action }}</strong>
              <small class="mono">{{ row.action }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="资源" min-width="170">
          <template #default="{ row }">
            <div class="resource-cell">
              <strong>{{ row.resourceType }}</strong>
              <small class="mono">{{ row.resourceId || '—' }}</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="详情" min-width="260">
          <template #default="{ row }">
            <code>{{ Object.keys(row.details).length ? JSON.stringify(row.details) : '—' }}</code>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="170">
          <template #default="{ row }">
            <span class="muted">{{ formatDateTime(row.createdAt) }}</span>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-row">
        <span>共 {{ total }} 条记录</span>
        <el-pagination
          size="small"
          background
          layout="prev, pager, next"
          :total="total"
          :page-size="pageSize"
          :current-page="page"
          @current-change="load"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 24px;
}

.audit-summary {
  display: flex;
  min-height: 104px;
  align-items: center;
  gap: 16px;
  margin-bottom: 18px;
  padding: 20px;
}

.summary-icon {
  display: grid;
  width: 46px;
  height: 46px;
  place-items: center;
  border-radius: 7px;
  color: var(--green-dark);
  background: rgba(11, 151, 103, 0.08);
}

.audit-summary > div {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  column-gap: 8px;
}

.audit-summary .panel-kicker {
  grid-column: span 2;
}

.audit-summary strong {
  margin-top: 5px;
  font-size: 27px;
}

.audit-summary p {
  margin: 0;
  color: var(--muted);
  font-size: 9px;
}

.table-panel {
  overflow: hidden;
}

.action-cell,
.resource-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.action-cell strong,
.resource-cell strong {
  font-size: 10px;
}

.action-cell small,
.resource-cell small,
.muted {
  overflow: hidden;
  color: var(--muted);
  font-size: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

code {
  display: block;
  overflow: hidden;
  color: var(--ink-2);
  font-family: "Bricolage Grotesque Variable", monospace;
  font-size: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
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

@media (max-width: 760px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
