<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Bell,
  Check,
  Database,
  Gauge,
  Globe2,
  KeyRound,
  Monitor,
  RefreshCw,
  RotateCcw,
  Save,
  ShieldCheck,
  UserRound,
  Wifi,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { api } from '@/services/api'

const auth = useAuthStore()
const settings = useSettingsStore()
const activeSection = ref('profile')
const saving = ref(false)
const databaseState = ref<'ok' | 'error' | 'unknown'>('unknown')
const passwordOpen = ref(false)
const passwordSaving = ref(false)
const passwordForm = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const sections = [
  { id: 'profile', label: '账号资料', icon: UserRound },
  { id: 'notification', label: '通知提醒', icon: Bell },
  { id: 'refresh', label: '刷新与数据', icon: Gauge },
  { id: 'service', label: '服务连接', icon: Wifi },
  { id: 'security', label: '安全与权限', icon: ShieldCheck },
]

const sectionRefs = ref<Record<string, HTMLElement | null>>({})

function setSectionRef(id: string, el: HTMLElement | null) {
  sectionRefs.value[id] = el
}

function scrollToSection(id: string) {
  activeSection.value = id
  sectionRefs.value[id]?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function saveSettings() {
  saving.value = true
  try {
    await settings.save()
    auth.setDisplayName(settings.form.displayName)
    ElMessage.success('设置已保存到平台')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '设置保存失败')
  } finally {
    saving.value = false
  }
}

function resetSettings() {
  settings.reset()
  ElMessage.info('已恢复表单默认值，点击保存后写入平台')
}

function openPasswordDialog() {
  passwordForm.currentPassword = ''
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  passwordOpen.value = true
}

async function changePassword() {
  if (passwordForm.newPassword.length < 8) {
    ElMessage.warning('新密码至少 8 位')
    return
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  passwordSaving.value = true
  try {
    await api.auth.password(passwordForm.currentPassword, passwordForm.newPassword)
    passwordOpen.value = false
    ElMessage.success('密码已修改')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '密码修改失败')
  } finally {
    passwordSaving.value = false
  }
}

onMounted(async () => {
  try {
    await Promise.all([settings.load(), auth.refreshUser()])
    const dashboard = await api.dashboard.get()
    databaseState.value = dashboard.database === 'ok' ? 'ok' : 'error'
  } catch (error) {
    databaseState.value = 'error'
    ElMessage.error(error instanceof Error ? error.message : '设置加载失败')
  }
})
</script>

<template>
  <div class="page-shell settings-page">
    <header class="page-header">
      <div>
        <p class="eyebrow">CONSOLE PREFERENCES</p>
        <h1 class="page-title">平台设置</h1>
        <p class="page-lead">管理个人偏好、通知策略、数据保留周期与实时连接状态。</p>
      </div>
      <div class="header-actions">
        <el-button @click="resetSettings"><RotateCcw :size="14" />恢复默认</el-button>
        <el-button type="primary" :loading="saving || settings.loading" @click="saveSettings">
          <Save v-if="!saving" :size="14" />保存更改
        </el-button>
      </div>
    </header>

    <div class="settings-layout">
      <aside class="settings-nav">
        <span class="panel-kicker">SETTINGS INDEX</span>
        <button
          v-for="section in sections"
          :key="section.id"
          :class="{ active: activeSection === section.id }"
          @click="scrollToSection(section.id)"
        >
          <component :is="section.icon" :size="15" />
          {{ section.label }}
          <span />
        </button>
        <div class="settings-version">
          <span class="mono">SHIKAN CONSOLE</span>
          <strong class="mono">v0.1.0</strong>
        </div>
      </aside>

      <main class="settings-content">
        <section :ref="(el) => setSectionRef('profile', el as HTMLElement | null)" class="setting-section panel">
          <div class="section-heading">
            <span class="section-icon"><UserRound :size="17" /></span>
            <div>
              <h2>账号资料</h2>
              <p>工作台中展示的用户信息。</p>
            </div>
          </div>
          <div class="section-body profile-body">
            <div class="profile-avatar">{{ (auth.user?.name || 'A').slice(0, 1) }}</div>
            <div class="profile-fields">
              <div class="field">
                <label>显示名称</label>
                <el-input v-model="settings.form.displayName" />
              </div>
              <div class="field">
                <label>登录邮箱</label>
                <el-input :model-value="auth.user?.email" disabled />
              </div>
              <div class="field">
                <label>角色</label>
                <el-input :model-value="auth.user?.role" disabled />
              </div>
              <div class="field">
                <label>时区</label>
                <el-select v-model="settings.form.timezone">
                  <el-option label="中国标准时间 (UTC+08:00)" value="Asia/Shanghai" />
                  <el-option label="UTC" value="UTC" />
                </el-select>
              </div>
            </div>
          </div>
        </section>

        <section :ref="(el) => setSectionRef('notification', el as HTMLElement | null)" class="setting-section panel">
          <div class="section-heading">
            <span class="section-icon"><Bell :size="17" /></span>
            <div>
              <h2>通知提醒</h2>
              <p>选择需要在工作台和通知中心接收的运行事件。</p>
            </div>
          </div>
          <div class="section-body switch-list">
            <label>
              <span class="switch-icon warning"><Monitor :size="15" /></span>
              <span class="switch-copy">
                <strong>Agent 离线提醒</strong>
                <small>节点持续 60 秒未上报心跳时发送通知</small>
              </span>
              <el-switch v-model="settings.form.notifyAgentOffline" />
            </label>
            <label>
              <span class="switch-icon danger"><RefreshCw :size="15" /></span>
              <span class="switch-copy">
                <strong>任务失败提醒</strong>
                <small>采集任务中断、登录失效或重试耗尽时发送通知</small>
              </span>
              <el-switch v-model="settings.form.notifyTaskFailed" />
            </label>
            <label>
              <span class="switch-icon success"><Database :size="15" /></span>
              <span class="switch-copy">
                <strong>每日内容摘要</strong>
                <small>每天 09:00 汇总前一天新增文章与采集质量</small>
              </span>
              <el-switch v-model="settings.form.notifyDailyDigest" />
            </label>
            <div class="data-note">
              <Bell :size="15" />
              <span>通知偏好会持久化到平台；消息发送通道暂未接入。</span>
            </div>
          </div>
        </section>

        <section :ref="(el) => setSectionRef('refresh', el as HTMLElement | null)" class="setting-section panel">
          <div class="section-heading">
            <span class="section-icon"><Gauge :size="17" /></span>
            <div>
              <h2>刷新与数据</h2>
              <p>控制实时画面、任务进度与内容数据的刷新策略。</p>
            </div>
          </div>
          <div class="section-body">
            <label class="inline-switch">
              <span class="switch-copy">
                <strong>启用自动刷新</strong>
                <small>页面处于前台时自动同步列表和状态</small>
              </span>
              <el-switch v-model="settings.form.autoRefresh" />
            </label>
            <div class="form-grid">
              <div class="field">
                <label>自动刷新间隔</label>
                <el-select v-model="settings.form.refreshInterval" :disabled="!settings.form.autoRefresh">
                  <el-option label="5 秒" :value="5" />
                  <el-option label="10 秒" :value="10" />
                  <el-option label="15 秒" :value="15" />
                  <el-option label="30 秒" :value="30" />
                  <el-option label="60 秒" :value="60" />
                </el-select>
              </div>
              <div class="field">
                <label>服务端数据保留</label>
                <el-input model-value="由部署环境变量控制" disabled />
              </div>
            </div>
            <div class="data-note">
              <Database :size="15" />
              <span>文章、日志与截图的清理周期由服务端部署策略控制。</span>
            </div>
          </div>
        </section>

        <section :ref="(el) => setSectionRef('service', el as HTMLElement | null)" class="setting-section panel">
          <div class="section-heading">
            <span class="section-icon"><Wifi :size="17" /></span>
            <div>
              <h2>服务连接</h2>
              <p>当前管理端与后端 API、实时事件通道的连接状态。</p>
            </div>
          </div>
          <div class="section-body">
            <div class="connection-card">
              <div class="connection-row">
                <span class="connection-icon"><Wifi :size="17" /></span>
                <div>
                  <strong>数据同步方式</strong>
                  <small>管理端使用 REST 短轮询同步状态</small>
                </div>
                <span class="connection-state connected">
                  <i />
                  轮询中
                </span>
              </div>
              <div class="connection-row">
                <span class="connection-icon"><Globe2 :size="17" /></span>
                <div>
                  <strong>REST API</strong>
                  <small>页面请求、登录与任务接口</small>
                </div>
                <span class="connection-state" :class="{ connected: databaseState !== 'error' }">
                  <i />{{ databaseState === 'error' ? '请求失败' : '正常' }}
                </span>
              </div>
              <div class="connection-row">
                <span class="connection-icon"><Database :size="17" /></span>
                <div>
                  <strong>SQLite 数据库</strong>
                  <small>用户、Agent、任务、事件与文章</small>
                </div>
                <span class="connection-state" :class="{ connected: databaseState === 'ok' }">
                  <i />{{ databaseState === 'ok' ? '正常' : databaseState === 'error' ? '异常' : '检测中' }}
                </span>
              </div>
            </div>
            <div class="endpoint-row">
              <span>运行模式</span>
              <code>REST / POLLING</code>
            </div>
          </div>
        </section>

        <section :ref="(el) => setSectionRef('security', el as HTMLElement | null)" class="setting-section panel">
          <div class="section-heading">
            <span class="section-icon"><ShieldCheck :size="17" /></span>
            <div>
              <h2>安全与权限</h2>
              <p>登录凭据和敏感操作的保护策略。</p>
            </div>
          </div>
          <div class="section-body security-list">
            <div>
              <span class="security-icon"><KeyRound :size="16" /></span>
              <div>
                <strong>登录密码</strong>
                <small>修改后立即生效</small>
              </div>
              <el-button size="small" @click="openPasswordDialog">修改密码</el-button>
            </div>
            <div>
              <span class="security-icon"><ShieldCheck :size="16" /></span>
              <div>
                <strong>双重验证</strong>
                <small>增强管理员账户的登录安全</small>
              </div>
              <span class="security-status"><Check :size="11" />暂未接入</span>
            </div>
          </div>
        </section>
      </main>
    </div>

    <el-dialog v-model="passwordOpen" title="修改密码" width="min(420px, 92vw)">
      <el-form label-position="top">
        <el-form-item label="当前密码">
          <el-input v-model="passwordForm.currentPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="passwordForm.newPassword" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认新密码">
          <el-input v-model="passwordForm.confirmPassword" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordOpen = false">取消</el-button>
        <el-button type="primary" :loading="passwordSaving" @click="changePassword">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.settings-layout {
  display: grid;
  align-items: start;
  gap: 18px;
  grid-template-columns: 205px minmax(0, 1fr);
}

.settings-nav {
  position: sticky;
  top: 92px;
  display: flex;
  padding: 16px 10px;
  border: 1px solid var(--line);
  border-radius: 8px;
  flex-direction: column;
  background: var(--surface);
}

.settings-nav > .panel-kicker {
  padding: 2px 9px 10px;
}

.settings-nav button {
  display: flex;
  height: 38px;
  align-items: center;
  gap: 9px;
  padding: 0 9px;
  border: 0;
  border-radius: 5px;
  color: var(--muted);
  background: transparent;
  font-size: 10px;
  text-align: left;
  cursor: pointer;
}

.settings-nav button:hover {
  background: var(--paper);
}

.settings-nav button.active {
  color: var(--ink);
  background: var(--paper-deep);
  font-weight: 650;
}

.settings-nav button span {
  width: 4px;
  height: 4px;
  margin-left: auto;
  border-radius: 50%;
}

.settings-nav button.active span {
  background: var(--green);
}

.settings-version {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 20px;
  padding: 12px 9px 2px;
  border-top: 1px solid var(--line);
  color: #9ca099;
  font-size: 7px;
}

.settings-content {
  display: flex;
  gap: 18px;
  flex-direction: column;
}

.setting-section {
  overflow: hidden;
  scroll-margin-top: 92px;
  animation: rise-in 0.45s ease both;
}

.section-heading {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--line);
}

.section-icon {
  display: grid;
  width: 31px;
  height: 31px;
  flex: 0 0 31px;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 5px;
  color: var(--green-dark);
  background: var(--paper);
}

.section-heading h2 {
  margin: 0;
  font-size: 13px;
  font-weight: 650;
}

.section-heading p {
  margin: 4px 0 0;
  color: var(--muted);
  font-size: 9px;
}

.section-body {
  padding: 20px;
}

.profile-body {
  display: flex;
  align-items: flex-start;
  gap: 22px;
}

.profile-avatar {
  display: grid;
  width: 76px;
  height: 76px;
  flex: 0 0 76px;
  place-items: center;
  border: 1px solid rgba(21, 34, 27, 0.08);
  border-radius: 7px;
  color: #2c3b1b;
  background: #dff58f;
  font-size: 25px;
  font-weight: 700;
}

.profile-fields {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 13px;
  grid-template-columns: 1fr 1fr;
}

.field {
  display: flex;
  gap: 7px;
  flex-direction: column;
}

.field label {
  color: var(--ink-2);
  font-size: 9px;
  font-weight: 600;
}

.switch-list {
  display: flex;
  flex-direction: column;
}

.switch-list > label {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 15px 0;
  border-bottom: 1px solid var(--line);
}

.switch-list > label:first-child {
  padding-top: 0;
}

.switch-list > label:last-child {
  padding-bottom: 0;
  border-bottom: 0;
}

.switch-icon {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border-radius: 5px;
  color: var(--green-dark);
  background: rgba(11, 151, 103, 0.07);
}

.switch-icon.warning {
  color: #9a6216;
  background: rgba(213, 141, 40, 0.09);
}

.switch-icon.danger {
  color: var(--danger);
  background: rgba(221, 85, 63, 0.07);
}

.switch-copy {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.switch-copy strong {
  font-size: 10px;
  font-weight: 600;
}

.switch-copy small {
  color: var(--muted);
  font-size: 8px;
}

.inline-switch {
  display: flex;
  align-items: center;
  gap: 20px;
  padding-bottom: 17px;
  border-bottom: 1px solid var(--line);
}

.form-grid {
  display: grid;
  gap: 13px;
  margin-top: 17px;
  grid-template-columns: repeat(3, 1fr);
}

.data-note {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  padding: 10px 12px;
  border-radius: 5px;
  color: var(--muted);
  background: var(--paper);
  font-size: 8px;
}

.connection-card {
  overflow: hidden;
  border: 1px solid var(--line);
  border-radius: 6px;
}

.connection-row {
  display: flex;
  min-height: 65px;
  align-items: center;
  gap: 11px;
  padding: 11px 13px;
  border-bottom: 1px solid var(--line);
}

.connection-row:last-child {
  border-bottom: 0;
}

.connection-icon {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border-radius: 5px;
  color: var(--green-dark);
  background: var(--paper);
}

.connection-row > div {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.connection-row strong {
  font-size: 10px;
}

.connection-row small {
  color: var(--muted);
  font-size: 8px;
}

.connection-state {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--amber);
  font-size: 8px;
}

.connection-state i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
}

.connection-state.connected {
  color: var(--green-dark);
}

.endpoint-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
  color: var(--muted);
  font-size: 8px;
}

.endpoint-row code {
  padding: 4px 7px;
  border: 1px solid var(--line);
  border-radius: 3px;
  color: var(--green-dark);
  background: var(--paper);
  font-family: "Bricolage Grotesque Variable", monospace;
  font-size: 8px;
}

.security-list {
  display: flex;
  flex-direction: column;
}

.security-list > div {
  display: flex;
  min-height: 66px;
  align-items: center;
  gap: 11px;
  border-bottom: 1px solid var(--line);
}

.security-list > div:first-child {
  min-height: 51px;
  padding-bottom: 14px;
}

.security-list > div:last-child {
  min-height: 51px;
  padding-top: 14px;
  border-bottom: 0;
}

.security-icon {
  display: grid;
  width: 31px;
  height: 31px;
  place-items: center;
  border-radius: 5px;
  color: var(--muted);
  background: var(--paper);
}

.security-list > div > div {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.security-list strong {
  font-size: 10px;
}

.security-list small {
  color: var(--muted);
  font-size: 8px;
}

.security-status {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--muted);
  font-size: 8px;
}

@media (max-width: 880px) {
  .settings-layout {
    grid-template-columns: 1fr;
  }

  .settings-nav {
    position: static;
    overflow-x: auto;
    padding: 8px;
    flex-direction: row;
  }

  .settings-nav > .panel-kicker,
  .settings-version,
  .settings-nav button span {
    display: none;
  }

  .settings-nav button {
    flex: 0 0 auto;
  }

  .profile-fields,
  .form-grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 760px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 560px) {
  .profile-body {
    flex-direction: column;
  }

  .profile-fields,
  .form-grid {
    width: 100%;
    grid-template-columns: 1fr;
  }

  .switch-copy small {
    line-height: 1.5;
  }
}
</style>
