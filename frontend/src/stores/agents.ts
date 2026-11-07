import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '@/services/api'
import type {
  Agent,
  AgentListQuery,
  AgentLog,
  AgentStatus,
  QrCodePayload,
  ScreenSnapshot,
} from '@/types'

export const useAgentStore = defineStore('agents', () => {
  const items = ref<Agent[]>([])
  const total = ref(0)
  const loading = ref(false)
  const detailLoading = ref(false)
  const currentAgent = ref<Agent | null>(null)
  const logs = ref<AgentLog[]>([])
  const qrcode = ref<QrCodePayload | null>(null)
  const qrcodeLoading = ref(false)
  const screen = ref<ScreenSnapshot | null>(null)
  const query = ref<AgentListQuery>({
    page: 1,
    pageSize: 10,
    status: 'all',
    keyword: '',
  })

  const onlineCount = computed(
    () => items.value.filter((agent) => agent.status === 'online' || agent.status === 'busy').length,
  )

  async function fetchAgents(nextQuery?: Partial<AgentListQuery>) {
    query.value = { ...query.value, ...nextQuery }
    loading.value = true
    try {
      const result = await api.agents.list(query.value)
      items.value = result.items
      total.value = result.total
    } finally {
      loading.value = false
    }
  }

  async function fetchAgentDetail(id: string) {
    detailLoading.value = true
    try {
      const [agent, agentLogs, snapshot, latestQr] = await Promise.all([
        api.agents.get(id),
        api.agents.logs(id),
        api.agents.screen(id),
        api.agents.latestLogin(id),
      ])
      currentAgent.value = agent
      const existingIndex = items.value.findIndex((item) => item.id === agent.id)
      if (existingIndex >= 0) {
        items.value[existingIndex] = agent
      } else {
        items.value.push(agent)
      }
      if (total.value < items.value.length) {
        total.value = items.value.length
      }
      logs.value = agentLogs
      screen.value = snapshot
      qrcode.value = latestQr
    } finally {
      detailLoading.value = false
    }
  }

  async function refreshQrCode() {
    if (!currentAgent.value) return
    qrcodeLoading.value = true
    try {
      qrcode.value = await api.agents.requestLogin(currentAgent.value.id)
    } finally {
      qrcodeLoading.value = false
    }
  }

  async function refreshLoginStatus() {
    if (!currentAgent.value) return
    qrcode.value = await api.agents.latestLogin(currentAgent.value.id)
  }

  async function updateConfig(maxConcurrency: number, config?: Record<string, unknown>) {
    if (!currentAgent.value) return
    currentAgent.value = await api.agents.updateConfig(
      currentAgent.value.id,
      { maxConcurrency, config },
    )
  }

  async function disableAgent() {
    if (!currentAgent.value) return
    await api.agents.disable(currentAgent.value.id)
    currentAgent.value.status = 'disabled'
  }

  function applyStatus(agentId: string, status: AgentStatus, lastHeartbeat: string) {
    const inList = items.value.find((agent) => agent.id === agentId)
    if (inList) {
      inList.status = status
      inList.lastHeartbeat = lastHeartbeat
    }
    if (currentAgent.value?.id === agentId) {
      currentAgent.value.status = status
      currentAgent.value.lastHeartbeat = lastHeartbeat
    }
  }

  function appendLog(log: AgentLog) {
    if (currentAgent.value?.id !== log.agentId) return
    logs.value.unshift(log)
    logs.value = logs.value.slice(0, 50)
  }

  function updateScreenTimestamp(capturedAt: string) {
    if (screen.value) screen.value.capturedAt = capturedAt
  }

  return {
    items,
    total,
    loading,
    detailLoading,
    currentAgent,
    logs,
    qrcode,
    qrcodeLoading,
    screen,
    query,
    onlineCount,
    fetchAgents,
    fetchAgentDetail,
    refreshQrCode,
    refreshLoginStatus,
    updateConfig,
    disableAgent,
    applyStatus,
    appendLog,
    updateScreenTimestamp,
  }
})
