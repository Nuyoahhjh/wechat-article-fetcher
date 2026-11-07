<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  Bell,
  ChevronDown,
  Menu,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
} from 'lucide-vue-next'
import dayjs from 'dayjs'
import CommandPalette from './CommandPalette.vue'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const ui = useUiStore()
const auth = useAuthStore()
const paletteOpen = ref(false)
const notificationsOpen = ref(false)

const title = computed(() => String(route.meta.title ?? '工作台'))
const eyebrow = computed(() => String(route.meta.eyebrow ?? 'CONTROL ROOM'))
const today = dayjs().format('YYYY.MM.DD / dddd')

function handleHotkey(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    paletteOpen.value = true
  }
}

onMounted(() => window.addEventListener('keydown', handleHotkey))
onBeforeUnmount(() => window.removeEventListener('keydown', handleHotkey))
</script>

<template>
  <header class="topbar">
    <div class="topbar-left">
      <button class="mobile-menu icon-button" title="打开导航" @click="ui.mobileNavOpen = true">
        <Menu :size="18" />
      </button>
      <button
        class="collapse-button icon-button"
        :title="ui.sidebarCollapsed ? '展开导航' : '收起导航'"
        @click="ui.toggleSidebar"
      >
        <PanelLeftOpen v-if="ui.sidebarCollapsed" :size="17" />
        <PanelLeftClose v-else :size="17" />
      </button>
      <div class="route-title">
        <span>{{ eyebrow }}</span>
        <strong>{{ title }}</strong>
      </div>
    </div>

    <div class="topbar-right">
      <button class="search-trigger" @click="paletteOpen = true">
        <Search :size="15" :stroke-width="1.8" />
        <span>搜索</span>
        <kbd>CTRL K</kbd>
      </button>

      <div class="realtime">
        <i />
        <span>REST 轮询</span>
      </div>

      <span class="today mono">{{ today }}</span>

      <div class="notification-wrap">
        <button class="icon-button" title="通知" @click="notificationsOpen = !notificationsOpen">
          <Bell :size="17" />
        </button>
        <Transition name="popover">
          <div v-if="notificationsOpen" class="notification-popover">
            <header>
              <strong>运行通知</strong>
              <span>暂未接入</span>
            </header>
            <div class="notification-empty">通知发送通道尚未接入</div>
          </div>
        </Transition>
      </div>

      <button class="profile-chip">
        <span>{{ (auth.user?.name || 'A').slice(0, 1) }}</span>
        <ChevronDown :size="13" />
      </button>
    </div>
  </header>

  <CommandPalette v-model="paletteOpen" />
</template>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 30;
  display: flex;
  height: 72px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 0 30px;
  border-bottom: 1px solid var(--line);
  background: rgba(243, 240, 231, 0.9);
  backdrop-filter: blur(14px);
}

.topbar-left,
.topbar-right {
  display: flex;
  align-items: center;
}

.topbar-left {
  gap: 14px;
}

.topbar-right {
  gap: 12px;
}

.collapse-button {
  border-color: transparent;
  background: transparent;
}

.mobile-menu {
  display: none;
  border-color: transparent;
  background: transparent;
}

.route-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.route-title span {
  color: var(--muted);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 0.18em;
}

.route-title strong {
  font-size: 15px;
  font-weight: 650;
}

.search-trigger {
  display: flex;
  height: 34px;
  align-items: center;
  gap: 8px;
  padding: 0 9px 0 11px;
  border: 1px solid var(--line);
  border-radius: 6px;
  color: var(--muted);
  background: rgba(255, 255, 255, 0.5);
  font-size: 11px;
  cursor: pointer;
}

.search-trigger:hover {
  border-color: var(--line-dark);
  background: #fff;
}

.search-trigger kbd {
  padding: 3px 5px;
  border: 1px solid var(--line);
  border-radius: 3px;
  color: #858b84;
  background: var(--paper);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 8px;
}

.realtime {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  color: var(--muted);
  font-size: 10px;
  white-space: nowrap;
}

.realtime i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--green);
  box-shadow: 0 0 0 3px rgba(11, 151, 103, 0.1);
}

.realtime i.disconnected {
  background: var(--orange);
  box-shadow: 0 0 0 3px rgba(239, 113, 77, 0.1);
}

.realtime i.connecting {
  background: var(--amber);
  animation: blink 1.2s infinite;
}

.today {
  color: var(--muted);
  font-size: 9px;
  letter-spacing: 0.03em;
  white-space: nowrap;
}

.notification-wrap {
  position: relative;
}

.notification-dot {
  position: absolute;
  top: 7px;
  right: 7px;
  width: 5px;
  height: 5px;
  border: 1px solid var(--paper);
  border-radius: 50%;
  background: var(--orange);
}

.notification-popover {
  position: absolute;
  top: 43px;
  right: 0;
  width: 330px;
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: var(--shadow);
}

.notification-popover header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid var(--line);
}

.notification-popover header strong {
  font-size: 12px;
}

.notification-popover header span {
  color: var(--muted);
  font-size: 9px;
}

.notification-item {
  display: flex;
  gap: 10px;
  padding: 13px 16px;
  border-bottom: 1px solid var(--line);
}

.notification-item:last-child {
  border-bottom: 0;
}

.notification-item > i {
  width: 6px;
  height: 6px;
  flex: 0 0 6px;
  margin-top: 5px;
  border-radius: 50%;
  background: var(--green);
}

.notification-item > i.warning {
  background: var(--amber);
}

.notification-item div {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.notification-item strong {
  font-size: 11px;
  font-weight: 650;
}

.notification-item span {
  overflow: hidden;
  color: var(--muted);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-item small {
  color: #9a9e97;
  font-size: 9px;
}

.profile-chip {
  display: flex;
  height: 34px;
  align-items: center;
  gap: 5px;
  padding: 0 7px 0 3px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--surface);
  cursor: pointer;
}

.profile-chip span {
  display: grid;
  width: 27px;
  height: 27px;
  place-items: center;
  border-radius: 5px;
  color: #2f3b1d;
  background: #dff58f;
  font-size: 10px;
  font-weight: 700;
}

.popover-enter-active,
.popover-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
  transform-origin: top right;
}

.popover-enter-from,
.popover-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

@keyframes blink {
  50% {
    opacity: 0.35;
  }
}

@media (max-width: 1100px) {
  .today,
  .realtime span {
    display: none;
  }

  .search-trigger span {
    display: none;
  }
}

@media (max-width: 900px) {
  .topbar {
    height: 64px;
    padding: 0 16px;
  }

  .mobile-menu {
    display: inline-flex;
  }

  .collapse-button,
  .route-title span {
    display: none;
  }

  .search-trigger kbd,
  .profile-chip svg {
    display: none;
  }
}

@media (max-width: 560px) {
  .search-trigger,
  .realtime {
    display: none;
  }
}
</style>
