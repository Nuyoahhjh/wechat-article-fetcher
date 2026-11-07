<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  ArrowRight,
  Eye,
  EyeOff,
  Fingerprint,
  LockKeyhole,
  Mail,
  ShieldCheck,
} from 'lucide-vue-next'
import BrandMark from '@/components/common/BrandMark.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const showPassword = ref(false)
const form = reactive({
  email: '',
  password: '',
})

async function submit() {
  if (!form.email || !form.password) {
    ElMessage.warning('请输入账号和密码')
    return
  }
  try {
    await auth.login({ email: form.email, password: form.password })
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '登录失败，请稍后重试')
  }
}
</script>

<template>
  <div class="login-page">
    <section class="signal-side">
      <div class="signal-side-inner">
        <header class="login-brand">
          <BrandMark inverse />
          <span class="build mono">V3 / WINDOWS AGENT</span>
        </header>

        <div class="signal-copy">
          <p class="signal-eyebrow"><i /> OPERATIONS FOR PUBLISHING INTELLIGENCE</p>
          <h1>让每一个采集节点，<br />都留在视野之内。</h1>
          <p class="signal-description">
            统一编排 Windows Agent，持续获取公众号文章，<br class="desktop-only" />
            让登录状态、任务进度与内容资产保持同步。
          </p>
        </div>

        <div class="network-visual">
          <div class="network-grid" />
          <div class="network-axis axis-x" />
          <div class="network-axis axis-y" />
          <div class="node node-a">
            <span class="node-ring" />
            <i />
            <strong>Windows Agent</strong>
            <small>DESKTOP</small>
          </div>
          <div class="node node-b">
            <span class="node-ring" />
            <i />
            <strong>微信登录</strong>
            <small>SESSION</small>
          </div>
          <div class="node node-c">
            <span class="node-ring" />
            <i />
            <strong>文章入库</strong>
            <small>ARCHIVE</small>
          </div>
          <div class="network-center">
            <span class="center-pulse" />
            <Fingerprint :size="28" :stroke-width="1.2" />
            <strong>拾刊</strong>
            <small>CONTROL</small>
          </div>
          <span class="coord coord-1 mono">CONTROLLED AUTOMATION</span>
          <span class="coord coord-2 mono">WINDOWS / WECHAT</span>
        </div>

        <footer class="signal-footer">
          <div>
            <span>多节点接入</span>
            <small>AGENT REGISTRATION</small>
          </div>
          <div>
            <span>扫码授权</span>
            <small>WECHAT LOGIN</small>
          </div>
          <div>
            <span>真实入库</span>
            <small>ARTICLE ARCHIVE</small>
          </div>
          <span class="footer-code mono">V3 / READY</span>
        </footer>
      </div>
    </section>

    <section class="form-side">
      <div class="form-wrap">
        <div class="form-topline">
          <span class="mono">SECURE ACCESS / 01</span>
          <span class="secure-note"><ShieldCheck :size="13" /> 管理端</span>
        </div>

        <div class="form-heading">
          <p class="eyebrow">WELCOME BACK</p>
          <h2>登录工作台</h2>
          <p>使用平台账号继续管理采集网络。</p>
        </div>

        <form @submit.prevent="submit">
          <label class="field-label" for="email">账号</label>
          <div class="field-control">
            <Mail :size="17" :stroke-width="1.7" />
            <input
              id="email"
              v-model="form.email"
              type="text"
              autocomplete="username"
              placeholder="name@company.com"
            />
          </div>

          <div class="password-label">
            <label class="field-label" for="password">密码</label>
            <span />
          </div>
          <div class="field-control">
            <LockKeyhole :size="17" :stroke-width="1.7" />
            <input
              id="password"
              v-model="form.password"
              :type="showPassword ? 'text' : 'password'"
              autocomplete="current-password"
              placeholder="输入登录密码"
            />
            <button
              type="button"
              class="password-toggle"
              :title="showPassword ? '隐藏密码' : '显示密码'"
              @click="showPassword = !showPassword"
            >
              <EyeOff v-if="showPassword" :size="16" />
              <Eye v-else :size="16" />
            </button>
          </div>

          <button class="submit-button" type="submit" :disabled="auth.loading">
            <span>{{ auth.loading ? '正在验证...' : '进入控制台' }}</span>
            <ArrowRight v-if="!auth.loading" :size="18" />
            <i v-else />
          </button>
        </form>

        <footer class="form-footer">
          <span>数据仅用于内部内容运营</span>
          <span class="mono">v0.1.0</span>
        </footer>
      </div>
    </section>
  </div>
</template>

<style scoped>
.login-page {
  display: grid;
  min-height: 100vh;
  grid-template-columns: minmax(0, 1.45fr) minmax(420px, 0.95fr);
  background: var(--paper);
}

.signal-side {
  position: relative;
  min-height: 100vh;
  overflow: hidden;
  color: #f1f4ed;
  background: var(--ink);
}

.signal-side::after {
  position: absolute;
  inset: 0;
  content: "";
  pointer-events: none;
  background:
    linear-gradient(90deg, rgba(255, 255, 255, 0.028) 1px, transparent 1px),
    linear-gradient(rgba(255, 255, 255, 0.022) 1px, transparent 1px);
  background-size: 64px 64px;
  mask-image: linear-gradient(to bottom, transparent 0%, #000 20%, #000 84%, transparent 100%);
}

.signal-side-inner {
  position: relative;
  z-index: 1;
  display: flex;
  width: 100%;
  max-width: 980px;
  min-height: 100vh;
  margin: 0 auto;
  padding: 30px clamp(32px, 5vw, 74px) 24px;
  flex-direction: column;
}

.login-brand {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.build {
  color: rgba(241, 244, 237, 0.28);
  font-size: 8px;
  font-weight: 650;
  letter-spacing: 0.15em;
}

.signal-copy {
  position: relative;
  z-index: 2;
  margin-top: clamp(60px, 8vh, 105px);
}

.signal-eyebrow {
  display: flex;
  align-items: center;
  gap: 9px;
  margin: 0 0 22px;
  color: rgba(241, 244, 237, 0.47);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 9px;
  font-weight: 650;
  letter-spacing: 0.19em;
}

.signal-eyebrow i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--acid);
  box-shadow: 0 0 0 4px rgba(201, 243, 106, 0.08);
}

h1 {
  max-width: 700px;
  margin: 0;
  font-family: "Microsoft YaHei UI", "Microsoft YaHei", sans-serif;
  font-size: clamp(38px, 4.2vw, 64px);
  font-weight: 650;
  letter-spacing: -0.055em;
  line-height: 1.2;
}

.signal-description {
  margin: 24px 0 0;
  color: rgba(241, 244, 237, 0.5);
  font-size: 13px;
  line-height: 1.9;
}

.network-visual {
  position: relative;
  width: min(660px, 88%);
  height: 270px;
  margin: 24px auto 0;
}

.network-grid {
  position: absolute;
  inset: 30px 0 20px;
  border: 1px solid rgba(255, 255, 255, 0.07);
  background-image:
    radial-gradient(rgba(255, 255, 255, 0.12) 1px, transparent 1px);
  background-size: 17px 17px;
  opacity: 0.34;
  mask-image: radial-gradient(ellipse at center, #000 10%, transparent 68%);
}

.network-axis {
  position: absolute;
  background: rgba(201, 243, 106, 0.18);
}

.axis-x {
  top: 50%;
  left: 10%;
  width: 80%;
  height: 1px;
}

.axis-y {
  top: 12%;
  left: 50%;
  width: 1px;
  height: 76%;
}

.network-center {
  position: absolute;
  top: 50%;
  left: 50%;
  display: flex;
  width: 112px;
  height: 112px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  border: 1px solid rgba(201, 243, 106, 0.42);
  border-radius: 50%;
  color: var(--acid);
  background: rgba(201, 243, 106, 0.05);
  transform: translate(-50%, -50%);
  backdrop-filter: blur(3px);
}

.network-center strong {
  margin-top: 4px;
  color: #f1f4ed;
  font-size: 12px;
  letter-spacing: 0.12em;
}

.network-center small {
  margin-top: 2px;
  color: rgba(241, 244, 237, 0.33);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 7px;
  letter-spacing: 0.18em;
}

.center-pulse {
  position: absolute;
  inset: -12px;
  border: 1px solid rgba(201, 243, 106, 0.12);
  border-radius: 50%;
  animation: center-pulse 3s ease-out infinite;
}

.node {
  position: absolute;
  display: flex;
  min-width: 122px;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid rgba(255, 255, 255, 0.09);
  background: rgba(10, 16, 12, 0.82);
  backdrop-filter: blur(5px);
}

.node::after {
  position: absolute;
  height: 1px;
  content: "";
  background: rgba(201, 243, 106, 0.2);
}

.node i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--acid);
}

.node strong {
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 10px;
  font-weight: 650;
}

.node small {
  margin-left: auto;
  color: rgba(241, 244, 237, 0.3);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 6px;
  letter-spacing: 0.1em;
}

.node-a {
  top: 26px;
  left: 10%;
}

.node-a::after {
  top: 50%;
  left: 100%;
  width: 210px;
  transform: rotate(23deg);
  transform-origin: left;
}

.node-b {
  top: 45px;
  right: 7%;
}

.node-b i {
  background: #e3a84a;
}

.node-b::after {
  top: 50%;
  right: 100%;
  width: 190px;
  transform: rotate(-19deg);
  transform-origin: right;
}

.node-c {
  bottom: 25px;
  left: 25%;
}

.node-c::after {
  right: 100%;
  bottom: 50%;
  width: 120px;
  transform: rotate(-35deg);
  transform-origin: right;
}

.node-ring {
  position: absolute;
  left: 7px;
  width: 9px;
  height: 9px;
  border: 1px solid rgba(201, 243, 106, 0.22);
  border-radius: 50%;
}

.coord {
  position: absolute;
  color: rgba(241, 244, 237, 0.22);
  font-size: 7px;
  letter-spacing: 0.1em;
}

.coord-1 {
  bottom: 20px;
  left: 0;
}

.coord-2 {
  right: 0;
  bottom: 20px;
}

.signal-footer {
  display: flex;
  align-items: flex-end;
  gap: clamp(26px, 5vw, 70px);
  margin-top: auto;
  padding-top: 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.09);
}

.signal-footer div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.signal-footer div span {
  color: #eef3ec;
  font-size: 18px;
  font-weight: 550;
  letter-spacing: -0.02em;
}

.signal-footer div small {
  color: rgba(241, 244, 237, 0.28);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 7px;
  font-weight: 650;
  letter-spacing: 0.14em;
}

.footer-code {
  margin-left: auto;
  color: var(--acid);
  font-size: 8px;
  letter-spacing: 0.13em;
}

.form-side {
  position: relative;
  display: flex;
  min-height: 100vh;
  align-items: center;
  justify-content: center;
  background:
    linear-gradient(90deg, rgba(21, 34, 27, 0.035) 1px, transparent 1px),
    var(--paper);
  background-size: 86px 100%;
}

.form-side::before {
  position: absolute;
  top: 0;
  left: 0;
  width: 5px;
  height: 100%;
  content: "";
  background: var(--acid);
}

.form-wrap {
  width: min(390px, calc(100% - 64px));
  padding: 44px 0 34px;
}

.form-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: clamp(60px, 10vh, 110px);
  color: var(--muted);
  font-size: 8px;
  font-weight: 650;
  letter-spacing: 0.13em;
}

.secure-note {
  display: flex;
  align-items: center;
  gap: 5px;
  letter-spacing: 0;
}

.form-heading h2 {
  margin: 0;
  font-size: 30px;
  font-weight: 680;
  letter-spacing: -0.04em;
}

.form-heading > p:last-child {
  margin: 10px 0 32px;
  color: var(--muted);
  font-size: 12px;
}

.field-label {
  display: block;
  margin: 0 0 8px;
  color: var(--ink-2);
  font-size: 11px;
  font-weight: 650;
}

.field-control {
  display: flex;
  height: 47px;
  align-items: center;
  gap: 10px;
  margin-bottom: 20px;
  padding: 0 13px;
  border: 1px solid var(--line);
  border-radius: 6px;
  color: var(--muted);
  background: rgba(255, 255, 255, 0.72);
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.field-control:focus-within {
  border-color: var(--green);
  box-shadow: 0 0 0 3px rgba(11, 151, 103, 0.07);
}

.field-control input {
  min-width: 0;
  flex: 1;
  border: 0;
  outline: 0;
  color: var(--ink);
  background: transparent;
  font-size: 13px;
}

.field-control input::placeholder {
  color: #a1a59e;
}

.password-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.forgot-button {
  margin-bottom: 8px;
  padding: 0;
  border: 0;
  color: var(--green-dark);
  background: transparent;
  font-size: 10px;
  cursor: pointer;
}

.password-toggle {
  display: grid;
  width: 27px;
  height: 27px;
  place-items: center;
  border: 0;
  border-radius: 4px;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.password-toggle:hover {
  background: var(--paper-deep);
}

.remember-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin: 3px 0 24px;
  color: var(--muted);
  font-size: 11px;
  cursor: pointer;
}

.remember-row input {
  position: absolute;
  opacity: 0;
}

.remember-row span {
  display: grid;
  width: 15px;
  height: 15px;
  place-items: center;
  border: 1px solid var(--line-dark);
  border-radius: 3px;
  background: #fff;
}

.remember-row input:checked + span {
  border-color: var(--ink);
  background: var(--ink);
}

.remember-row input:checked + span::after {
  width: 7px;
  height: 4px;
  border-bottom: 1.5px solid var(--acid);
  border-left: 1.5px solid var(--acid);
  content: "";
  transform: rotate(-45deg) translateY(-1px);
}

.submit-button {
  display: flex;
  width: 100%;
  height: 48px;
  align-items: center;
  justify-content: space-between;
  padding: 0 18px;
  border: 0;
  border-radius: 6px;
  color: #f7f8f3;
  background: var(--ink);
  font-size: 12px;
  font-weight: 650;
  cursor: pointer;
  transition: background 0.18s ease, transform 0.18s ease;
}

.submit-button:hover:not(:disabled) {
  background: var(--green-dark);
  transform: translateY(-1px);
}

.submit-button:disabled {
  cursor: wait;
  opacity: 0.72;
}

.submit-button i {
  width: 15px;
  height: 15px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: var(--acid);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.demo-access {
  margin-top: 26px;
  padding: 13px 14px;
  border: 1px dashed var(--line-dark);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.32);
}

.demo-access span {
  color: var(--green-dark);
  font-size: 8px;
  font-weight: 700;
  letter-spacing: 0.12em;
}

.demo-access p {
  margin: 5px 0 0;
  color: var(--muted);
  font-size: 10px;
  line-height: 1.6;
}

.form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: clamp(60px, 11vh, 110px);
  color: #9a9e97;
  font-size: 9px;
}

@keyframes center-pulse {
  0% {
    opacity: 0.8;
    transform: scale(0.9);
  }
  100% {
    opacity: 0;
    transform: scale(1.45);
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .login-page {
    grid-template-columns: minmax(0, 1.1fr) minmax(400px, 1fr);
  }

  .network-visual {
    width: 100%;
  }

  .node-a {
    left: 3%;
  }

  .node-b {
    right: 0;
  }
}

@media (max-width: 820px) {
  .login-page {
    display: block;
  }

  .signal-side {
    min-height: 290px;
  }

  .signal-side-inner {
    min-height: 290px;
    padding: 22px 26px;
  }

  .signal-copy {
    margin-top: 46px;
  }

  h1 {
    font-size: 36px;
  }

  .signal-description,
  .network-visual,
  .signal-footer {
    display: none;
  }

  .form-side {
    min-height: calc(100vh - 290px);
    padding: 30px 0;
  }

  .form-wrap {
    width: min(390px, calc(100% - 48px));
  }

  .form-topline {
    margin-bottom: 42px;
  }

  .form-footer {
    margin-top: 52px;
  }
}

@media (max-width: 480px) {
  .signal-side {
    min-height: 250px;
  }

  .signal-side-inner {
    min-height: 250px;
  }

  .signal-copy {
    margin-top: 35px;
  }

  h1 {
    font-size: 28px;
  }

  .desktop-only {
    display: none;
  }

  .build {
    display: none;
  }

  .form-side {
    min-height: calc(100vh - 250px);
  }
}
</style>
