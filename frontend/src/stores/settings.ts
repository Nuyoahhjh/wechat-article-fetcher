import { reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '@/services/api'
import type { UserSettings } from '@/types'

export type { UserSettings } from '@/types'

const defaults: UserSettings = {
  displayName: '',
  notifyTaskFailed: true,
  notifyAgentOffline: true,
  notifyDailyDigest: false,
  autoRefresh: true,
  refreshInterval: 15,
  retentionDays: 0,
  screenshotQuality: 'standard',
  timezone: 'Asia/Shanghai',
}

export const useSettingsStore = defineStore('settings', () => {
  const form = reactive<UserSettings>({ ...defaults })
  const loading = ref(false)
  const loaded = ref(false)
  const saving = ref(false)

  async function load() {
    loading.value = true
    try {
      const response = await api.settings.get()
      Object.assign(form, defaults, response.settings)
      if (response.displayName) form.displayName = response.displayName
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  async function save() {
    saving.value = true
    try {
      const response = await api.settings.update(form.displayName, form)
      if (response.displayName) form.displayName = response.displayName
      loaded.value = true
    } finally {
      saving.value = false
    }
  }

  function reset() {
    Object.assign(form, defaults)
  }

  return { form, loading, loaded, saving, load, save, reset }
})
