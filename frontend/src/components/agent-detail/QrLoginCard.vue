<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import dayjs from 'dayjs'
import {
  CheckCircle2,
  Clock3,
  QrCode,
  RefreshCw,
  ScanLine,
  Smartphone,
} from 'lucide-vue-next'
import type { Agent, QrCodePayload } from '@/types'

const props = defineProps<{
  agent: Agent
  qrcode: QrCodePayload | null
  loading: boolean
}>()

const emit = defineEmits<{
  refresh: []
}>()

const now = ref(dayjs())
let timer: number | undefined

const secondsLeft = computed(() => {
  if (!props.qrcode) return 0
  return Math.max(0, dayjs(props.qrcode.expiresAt).diff(now.value, 'second'))
})

const countdown = computed(() => {
  const minutes = Math.floor(secondsLeft.value / 60)
  const seconds = secondsLeft.value % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
})

const statusLabel = computed(() => {
  if (props.agent.status === 'offline' || props.agent.status === 'disabled') return 'Agent 不可用'
  if (!props.qrcode) return '等待生成'
  if (props.qrcode.status === 'confirmed') return '登录已确认'
  if (props.qrcode.status === 'failed') return props.qrcode.message || '获取登录画面失败'
  if (props.qrcode.status === 'canceled') return '登录会话已取消'
  if (props.qrcode.status === 'expired' || secondsLeft.value === 0) return '登录会话已过期'
  if (props.qrcode.status === 'scanned') return '已扫码，请在手机确认'
  if (props.qrcode.status === 'qr_ready') return '登录画面已上传'
  return '等待 Agent 上传登录画面'
})

onMounted(() => {
  timer = window.setInterval(() => {
    now.value = dayjs()
  }, 1000)
})

onBeforeUnmount(() => window.clearInterval(timer))
</script>

<template>
  <section class="qr-card panel">
    <div class="panel-head">
      <div>
        <span class="panel-kicker">WECHAT AUTH</span>
        <h2 class="panel-title">微信登录</h2>
      </div>
      <button
        class="icon-button"
        title="刷新二维码"
        :disabled="loading || agent.status === 'offline' || agent.status === 'disabled'"
        @click="emit('refresh')"
      >
        <RefreshCw :size="15" :class="{ spinning: loading }" />
      </button>
    </div>

    <div class="qr-body">
      <div class="qr-frame" :class="{ disabled: agent.status === 'offline' }">
        <div class="qr-corners">
          <i v-for="n in 4" :key="n" />
        </div>
        <img v-if="qrcode?.imageUrl" :src="qrcode.imageUrl" alt="微信登录画面" />
        <div v-else class="qr-placeholder">
          <QrCode :size="46" :stroke-width="1.1" />
          <span>{{ agent.status === 'offline' ? '节点离线' : '登录画面加载中' }}</span>
        </div>
        <div v-if="qrcode && (secondsLeft === 0 || qrcode.status === 'failed' || qrcode.status === 'expired')" class="qr-expired">
          <Clock3 :size="24" />
          <strong>{{ qrcode.status === 'failed' ? '获取登录画面失败' : '登录会话已过期' }}</strong>
          <button @click="emit('refresh')">重新请求</button>
        </div>
      </div>

      <div class="qr-side">
        <span class="qr-status" :class="{ active: qrcode && secondsLeft > 0, expired: qrcode && secondsLeft === 0 }">
          <i />
          {{ statusLabel }}
        </span>
        <div class="wechat-account">
          <span>当前微信账号</span>
          <strong>{{ agent.wechat.nickname || '未识别' }}</strong>
          <small class="mono">{{ agent.wechat.accountId || agent.hostname }}</small>
        </div>
        <div class="expiry">
          <Clock3 :size="13" />
          <span>剩余有效时间</span>
          <strong class="mono">{{ countdown }}</strong>
        </div>
        <div class="scan-steps">
          <div>
            <span class="mono">01</span>
            <p><Smartphone :size="13" />打开手机微信</p>
          </div>
          <div>
            <span class="mono">02</span>
            <p><ScanLine :size="13" />在 Agent 桌面完成确认</p>
          </div>
          <div>
            <span class="mono">03</span>
            <p><CheckCircle2 :size="13" />在手机端确认登录</p>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.qr-card {
  overflow: hidden;
}

.qr-body {
  display: grid;
  gap: 22px;
  padding: 22px;
  grid-template-columns: 194px minmax(0, 1fr);
}

.qr-frame {
  position: relative;
  display: grid;
  width: 194px;
  height: 194px;
  padding: 11px;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: #fff;
}

.qr-frame.disabled {
  opacity: 0.58;
}

.qr-frame img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.qr-corners i {
  position: absolute;
  width: 13px;
  height: 13px;
  border-color: var(--green);
}

.qr-corners i:nth-child(1) {
  top: -1px;
  left: -1px;
  border-top: 2px solid;
  border-left: 2px solid;
}

.qr-corners i:nth-child(2) {
  top: -1px;
  right: -1px;
  border-top: 2px solid;
  border-right: 2px solid;
}

.qr-corners i:nth-child(3) {
  bottom: -1px;
  left: -1px;
  border-bottom: 2px solid;
  border-left: 2px solid;
}

.qr-corners i:nth-child(4) {
  right: -1px;
  bottom: -1px;
  border-right: 2px solid;
  border-bottom: 2px solid;
}

.qr-placeholder {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-direction: column;
  color: #a2a69f;
}

.qr-placeholder span {
  font-size: 9px;
}

.qr-expired {
  position: absolute;
  inset: 11px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  color: #f1f4ed;
  background: rgba(21, 34, 27, 0.91);
  backdrop-filter: blur(3px);
}

.qr-expired strong {
  margin-top: 8px;
  font-size: 11px;
}

.qr-expired button {
  margin-top: 10px;
  padding: 6px 10px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 4px;
  color: var(--acid);
  background: transparent;
  font-size: 8px;
  cursor: pointer;
}

.qr-side {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.qr-status {
  display: inline-flex;
  width: fit-content;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  border: 1px solid var(--line);
  border-radius: 999px;
  color: var(--muted);
  font-size: 8px;
}

.qr-status i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #a9aca7;
}

.qr-status.active {
  border-color: rgba(11, 151, 103, 0.2);
  color: var(--green-dark);
  background: rgba(11, 151, 103, 0.055);
}

.qr-status.active i {
  background: var(--green);
  animation: blink 1.5s infinite;
}

.qr-status.expired {
  color: var(--orange);
}

.qr-status.expired i {
  background: var(--orange);
}

.wechat-account {
  display: flex;
  margin-top: 14px;
  flex-direction: column;
  gap: 4px;
}

.wechat-account span {
  color: var(--muted);
  font-size: 8px;
}

.wechat-account strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wechat-account small {
  color: #969a94;
  font-size: 8px;
}

.expiry {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 13px;
  padding: 8px 9px;
  border-radius: 5px;
  color: var(--muted);
  background: var(--paper);
}

.expiry span {
  flex: 1;
  font-size: 8px;
}

.expiry strong {
  color: var(--ink);
  font-size: 9px;
}

.scan-steps {
  display: flex;
  gap: 8px;
  margin-top: auto;
  padding-top: 14px;
  flex-direction: column;
}

.scan-steps > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.scan-steps .mono {
  color: #a0a49d;
  font-size: 7px;
}

.scan-steps p {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 0;
  color: var(--muted);
  font-size: 8px;
}

.spinning {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes blink {
  50% {
    opacity: 0.35;
  }
}

@media (max-width: 660px) {
  .qr-body {
    justify-items: center;
    grid-template-columns: 1fr;
  }

  .qr-side {
    width: 100%;
  }

  .scan-steps {
    margin-top: 10px;
  }
}
</style>
