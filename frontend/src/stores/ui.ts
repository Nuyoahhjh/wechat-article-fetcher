import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', () => {
  const sidebarCollapsed = ref(false)
  const mobileNavOpen = ref(false)

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function closeMobileNav() {
    mobileNavOpen.value = false
  }

  return {
    sidebarCollapsed,
    mobileNavOpen,
    toggleSidebar,
    closeMobileNav,
  }
})
