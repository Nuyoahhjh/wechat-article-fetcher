import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import ConsoleLayout from '@/layouts/ConsoleLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true, title: '登录' },
    },
    {
      path: '/',
      component: ConsoleLayout,
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { title: '工作台', eyebrow: 'CONTROL ROOM' },
        },
        {
          path: 'agents',
          name: 'agents',
          component: () => import('@/views/AgentsView.vue'),
          meta: { title: 'Agent 网络', eyebrow: 'FLEET' },
        },
        {
          path: 'agents/:id',
          name: 'agent-detail',
          component: () => import('@/views/AgentDetailView.vue'),
          meta: { title: 'Agent 详情', eyebrow: 'NODE INSPECTOR' },
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: () => import('@/views/TasksView.vue'),
          meta: { title: '采集任务', eyebrow: 'PIPELINES' },
        },
        {
          path: 'articles',
          name: 'articles',
          component: () => import('@/views/ArticlesView.vue'),
          meta: { title: '文章资料库', eyebrow: 'ARCHIVE' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue'),
          meta: { title: '平台设置', eyebrow: 'PREFERENCES' },
        },
        {
          path: 'users',
          name: 'users',
          component: () => import('@/views/UsersView.vue'),
          meta: { title: '用户与权限', eyebrow: 'ACCESS', roles: ['admin'] },
        },
        {
          path: 'audit-logs',
          name: 'audit-logs',
          component: () => import('@/views/AuditLogsView.vue'),
          meta: { title: '操作审计', eyebrow: 'AUDIT', roles: ['admin'] },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue'),
      meta: { public: true, title: '页面不存在' },
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
  const roles = Array.isArray(to.meta.roles) ? to.meta.roles : []
  if (roles.length && auth.user && !roles.includes(auth.user.role)) {
    return { name: 'dashboard' }
  }
  document.title = to.meta.title ? `${String(to.meta.title)} · 拾刊` : '拾刊 · Agent Console'
  return true
})

export default router
