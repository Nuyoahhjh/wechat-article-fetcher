<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  ArrowUpRight,
  Download,
  FileJson,
  FileSpreadsheet,
  FileText,
  RefreshCw,
  Search,
  SlidersHorizontal,
} from 'lucide-vue-next'
import { useArticleStore } from '@/stores/articles'
import type { ArticleStatus } from '@/types'
import { formatDateTime, formatRelative } from '@/utils/format'
import StatusPill from '@/components/common/StatusPill.vue'
import ArticlePreviewDrawer from '@/components/articles/ArticlePreviewDrawer.vue'
import { api } from '@/services/api'
import { usePolling } from '@/composables/usePolling'

const route = useRoute()
const articles = useArticleStore()
const keyword = ref('')
const activeStatus = ref<ArticleStatus | 'all'>('all')
const accountName = ref('')

const filters: Array<{ value: ArticleStatus | 'all'; label: string }> = [
  { value: 'all', label: '全部内容' },
  { value: 'complete', label: '内容完整' },
  { value: 'metadata_only', label: '仅元数据' },
  { value: 'content_missing', label: '正文缺失' },
  { value: 'failed', label: '失败' },
]

const completeCount = computed(
  () => articles.items.filter((article) => article.status === 'complete').length,
)
const metadataCount = computed(
  () => articles.items.filter((article) => article.status === 'metadata_only').length,
)
const missingCount = computed(
  () => articles.items.filter((article) => article.status === 'content_missing').length,
)

async function load() {
  await articles.fetchArticles({
    page: 1,
    keyword: keyword.value,
    status: activeStatus.value,
    accountName: accountName.value || undefined,
  })
}

async function selectStatus(status: ArticleStatus | 'all') {
  activeStatus.value = status
  await load()
}

async function handleQueryArticle() {
  if (typeof route.query.article !== 'string') return
  const article = await api.articles.get(route.query.article)
  await articles.openArticle(article)
}

async function exportAs(format: 'json' | 'csv' | 'markdown') {
  try {
    const count = await articles.exportArticles(format)
    ElMessage.success(`已导出 ${count} 篇文章`)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '导出失败')
  }
}

onMounted(async () => {
  await Promise.all([load(), articles.fetchAccounts()])
  await handleQueryArticle()
})

usePolling(() => articles.fetchArticles())
</script>

<template>
  <div class="page-shell articles-page">
    <header class="page-header">
      <div>
        <p class="eyebrow">PUBLISHING ARCHIVE</p>
        <h1 class="page-title">文章资料库</h1>
        <p class="page-lead">检索所有 Agent 已采集的公众号文章，快速核对内容、来源与采集状态。</p>
      </div>
      <div class="header-actions">
        <el-dropdown trigger="click" @command="exportAs">
          <el-button :loading="articles.exporting">
            <Download v-if="!articles.exporting" :size="15" />
            导出
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="json"><FileJson :size="14" />JSON</el-dropdown-item>
              <el-dropdown-item command="csv"><FileSpreadsheet :size="14" />CSV</el-dropdown-item>
              <el-dropdown-item command="markdown"><FileText :size="14" />Markdown</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button :loading="articles.loading" @click="load">
          <RefreshCw v-if="!articles.loading" :size="15" />
          刷新
        </el-button>
      </div>
    </header>

    <section class="archive-summary">
      <div class="archive-primary">
        <span class="panel-kicker">CONTENT INVENTORY</span>
        <div class="archive-count">
          <strong class="mono">{{ articles.total.toLocaleString() }}</strong>
          <span>篇文章</span>
        </div>
        <p>当前页涉及 {{ articles.accountOptions.length }} 个公众号</p>
        <span class="archive-signal"><i /> 真实接口数据</span>
      </div>
      <div class="archive-metric">
        <span><FileText :size="14" />本页完整</span>
        <strong class="mono">{{ completeCount }}</strong>
        <small>含正文</small>
      </div>
      <div class="archive-metric">
        <span><FileJson :size="14" />仅元数据</span>
        <strong class="mono">{{ metadataCount }}</strong>
        <small>未抓正文</small>
      </div>
      <div class="archive-metric">
        <span><FileSpreadsheet :size="14" />正文缺失</span>
        <strong class="mono">{{ missingCount }}</strong>
        <small>详情提取失败</small>
      </div>
      <div class="archive-trend">
        <span class="panel-kicker">EXPORT READY</span>
        <div class="export-copy">
          <strong>JSON / CSV / Markdown</strong>
          <small>导出遵循当前筛选条件</small>
        </div>
      </div>
    </section>

    <section class="archive-panel panel">
      <div class="archive-toolbar">
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
          v-model="accountName"
          class="account-filter"
          placeholder="全部公众号"
          clearable
          @change="load"
        >
          <template #prefix><SlidersHorizontal :size="13" /></template>
          <el-option
            v-for="account in articles.accountOptions"
            :key="account"
            :label="account"
            :value="account"
          />
        </el-select>
        <el-input
          v-model="keyword"
          class="article-search"
          placeholder="搜索标题、作者或摘要"
          clearable
          @keyup.enter="load"
          @clear="load"
        >
          <template #prefix><Search :size="14" /></template>
        </el-input>
      </div>

      <el-table
        v-loading="articles.loading"
        :data="articles.items"
        row-key="id"
        @row-click="articles.openArticle"
      >
        <el-table-column label="文章内容" min-width="440">
          <template #default="{ row }">
            <div class="article-cell">
              <span class="article-thumb">
                <img v-if="row.coverUrl" :src="row.coverUrl" alt="" loading="lazy" />
                <span v-else>{{ row.accountName.slice(0, 1) }}</span>
              </span>
              <span class="article-copy">
                <span class="title-line">
                  <strong>{{ row.title }}</strong>
                </span>
                <small>{{ row.digest || row.contentText.slice(0, 100) || '暂无摘要' }}</small>
                <span class="article-byline">
                  <span>{{ row.accountName }}</span>
                  <i />
                  <span>{{ row.author }}</span>
                  <i />
                  <span class="mono">{{ row.id }}</span>
                </span>
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }"><StatusPill :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="采集来源" width="125">
          <template #default="{ row }">
            <span class="relative-cell">{{ row.source || 'wechat_desktop' }}</span>
          </template>
        </el-table-column>
        <el-table-column
          label="发布时间"
          width="140"
          class-name="article-published-column"
          label-class-name="article-published-column"
        >
          <template #default="{ row }">
            <span class="date-cell">{{ row.publishedAt ? formatDateTime(row.publishedAt) : '未知' }}</span>
          </template>
        </el-table-column>
        <el-table-column
          label="入库时间"
          width="100"
          class-name="article-ingested-column"
          label-class-name="article-ingested-column"
        >
          <template #default="{ row }">
            <span class="relative-cell">{{ formatRelative(row.collectedAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="" width="48">
          <template #default="{ row }">
            <button class="row-link" title="查看文章" @click.stop="articles.openArticle(row)">
              <ArrowUpRight :size="15" />
            </button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-row">
        <span>显示 {{ articles.items.length }} / {{ articles.total }} 篇文章</span>
        <el-pagination
          size="small"
          background
          layout="prev, pager, next"
          :total="articles.total"
          :page-size="articles.query.pageSize"
          @current-change="(page: number) => articles.fetchArticles({ page })"
        />
      </div>
    </section>

    <ArticlePreviewDrawer />
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

.archive-summary {
  display: grid;
  min-height: 122px;
  margin-bottom: 18px;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--surface);
  grid-template-columns: 1.4fr repeat(3, 0.82fr) 1fr;
  animation: rise-in 0.45s 0.05s ease both;
}

.archive-primary {
  position: relative;
  padding: 18px 20px;
  color: #f4f6f1;
  background: var(--ink);
}

.archive-primary .panel-kicker {
  color: rgba(255, 255, 255, 0.32);
}

.archive-count {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin-top: 7px;
}

.archive-count strong {
  color: var(--acid);
  font-size: 34px;
  font-weight: 500;
  letter-spacing: -0.05em;
  line-height: 1;
}

.archive-count span {
  color: rgba(255, 255, 255, 0.4);
  font-size: 9px;
}

.archive-primary p {
  margin: 6px 0 0;
  color: rgba(255, 255, 255, 0.38);
  font-size: 8px;
}

.archive-signal {
  position: absolute;
  right: 17px;
  bottom: 16px;
  display: flex;
  align-items: center;
  gap: 5px;
  color: rgba(255, 255, 255, 0.32);
  font-size: 8px;
}

.archive-signal i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--acid);
}

.archive-metric {
  display: flex;
  justify-content: center;
  padding: 18px;
  border-right: 1px solid var(--line);
  flex-direction: column;
  gap: 5px;
}

.archive-metric span {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  font-size: 8px;
}

.archive-metric strong {
  font-size: 24px;
  font-weight: 550;
}

.archive-metric small {
  color: var(--green-dark);
  font-size: 8px;
}

.archive-trend {
  position: relative;
  display: flex;
  justify-content: center;
  padding: 17px;
  flex-direction: column;
}

.export-copy {
  display: flex;
  flex-direction: column;
  gap: 5px;
  margin-top: 10px;
}

.export-copy strong {
  font-size: 11px;
}

.export-copy small {
  color: var(--muted);
  font-size: 8px;
}

.archive-panel {
  overflow: hidden;
  animation: rise-in 0.45s 0.1s ease both;
}

.archive-toolbar {
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
  padding: 0 10px;
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

.account-filter {
  width: 155px;
}

.article-search {
  width: 230px;
}

.article-cell {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.article-thumb {
  position: relative;
  display: grid;
  width: 58px;
  height: 58px;
  flex: 0 0 58px;
  overflow: hidden;
  place-items: center;
  border-radius: 5px;
  color: rgba(255, 255, 255, 0.75);
  background: var(--ink);
  font-family: "Microsoft YaHei UI", sans-serif;
  font-size: 18px;
  font-weight: 700;
}

.article-thumb img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.article-thumb span {
  position: relative;
  z-index: 1;
}

.article-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 5px;
}

.title-line {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.title-line strong {
  overflow: hidden;
  font-size: 11px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.title-line em {
  padding: 2px 4px;
  border: 1px solid rgba(11, 151, 103, 0.22);
  border-radius: 3px;
  color: var(--green-dark);
  font-size: 7px;
  font-style: normal;
  white-space: nowrap;
}

.article-copy > small {
  overflow: hidden;
  color: var(--muted);
  font-size: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.article-byline {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #939891;
  font-size: 7px;
}

.article-byline i {
  width: 2px;
  height: 2px;
  border-radius: 50%;
  background: #b7bab4;
}

.engagement-cell {
  display: flex;
  gap: 9px;
}

.engagement-cell span {
  display: flex;
  align-items: center;
  gap: 3px;
  color: var(--muted);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 8px;
}

.date-cell,
.relative-cell {
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

.archive-panel :deep(.el-table__row) {
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
  .archive-summary {
    grid-template-columns: 1.3fr repeat(3, 0.8fr);
  }

  .archive-trend {
    display: none;
  }
}

@media (max-width: 900px) {
  .archive-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .status-tabs {
    overflow-x: auto;
  }

  .account-filter,
  .article-search {
    width: 100%;
  }
}

@media (max-width: 760px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .archive-summary {
    grid-template-columns: 1fr 1fr;
  }

  .archive-primary {
    grid-column: span 2;
    min-height: 112px;
  }

  .archive-metric {
    border-bottom: 1px solid var(--line);
  }

  .archive-metric:nth-child(3) {
    border-right: 0;
  }

  .archive-metric:nth-child(4) {
    grid-column: span 2;
    border-right: 0;
    border-bottom: 0;
  }
}

@media (max-width: 980px) {
  :deep(.article-published-column),
  :deep(.article-ingested-column) {
    display: none;
  }
}
</style>
