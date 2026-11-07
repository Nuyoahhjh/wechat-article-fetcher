<script setup lang="ts">
import { computed } from 'vue'
import SideNavigation from '@/components/layout/SideNavigation.vue'
import TopBar from '@/components/layout/TopBar.vue'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
const contentClass = computed(() => ({ collapsed: ui.sidebarCollapsed }))
</script>

<template>
  <div class="console">
    <SideNavigation />
    <Transition name="overlay">
      <button
        v-if="ui.mobileNavOpen"
        class="nav-overlay"
        aria-label="关闭导航"
        @click="ui.closeMobileNav"
      />
    </Transition>
    <SideNavigation v-if="ui.mobileNavOpen" mobile class="mobile-sidebar" />

    <div class="console-content" :class="contentClass">
      <TopBar />
      <main class="main-content">
        <RouterView v-slot="{ Component }">
          <Transition name="route" mode="out-in">
            <component :is="Component" :key="$route.fullPath" />
          </Transition>
        </RouterView>
      </main>
    </div>
  </div>
</template>

<style scoped>
.console {
  min-height: 100vh;
}

.console-content {
  min-height: 100vh;
  margin-left: 236px;
  transition: margin-left 0.24s ease;
}

.console-content.collapsed {
  margin-left: 72px;
}

.main-content {
  padding: 30px;
}

.nav-overlay {
  position: fixed;
  inset: 0;
  z-index: 45;
  border: 0;
  background: rgba(8, 13, 10, 0.44);
  backdrop-filter: blur(2px);
}

.mobile-sidebar {
  z-index: 55;
}

.overlay-enter-active,
.overlay-leave-active {
  transition: opacity 0.2s ease;
}

.overlay-enter-from,
.overlay-leave-to {
  opacity: 0;
}

.route-enter-active,
.route-leave-active {
  transition: opacity 0.14s ease, transform 0.14s ease;
}

.route-enter-from {
  opacity: 0;
  transform: translateY(4px);
}

.route-leave-to {
  opacity: 0;
}

@media (max-width: 900px) {
  .console-content,
  .console-content.collapsed {
    margin-left: 0;
  }

  .main-content {
    padding: 20px 16px 36px;
  }
}
</style>
