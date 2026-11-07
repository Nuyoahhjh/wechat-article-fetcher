<script setup lang="ts">
import { computed } from 'vue'
import {
  LayoutDashboard,
  ListChecks,
  LogOut,
  Newspaper,
  RadioTower,
  ScrollText,
  Settings,
  Sparkles,
  UsersRound,
} from 'lucide-vue-next'
import { useRoute, useRouter } from 'vue-router'
import BrandMark from '@/components/common/BrandMark.vue'
import { useAuthStore } from '@/stores/auth'
import { useAgentStore } from '@/stores/agents'
import { useUiStore } from '@/stores/ui'

defineProps<{
  mobile?: boolean
}>()

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const agents = useAgentStore()
const ui = useUiStore()

const availableAgents = computed(() => agents.onlineCount)
const totalAgents = computed(() => agents.total)

const menu = computed(() => [
  { name: 'dashboard', label: '工作台', icon: LayoutDashboard },
  { name: 'agents', label: 'Agent 网络', icon: RadioTower },
  { name: 'tasks', label: '采集任务', icon: ListChecks },
  { name: 'articles', label: '文章资料库', icon: Newspaper },
  ...(auth.user?.role === 'admin'
    ? [
        { name: 'users', label: '用户与权限', icon: UsersRound },
        { name: 'audit-logs', label: '操作审计', icon: ScrollText },
      ]
    : []),
])

const isActive = (name: string) => {
  if (name === 'agents') return route.name === 'agents' || route.name === 'agent-detail'
  return route.name === name
}

async function logout() {
  auth.logout()
  await router.push({ name: 'login' })
}
</script>

<template>
  <aside class="sidebar" :class="{ collapsed: ui.sidebarCollapsed && !mobile, mobile }">
    <div class="sidebar-brand">
      <BrandMark :compact="ui.sidebarCollapsed && !mobile" inverse />
    </div>

    <div class="nav-section">
      <span v-if="!ui.sidebarCollapsed || mobile" class="nav-label">MONITOR</span>
      <button
        v-for="item in menu"
        :key="item.name"
        class="nav-item"
        :class="{ active: isActive(item.name) }"
        :title="ui.sidebarCollapsed && !mobile ? item.label : undefined"
        @click="router.push({ name: item.name }); ui.closeMobileNav()"
      >
        <component :is="item.icon" :size="18" :stroke-width="1.8" />
        <span v-if="!ui.sidebarCollapsed || mobile">{{ item.label }}</span>
        <i v-if="isActive(item.name)" />
      </button>
    </div>

    <div class="sidebar-spacer" />

    <div class="nav-section nav-bottom">
      <button
        class="nav-item"
        :class="{ active: route.name === 'settings' }"
        :title="ui.sidebarCollapsed && !mobile ? '平台设置' : undefined"
        @click="router.push({ name: 'settings' }); ui.closeMobileNav()"
      >
        <Settings :size="18" :stroke-width="1.8" />
        <span v-if="!ui.sidebarCollapsed || mobile">平台设置</span>
      </button>
    </div>

    <div class="agent-note" v-if="!ui.sidebarCollapsed || mobile">
      <div class="note-icon"><Sparkles :size="15" /></div>
      <div>
        <strong>{{ agents.onlineCount > 0 ? '网络运行正常' : '等待 Agent 接入' }}</strong>
        <span>{{ availableAgents }} / {{ totalAgents }} Agent 可调度</span>
      </div>
    </div>

    <div class="sidebar-user">
      <div class="user-avatar">{{ auth.user?.name.slice(0, 1) ?? '陈' }}</div>
      <div v-if="!ui.sidebarCollapsed || mobile" class="user-copy">
        <strong>{{ auth.user?.name ?? '管理员' }}</strong>
        <span>{{ auth.user?.role ?? '平台管理员' }}</span>
      </div>
      <button
        v-if="!ui.sidebarCollapsed || mobile"
        class="logout-button"
        title="退出登录"
        @click="logout"
      >
        <LogOut :size="16" />
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 50;
  display: flex;
  width: 236px;
  padding: 18px 14px 14px;
  flex-direction: column;
  color: #e9eee9;
  background:
    linear-gradient(rgba(255, 255, 255, 0.027) 1px, transparent 1px),
    var(--ink);
  background-size: 100% 42px;
  transition: width 0.24s ease, transform 0.24s ease;
}

.sidebar.collapsed {
  width: 72px;
}

.sidebar-brand {
  display: flex;
  height: 48px;
  align-items: center;
  padding: 0 7px;
  margin-bottom: 28px;
}

.nav-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-label {
  padding: 0 10px 8px;
  color: rgba(233, 238, 233, 0.34);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.2em;
}

.nav-item {
  position: relative;
  display: flex;
  width: 100%;
  height: 42px;
  align-items: center;
  gap: 12px;
  padding: 0 11px;
  border: 0;
  border-radius: 6px;
  color: rgba(233, 238, 233, 0.62);
  background: transparent;
  font-size: 13px;
  font-weight: 520;
  text-align: left;
  cursor: pointer;
  transition: color 0.18s ease, background 0.18s ease;
}

.nav-item:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.06);
}

.nav-item.active {
  color: var(--ink);
  background: var(--acid);
  font-weight: 700;
}

.nav-item.active i {
  position: absolute;
  right: 10px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--ink);
}

.collapsed .nav-item {
  justify-content: center;
  padding: 0;
}

.sidebar-spacer {
  flex: 1;
}

.nav-bottom {
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.09);
}

.agent-note {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 16px 0;
  padding: 12px;
  border: 1px solid rgba(201, 243, 106, 0.16);
  border-radius: 7px;
  background: rgba(201, 243, 106, 0.055);
}

.note-icon {
  display: grid;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  place-items: center;
  border-radius: 5px;
  color: var(--ink);
  background: var(--acid);
}

.agent-note div:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.agent-note strong {
  font-size: 11px;
  font-weight: 650;
}

.agent-note span {
  color: rgba(233, 238, 233, 0.46);
  font-size: 9px;
}

.sidebar-user {
  display: flex;
  min-height: 51px;
  align-items: center;
  gap: 9px;
  padding: 7px;
}

.user-avatar {
  display: grid;
  width: 33px;
  height: 33px;
  flex: 0 0 33px;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 6px;
  color: var(--ink);
  background: #e4d6b2;
  font-size: 12px;
  font-weight: 750;
}

.user-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.user-copy strong {
  overflow: hidden;
  font-size: 11px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-copy span {
  color: rgba(233, 238, 233, 0.42);
  font-size: 9px;
}

.logout-button {
  display: grid;
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  place-items: center;
  border: 0;
  border-radius: 5px;
  color: rgba(233, 238, 233, 0.44);
  background: transparent;
  cursor: pointer;
}

.logout-button:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.07);
}

.mobile {
  width: 252px;
  box-shadow: 20px 0 50px rgba(0, 0, 0, 0.22);
}

@media (max-width: 900px) {
  .sidebar {
    transform: translateX(-105%);
  }

  .sidebar.mobile {
    transform: translateX(0);
  }
}
</style>
