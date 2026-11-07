import { onBeforeUnmount, onMounted, watch } from 'vue'
import { useSettingsStore } from '@/stores/settings'

export function usePolling(callback: () => void | Promise<void>) {
  const settings = useSettingsStore()
  let timer: number | undefined

  function stop() {
    if (timer !== undefined) {
      window.clearInterval(timer)
      timer = undefined
    }
  }

  function start() {
    stop()
    if (!settings.form.autoRefresh) return
    const seconds = Math.max(5, settings.form.refreshInterval)
    timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') {
        void callback()
      }
    }, seconds * 1000)
  }

  onMounted(async () => {
    if (!settings.loaded) {
      try {
        await settings.load()
      } catch {
        // Keep the page usable; the initial load still reports the real error.
      }
    }
    start()
  })

  watch(
    () => [settings.form.autoRefresh, settings.form.refreshInterval],
    start,
  )

  onBeforeUnmount(stop)
}
