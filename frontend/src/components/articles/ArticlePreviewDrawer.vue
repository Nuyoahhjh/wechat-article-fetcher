<script setup lang="ts">
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import {
  ArrowUpRight,
  CalendarDays,
  Copy,
  Newspaper,
  UserRound,
} from 'lucide-vue-next'
import { useArticleStore } from '@/stores/articles'
import { formatDateTime } from '@/utils/format'
import StatusPill from '@/components/common/StatusPill.vue'

const articles = useArticleStore()
const paragraphs = computed(() =>
  (articles.selectedArticle?.contentText || articles.selectedArticle?.digest || '暂无正文')
    .split(/\n{2,}|\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean),
)
const statusText = computed(() => {
  const status = articles.selectedArticle?.status
  if (status === 'complete') return '正文与元数据已入库'
  if (status === 'metadata_only') return '仅元数据入库，未抓取正文'
  if (status === 'content_missing') return '正文提取失败，已保留元数据'
  if (status === 'failed') return '文章入库失败'
  return '状态未知'
})

function close() {
  articles.drawerOpen = false
}

async function copyLink() {
  if (!articles.selectedArticle) return
  await navigator.clipboard.writeText(articles.selectedArticle.url)
  ElMessage.success('文章链接已复制')
}
</script>

<template>
  <el-drawer
    :model-value="articles.drawerOpen"
    append-to-body
    size="min(680px, 94vw)"
    :show-close="false"
    @update:model-value="articles.drawerOpen = $event"
  >
    <template #header>
      <div class="drawer-head">
        <div>
          <p class="eyebrow">ARCHIVE ITEM</p>
          <h2>文章详情</h2>
        </div>
        <div class="head-actions">
          <button class="icon-button" title="复制链接" @click="copyLink">
            <Copy :size="15" />
          </button>
          <a
            v-if="articles.selectedArticle"
            class="icon-button"
            title="打开原文"
            :href="articles.selectedArticle.url"
            target="_blank"
            rel="noreferrer"
          >
            <ArrowUpRight :size="16" />
          </a>
          <button class="icon-button" title="关闭" @click="close">×</button>
        </div>
      </div>
    </template>

    <article v-loading="articles.detailLoading" v-if="articles.selectedArticle" class="article-preview">
      <div class="article-hero">
        <img
          v-if="articles.selectedArticle.coverUrl"
          class="hero-cover"
          :src="articles.selectedArticle.coverUrl"
          alt=""
        />
        <span class="hero-grid" />
        <span class="hero-label mono">WECHAT / ARCHIVE</span>
        <div class="hero-mark">
          <strong>{{ articles.selectedArticle.accountName.slice(0, 1) }}</strong>
          <span>{{ articles.selectedArticle.accountName }}</span>
        </div>
        <span class="hero-date mono">
          {{ articles.selectedArticle.publishedAt ? formatDateTime(articles.selectedArticle.publishedAt) : '发布时间未知' }}
        </span>
      </div>

      <div class="article-content">
        <div class="article-status-line">
          <StatusPill :status="articles.selectedArticle.status" />
          <span class="article-id mono">{{ articles.selectedArticle.id }}</span>
        </div>

        <h1>{{ articles.selectedArticle.title }}</h1>
        <p class="article-digest">{{ articles.selectedArticle.digest }}</p>

        <div class="article-meta">
          <span><UserRound :size="13" />{{ articles.selectedArticle.author }}</span>
          <span><CalendarDays :size="13" />{{ articles.selectedArticle.publishedAt ? formatDateTime(articles.selectedArticle.publishedAt) : '发布时间未知' }}</span>
        </div>

        <div class="article-divider">
          <span>正文预览</span>
        </div>

        <div class="article-body">
          <p v-for="(paragraph, index) in paragraphs" :key="`${index}-${paragraph.slice(0, 16)}`">
            {{ paragraph }}
          </p>
        </div>

        <div class="source-note">
          <Newspaper :size="16" />
          <div>
            <strong>采集来源</strong>
            <span>{{ articles.selectedArticle.accountName }} · {{ articles.selectedArticle.source }}</span>
          </div>
          <span class="source-agent mono">{{ articles.selectedArticle.agentId.toUpperCase() }}</span>
        </div>

        <div class="archive-meta">
          <div>
            <span>采集时间</span>
            <strong>{{ formatDateTime(articles.selectedArticle.collectedAt) }}</strong>
          </div>
          <div>
            <span>内容状态</span>
            <strong>{{ statusText }}</strong>
          </div>
          <div>
            <span>来源链接</span>
            <a :href="articles.selectedArticle.url" target="_blank" rel="noreferrer">
              mp.weixin.qq.com
              <ArrowUpRight :size="11" />
            </a>
          </div>
        </div>
      </div>
    </article>
  </el-drawer>
</template>

<style scoped>
.drawer-head {
  display: flex;
  width: 100%;
  align-items: flex-start;
  justify-content: space-between;
}

.drawer-head h2 {
  margin: 0;
  font-size: 21px;
  font-weight: 680;
}

.head-actions {
  display: flex;
  gap: 6px;
}

.head-actions .icon-button:last-child {
  font-size: 20px;
}

.article-hero {
  position: relative;
  height: 220px;
  overflow: hidden;
  color: rgba(255, 255, 255, 0.88);
  background: var(--ink);
}

.hero-cover {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0.42;
}

.hero-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.08) 1px, transparent 1px);
  background-size: 34px 34px;
  mask-image: linear-gradient(to right, rgba(0, 0, 0, 0.55), transparent 80%);
}

.hero-label {
  position: absolute;
  top: 18px;
  left: 20px;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 0.15em;
  opacity: 0.5;
}

.hero-mark {
  position: absolute;
  right: 28px;
  bottom: 24px;
  display: flex;
  max-width: 240px;
  align-items: flex-end;
  gap: 9px;
  flex-direction: column;
  text-align: right;
}

.hero-mark strong {
  font-family: "Microsoft YaHei UI", sans-serif;
  font-size: 88px;
  font-weight: 700;
  letter-spacing: -0.08em;
  line-height: 0.75;
  opacity: 0.2;
}

.hero-mark span {
  font-size: 12px;
  font-weight: 650;
  letter-spacing: 0.04em;
}

.hero-date {
  position: absolute;
  bottom: 18px;
  left: 20px;
  font-size: 8px;
  opacity: 0.45;
}

.article-content {
  padding: 24px 30px 52px;
}

.article-status-line {
  display: flex;
  align-items: center;
  gap: 7px;
}

.original-tag {
  padding: 4px 7px;
  border: 1px solid rgba(11, 151, 103, 0.2);
  border-radius: 999px;
  color: var(--green-dark);
  font-size: 8px;
  font-weight: 650;
}

.article-id {
  margin-left: auto;
  color: #9ca099;
  font-size: 8px;
}

.article-content h1 {
  margin: 16px 0 12px;
  font-size: clamp(24px, 3.4vw, 34px);
  font-weight: 680;
  letter-spacing: -0.045em;
  line-height: 1.35;
}

.article-digest {
  margin: 0 0 17px;
  color: var(--muted);
  font-size: 12px;
  line-height: 1.8;
}

.article-meta {
  display: flex;
  gap: 16px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--line);
  color: var(--muted);
  flex-wrap: wrap;
}

.article-meta span {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 9px;
}

.article-divider {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 23px 0 17px;
  color: var(--muted);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 8px;
  font-weight: 650;
  letter-spacing: 0.12em;
}

.article-divider::after {
  height: 1px;
  flex: 1;
  content: "";
  background: var(--line);
}

.article-body {
  color: #344239;
  font-family: "Microsoft YaHei UI", "Microsoft YaHei", sans-serif;
  font-size: 13px;
  line-height: 2;
}

.article-body p {
  margin: 0 0 16px;
}

.source-note {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 24px;
  padding: 13px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--paper);
}

.source-note > div {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.source-note strong {
  font-size: 9px;
}

.source-note span {
  color: var(--muted);
  font-size: 8px;
}

.source-note .source-agent {
  font-size: 7px;
  letter-spacing: 0.06em;
}

.archive-meta {
  display: grid;
  gap: 1px;
  margin-top: 16px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--line);
  grid-template-columns: repeat(3, 1fr);
}

.archive-meta > div {
  display: flex;
  min-height: 71px;
  justify-content: center;
  padding: 11px;
  flex-direction: column;
  gap: 5px;
  background: var(--surface);
}

.archive-meta span {
  color: var(--muted);
  font-size: 8px;
}

.archive-meta strong,
.archive-meta a {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--ink);
  font-size: 9px;
  font-weight: 600;
}

.archive-meta a {
  color: var(--green-dark);
}

@media (max-width: 600px) {
  .article-hero {
    height: 170px;
  }

  .article-content {
    padding: 20px;
  }

  .archive-meta {
    grid-template-columns: 1fr;
  }

  .archive-meta > div {
    min-height: 58px;
  }
}
</style>
