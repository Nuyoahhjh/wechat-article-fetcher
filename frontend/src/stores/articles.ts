import { ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '@/services/api'
import type { Article, ArticleListQuery } from '@/types'

export const useArticleStore = defineStore('articles', () => {
  const items = ref<Article[]>([])
  const total = ref(0)
  const loading = ref(false)
  const detailLoading = ref(false)
  const exporting = ref(false)
  const selectedArticle = ref<Article | null>(null)
  const drawerOpen = ref(false)
  const query = ref<ArticleListQuery>({
    page: 1,
    pageSize: 10,
    status: 'all',
    keyword: '',
  })

  const accountOptions = ref<string[]>([])

  async function fetchArticles(nextQuery?: Partial<ArticleListQuery>) {
    query.value = { ...query.value, ...nextQuery }
    loading.value = true
    try {
      const result = await api.articles.list(query.value)
      items.value = result.items
      total.value = result.total
    } finally {
      loading.value = false
    }
  }

  async function openArticle(article: Article) {
    selectedArticle.value = article
    drawerOpen.value = true
    detailLoading.value = true
    try {
      selectedArticle.value = await api.articles.get(article.id)
    } finally {
      detailLoading.value = false
    }
  }

  async function fetchAccounts() {
    accountOptions.value = await api.articles.accounts()
  }

  async function exportArticles(format: 'json' | 'csv' | 'markdown') {
    exporting.value = true
    try {
      const response = await api.articles.export(format, query.value)
      const contentDisposition = String(response.headers['content-disposition'] ?? '')
      const match = contentDisposition.match(/filename="([^"]+)"/)
      const extension = format === 'markdown' ? 'md' : format
      const filename = match?.[1] ?? `shikan-articles.${extension}`
      const url = URL.createObjectURL(response.data)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = filename
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(url)
      return Number(response.headers['x-export-count'] ?? 0)
    } finally {
      exporting.value = false
    }
  }

  return {
    items,
    total,
    loading,
    detailLoading,
    exporting,
    selectedArticle,
    drawerOpen,
    query,
    accountOptions,
    fetchArticles,
    fetchAccounts,
    openArticle,
    exportArticles,
  }
})
