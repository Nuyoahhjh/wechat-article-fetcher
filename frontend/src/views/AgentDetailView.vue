<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  CheckCircle2,
  ChevronRight,
  CircleDot,
  Clock3,
  Copy,
  Database,
  Laptop,
  Plus,
  Power,
  RefreshCw,
  Settings2,
  ShieldCheck,
  Wifi,
} from 'lucide-vue-next'
import { useAgentStore } from '@/stores/agents'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/services/api'
import { formatDateTime, formatRelative } from '@/utils/format'
import AgentAvatar from '@/components/common/AgentAvatar.vue'
import StatusPill from '@/components/common/StatusPill.vue'
import QrLoginCard from '@/components/agent-detail/QrLoginCard.vue'
import ScreenPreview from '@/components/agent-detail/ScreenPreview.vue'
import OperationLog from '@/components/agent-detail/OperationLog.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const agents = useAgentStore()
const qrcodeError = ref('')
const screenRefreshing = ref(false)
const configOpen = ref(false)
const configSaving = ref(false)
const configForm = reactive({ maxConcurrency: 1 })
let pollTimer: number | undefined

const agent = computed(() => agents.currentAgent)
const agentId = computed(() => String(route.params.id))
const terminalQr = computed(
  () =>
    agents.qrcode?.status === 'confirmed' ||
    agents.qrcode?.status === 'expired' ||
    agents.qrcode?.status === 'failed' ||
    agents.qrcode?.status === 'canceled',
)

async function loadAgent(showError = true) {
  try {
    await agents.fetchAgentDetail(agentId.value)
  } catch (error) {
    if (showError) {
      ElMessage.error(error instanceof Error ? error.message : 'Agent 详情加载失败')
    }
  }
}

async function refreshQrCode(showMessage = true) {
  qrcodeError.value = ''
  try {
    await agents.refreshQrCode()
    if (showMessage) ElMessage.success('已请求 Agent 上传微信登录画面')
  } catch (error) {
    qrcodeError.value = error instanceof Error ? error.message : '登录请求失败'
    ElMessage.error(qrcodeError.value)
  }
}

async function refreshScreen() {
  screenRefreshing.value = true
  try {
    agents.screen = await api.agents.screen(agentId.value)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '截图刷新失败')
  } finally {
    screenRefreshing.value = false
  }
}

function openConfig() {
  if (!agent.value) return
  configForm.maxConcurrency = agent.value.config.maxConcurrency
  configOpen.value = true
}

async function saveConfig() {
  if (!agent.value) return
  configSaving.value = true
  try {
    await agents.updateConfig(configForm.maxConcurrency, agent.value.rawConfig)
    configOpen.value = false
    ElMessage.success('Agent 配置已更新')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '配置更新失败')
  } finally {
    configSaving.value = false
  }
}

async function disableAgent() {
  if (!agent.value) return
  try {
    await ElMessageBox.confirm(
      `禁用 ${agent.value.name}？平台会断开其 WebSocket，历史任务和文章会保留。`,
      '禁用 Agent',
      { type: 'warning', confirmButtonText: '禁用', cancelButtonText: '取消' },
    )
    await agents.disableAgent()
    ElMessage.success('Agent 已禁用')
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : '禁用 Agent 失败')
  }
}

async function copyAgentId() {
  if (!agent.value) return
  await navigator.clipboard.writeText(agent.value.id)
  ElMessage.success('Agent ID 已复制')
}

function createTask() {
  router.push({ name: 'tasks', query: { create: '1', agentId: agentId.value } })
}

onMounted(async () => {
  await loadAgent()
  pollTimer = window.setInterval(async () => {
    if (document.visibilityState !== 'visible') return
    await loadAgent(false)
    if (agents.qrcode && !terminalQr.value) {
      try {
        await agents.refreshLoginStatus()
      } catch {
        // Keep the last known QR state; the next poll retries.
      }
    }
  }, 5000)
})

onBeforeUnmount(() => window.clearInterval(pollTimer))
</script>

<template>
  <div v-loading="agents.detailLoading" class="page-shell detail-page">
    <template v-if="agent">
      <nav class="breadcrumb">
        <button @click="router.push({ name: 'agents' })"><ArrowLeft :size="13" />Agent 网络</button>
        <ChevronRight :size="11" />
        <span>{{ agent.code }}</span>
      </nav>

      <section class="agent-hero">
        <div class="hero-main">
          <div class="hero-identity">
            <div class="avatar-wrap">
              <AgentAvatar :name="agent.name" :size="58" />
              <span class="agent-status-dot" :class="agent.status" />
            </div>
            <div>
              <div class="hero-name-line">
                <h1>{{ agent.name }}</h1>
                <StatusPill :status="agent.status" />
              </div>
              <p>
                <span class="mono">{{ agent.code }}</span>
                <i />
                {{ agent.hostname || '未知主机' }}
                <i />
                微信：{{ agent.wechat.nickname || '未登录' }}
              </p>
            </div>
          </div>
          <div class="hero-actions">
            <button
              v-if="auth.user?.role === 'admin' && agent.status !== 'disabled'"
              class="hero-icon"
              title="禁用 Agent"
              @click="disableAgent"
            >
              <Power :size="16" />
            </button>
            <button class="hero-icon" title="Agent 配置" @click="openConfig">
              <Settings2 :size="16" />
            </button>
            <button class="hero-secondary" @click="loadAgent()"><RefreshCw :size="14" />刷新状态</button>
            <button class="hero-primary" @click="createTask"><Plus :size="15" />分配任务</button>
          </div>
        </div>
        <div class="hero-footer">
          <div class="hero-signal">
            <span class="signal-ring"><i /></span>
            <div>
              <span>最近心跳</span>
              <strong>{{ formatRelative(agent.lastHeartbeat) }}</strong>
            </div>
          </div>
          <div class="resource-metric">
            <span><Wifi :size="12" />微信登录</span>
            <strong>{{ agent.wechat.loginStatus === 'logged_in' ? '有效' : '未登录' }}</strong>
          </div>
          <div class="resource-metric">
            <span><CircleDot :size="12" />活跃任务</span>
            <strong>{{ agent.activeTasks }}</strong>
          </div>
          <div class="resource-metric">
            <span><Database :size="12" />并发槽位</span>
            <strong>{{ agent.config.maxConcurrency }}</strong>
          </div>
          <div class="resource-metric">
            <span><Clock3 :size="12" />配置版本</span>
            <strong class="mono">v{{ agent.configVersion }}</strong>
          </div>
          <div class="node-id">
            <span>AGENT ID</span>
            <button class="mono" @click="copyAgentId">{{ agent.id }} <Copy :size="10" /></button>
          </div>
        </div>
      </section>

      <div class="detail-grid">
        <div class="detail-main">
          <QrLoginCard
            :agent="agent"
            :qrcode="agents.qrcode"
            :loading="agents.qrcodeLoading"
            @refresh="refreshQrCode()"
          />
          <div v-if="qrcodeError" class="inline-error">{{ qrcodeError }}</div>
          <ScreenPreview
            :agent="agent"
            :screen="agents.screen"
            :loading="screenRefreshing"
            @refresh="refreshScreen"
          />
          <OperationLog :logs="agents.logs" />
        </div>

        <aside class="detail-side">
          <section class="account-panel panel">
            <div class="panel-head">
              <div>
                <span class="panel-kicker">WECHAT IDENTITY</span>
                <h2 class="panel-title">微信账号</h2>
              </div>
              <ShieldCheck :size="17" class="verified-icon" />
            </div>
            <div class="wechat-profile">
              <AgentAvatar :name="agent.wechat.nickname || agent.name" :size="52" />
              <div>
                <strong>{{ agent.wechat.nickname || '未识别账号' }}</strong>
                <span class="mono">{{ agent.wechat.accountId || agent.hostname }}</span>
              </div>
              <span class="login-chip" :class="agent.wechat.loginStatus">
                <CheckCircle2 v-if="agent.wechat.loginStatus === 'logged_in'" :size="11" />
                {{ agent.wechat.loginStatus === 'logged_in' ? '登录有效' : '需要登录' }}
              </span>
            </div>
            <div class="account-meta">
              <div>
                <span>最近在线</span>
                <strong>{{ formatDateTime(agent.wechat.lastSeenAt) }}</strong>
              </div>
              <div>
                <span>会话状态</span>
                <strong>{{ agent.wechat.loginStatus === 'logged_in' ? '保持中' : '未建立' }}</strong>
              </div>
            </div>
          </section>

          <section class="runtime-panel panel">
            <div class="panel-head">
              <div>
                <span class="panel-kicker">RUNTIME</span>
                <h2 class="panel-title">运行环境</h2>
              </div>
              <Settings2 :size="16" class="muted-icon" />
            </div>
            <div class="runtime-list">
              <div>
                <span><Laptop :size="13" />操作系统</span>
                <strong>{{ agent.os || '未上报' }}</strong>
              </div>
              <div>
                <span><Database :size="13" />主机名</span>
                <strong class="mono">{{ agent.hostname || '未知' }}</strong>
              </div>
              <div>
                <span><CircleDot :size="13" />Agent 版本</span>
                <strong class="mono">{{ agent.version ? `v${agent.version}` : '未上报' }}</strong>
              </div>
              <div>
                <span><Clock3 :size="13" />心跳间隔</span>
                <strong class="mono">{{ agent.config.heartbeatInterval }}s</strong>
              </div>
              <div>
                <span><Database :size="13" />最大并发</span>
                <strong class="mono">{{ agent.config.maxConcurrency }}</strong>
              </div>
              <div>
                <span><Wifi :size="13" />连接状态</span>
                <strong>{{ agent.status === 'online' || agent.status === 'busy' ? 'WebSocket 已连接' : '未连接' }}</strong>
              </div>
            </div>
            <div class="capability-list">
              <span v-for="capability in agent.capabilities" :key="capability">{{ capability }}</span>
            </div>
          </section>

          <section class="run-card">
            <div class="run-card-head">
              <span class="panel-kicker">RUN STATUS</span>
              <span class="signal-dot" />
            </div>
            <strong class="mono">{{ agent.activeTasks }}</strong>
            <p>个任务正在此节点执行</p>
            <div class="run-slots">
              <i
                v-for="slot in agent.config.maxConcurrency"
                :key="slot"
                :class="{ active: slot <= agent.activeTasks }"
              />
            </div>
            <div class="run-meta">
              <span>可用槽位</span>
              <span class="mono">{{ Math.max(0, agent.config.maxConcurrency - agent.activeTasks) }} / {{ agent.config.maxConcurrency }}</span>
            </div>
            <button @click="createTask">分配新任务 <ChevronRight :size="13" /></button>
          </section>
        </aside>
      </div>
    </template>

    <el-dialog v-model="configOpen" title="Agent 配置" width="min(420px, 92vw)">
      <el-form label-position="top">
        <el-form-item label="最大并发任务数">
          <el-input-number v-model="configForm.maxConcurrency" :min="1" :max="20" style="width: 100%" />
        </el-form-item>
      </el-form>
      <p class="config-note">该值会被服务端调度器使用，并实时下发给在线 Agent。</p>
      <template #footer>
        <el-button @click="configOpen = false">取消</el-button>
        <el-button type="primary" :loading="configSaving" @click="saveConfig">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.detail-page {
  min-height: 500px;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 14px;
  color: var(--muted);
  font-size: 9px;
}

.breadcrumb button {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 0;
  border: 0;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.breadcrumb button:hover {
  color: var(--ink);
}

.agent-hero {
  overflow: hidden;
  margin-bottom: 18px;
  border-radius: 8px;
  color: #eef2ec;
  background: var(--ink);
  animation: rise-in 0.4s ease both;
}

.hero-main {
  display: flex;
  min-height: 115px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px 18px;
}

.hero-identity {
  display: flex;
  align-items: center;
  gap: 14px;
}

.avatar-wrap {
  position: relative;
  display: inline-flex;
  flex: 0 0 auto;
}

.avatar-wrap > .agent-status-dot {
  position: absolute;
  right: -4px;
  bottom: -4px;
  width: 12px;
  height: 12px;
  border: 3px solid var(--ink);
  border-radius: 50%;
  background: var(--acid);
}

.avatar-wrap > .agent-status-dot.error {
  background: var(--amber);
}

.avatar-wrap > .agent-status-dot.offline,
.avatar-wrap > .agent-status-dot.disabled {
  background: #7f8981;
}

.hero-name-line {
  display: flex;
  align-items: center;
  gap: 10px;
}

.hero-name-line h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 650;
  letter-spacing: -0.03em;
}

.hero-identity p {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 8px 0 0;
  color: rgba(255, 255, 255, 0.4);
  font-size: 9px;
  flex-wrap: wrap;
}

.hero-identity p i {
  width: 2px;
  height: 2px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.28);
}

.hero-actions {
  display: flex;
  align-items: center;
  gap: 7px;
}

.hero-icon,
.hero-secondary,
.hero-primary {
  display: flex;
  height: 34px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border-radius: 5px;
  font-size: 9px;
  font-weight: 600;
  cursor: pointer;
}

.hero-icon {
  width: 34px;
  padding: 0;
  border: 1px solid rgba(255, 255, 255, 0.13);
  color: rgba(255, 255, 255, 0.62);
  background: rgba(255, 255, 255, 0.04);
}

.hero-secondary {
  border: 1px solid rgba(255, 255, 255, 0.14);
  color: #eef2ec;
  background: transparent;
}

.hero-primary {
  border: 1px solid var(--acid);
  color: var(--ink);
  background: var(--acid);
}

.hero-footer {
  display: grid;
  min-height: 85px;
  border-top: 1px solid rgba(255, 255, 255, 0.09);
  grid-template-columns: 1.1fr repeat(4, 0.85fr) 1.2fr;
}

.hero-signal,
.resource-metric,
.node-id {
  display: flex;
  justify-content: center;
  padding: 12px 17px;
  border-right: 1px solid rgba(255, 255, 255, 0.075);
  flex-direction: column;
}

.hero-signal {
  flex-direction: row;
  align-items: center;
  justify-content: flex-start;
  gap: 11px;
}

.signal-ring {
  display: grid;
  width: 29px;
  height: 29px;
  place-items: center;
  border: 1px solid rgba(201, 243, 106, 0.18);
  border-radius: 50%;
}

.signal-ring i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--acid);
  box-shadow: 0 0 0 4px rgba(201, 243, 106, 0.07);
}

.hero-signal > div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.hero-signal span,
.resource-metric > span,
.node-id > span {
  color: rgba(255, 255, 255, 0.32);
  font-size: 8px;
}

.hero-signal strong {
  font-size: 10px;
  font-weight: 550;
}

.resource-metric {
  gap: 5px;
}

.resource-metric > span {
  display: flex;
  align-items: center;
  gap: 5px;
}

.resource-metric strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-id {
  border-right: 0;
  gap: 4px;
}

.node-id button {
  display: flex;
  align-items: center;
  gap: 4px;
  overflow: hidden;
  padding: 0;
  border: 0;
  color: rgba(255, 255, 255, 0.66);
  background: transparent;
  font-size: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.detail-grid {
  display: grid;
  align-items: start;
  gap: 18px;
  grid-template-columns: minmax(0, 1.65fr) minmax(295px, 0.65fr);
}

.detail-main,
.detail-side {
  display: flex;
  min-width: 0;
  gap: 18px;
  flex-direction: column;
}

.account-panel,
.runtime-panel {
  overflow: hidden;
}

.verified-icon {
  color: var(--green);
}

.muted-icon {
  color: var(--muted);
}

.wechat-profile {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 17px;
  border-bottom: 1px solid var(--line);
}

.wechat-profile > div {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.wechat-profile strong {
  overflow: hidden;
  font-size: 11px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wechat-profile span {
  color: var(--muted);
  font-size: 8px;
}

.login-chip {
  display: inline-flex !important;
  align-items: center;
  gap: 4px;
  padding: 4px 6px;
  border-radius: 999px;
  color: var(--green-dark) !important;
  background: rgba(11, 151, 103, 0.07);
  font-size: 7px !important;
  white-space: nowrap;
}

.login-chip.logged_out,
.login-chip.unknown {
  color: var(--orange) !important;
  background: rgba(239, 113, 77, 0.08);
}

.account-meta {
  display: grid;
  gap: 1px;
  padding: 13px;
  background: var(--line);
  grid-template-columns: 1fr 1fr;
}

.account-meta div {
  display: flex;
  padding: 9px;
  flex-direction: column;
  gap: 5px;
  background: var(--surface);
}

.account-meta span {
  color: var(--muted);
  font-size: 8px;
}

.account-meta strong {
  font-size: 9px;
  font-weight: 600;
}

.runtime-list {
  display: flex;
  flex-direction: column;
}

.runtime-list > div {
  display: flex;
  min-height: 43px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0 16px;
  border-bottom: 1px solid var(--line);
}

.runtime-list span {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
  font-size: 8px;
}

.runtime-list strong {
  overflow: hidden;
  font-size: 8px;
  font-weight: 600;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.capability-list {
  display: flex;
  gap: 5px;
  padding: 12px 15px;
  flex-wrap: wrap;
}

.capability-list span {
  padding: 4px 6px;
  border: 1px solid var(--line);
  border-radius: 3px;
  color: var(--muted);
  background: var(--paper);
  font-size: 7px;
}

.run-card {
  padding: 17px;
  border-radius: 8px;
  color: #f1f4ed;
  background: var(--ink);
}

.run-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.run-card-head .panel-kicker {
  color: rgba(255, 255, 255, 0.3);
}

.run-card > strong {
  display: block;
  margin-top: 10px;
  color: var(--acid);
  font-size: 34px;
  font-weight: 500;
  letter-spacing: -0.05em;
  line-height: 1;
}

.run-card > p {
  margin: 5px 0 14px;
  color: rgba(255, 255, 255, 0.37);
  font-size: 8px;
}

.run-slots {
  display: flex;
  gap: 4px;
  height: 6px;
}

.run-slots i {
  flex: 1;
  border-radius: 1px;
  background: rgba(255, 255, 255, 0.1);
}

.run-slots i.active {
  background: var(--acid);
}

.run-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  color: rgba(255, 255, 255, 0.31);
  font-size: 7px;
}

.run-card button {
  display: flex;
  width: 100%;
  height: 31px;
  align-items: center;
  justify-content: space-between;
  margin-top: 15px;
  padding: 0 10px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 4px;
  color: rgba(255, 255, 255, 0.65);
  background: rgba(255, 255, 255, 0.03);
  font-size: 8px;
  cursor: pointer;
}

.inline-error {
  margin-top: -10px;
  padding: 9px 11px;
  border: 1px solid rgba(221, 85, 63, 0.18);
  border-radius: 5px;
  color: var(--danger);
  background: rgba(221, 85, 63, 0.06);
  font-size: 8px;
}

.config-note {
  margin: 0;
  color: var(--muted);
  font-size: 9px;
  line-height: 1.6;
}

@media (max-width: 1180px) {
  .detail-grid {
    grid-template-columns: 1fr;
  }

  .detail-side {
    display: grid;
    align-items: start;
    grid-template-columns: 1fr 1fr;
  }

  .run-card {
    grid-column: span 2;
  }

  .hero-footer {
    grid-template-columns: 1.1fr repeat(4, 0.9fr);
  }

  .node-id {
    display: none;
  }
}

@media (max-width: 760px) {
  .hero-main {
    align-items: flex-start;
    flex-direction: column;
  }

  .hero-actions {
    width: 100%;
  }

  .hero-secondary,
  .hero-primary {
    height: 44px;
    flex: 1;
  }

  .hero-icon {
    width: 44px;
    height: 44px;
  }

  .hero-footer {
    grid-template-columns: repeat(2, 1fr);
  }

  .hero-signal {
    grid-column: span 2;
    border-bottom: 1px solid rgba(255, 255, 255, 0.075);
  }

  .resource-metric:nth-child(4) {
    border-right: 0;
  }

  .detail-side {
    display: flex;
    align-items: stretch;
  }

  .detail-side > *,
  .run-card {
    width: 100%;
    grid-column: auto;
  }

  .runtime-list span,
  .runtime-list strong {
    font-size: 10px;
  }

  .hero-identity p {
    font-size: 10px;
  }
}

@media (max-width: 480px) {
  .hero-identity {
    align-items: flex-start;
  }

  .hero-name-line {
    align-items: flex-start;
    flex-direction: column;
  }

  .hero-name-line h1 {
    font-size: 21px;
  }
}
</style>
