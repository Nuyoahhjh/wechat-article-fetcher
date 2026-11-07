<script setup lang="ts">
import { ArrowLeft, Compass } from 'lucide-vue-next'
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <main class="not-found">
    <div class="not-found-grid" />
    <div class="error-code mono">404</div>
    <div class="error-content">
      <span class="error-icon"><Compass :size="23" :stroke-width="1.4" /></span>
      <p class="eyebrow">ROUTE NOT FOUND</p>
      <h1>这个页面不在采集网络里。</h1>
      <p>链接可能已经失效，或当前账号没有访问权限。</p>
      <button @click="router.push({ name: 'dashboard' })">
        <ArrowLeft :size="15" />
        返回工作台
      </button>
    </div>
    <span class="corner-code mono">SHIKAN / NAVIGATION LOST</span>
  </main>
</template>

<style scoped>
.not-found {
  position: relative;
  display: flex;
  min-height: 100vh;
  overflow: hidden;
  align-items: center;
  justify-content: center;
  color: #eff3ed;
  background: var(--ink);
}

.not-found-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.035) 1px, transparent 1px);
  background-size: 58px 58px;
  mask-image: radial-gradient(ellipse at center, #000 5%, transparent 70%);
}

.error-code {
  position: absolute;
  color: rgba(201, 243, 106, 0.06);
  font-size: min(40vw, 530px);
  font-weight: 650;
  letter-spacing: -0.1em;
  line-height: 1;
}

.error-content {
  position: relative;
  z-index: 1;
  max-width: 500px;
  padding: 30px;
  text-align: center;
}

.error-icon {
  display: grid;
  width: 48px;
  height: 48px;
  margin: 0 auto 24px;
  place-items: center;
  border: 1px solid rgba(201, 243, 106, 0.2);
  border-radius: 50%;
  color: var(--acid);
  background: rgba(201, 243, 106, 0.05);
}

.error-content .eyebrow {
  color: rgba(255, 255, 255, 0.34);
}

.error-content .eyebrow::before {
  display: none;
}

.error-content h1 {
  margin: 0;
  font-size: clamp(28px, 4vw, 44px);
  font-weight: 650;
  letter-spacing: -0.045em;
}

.error-content > p:not(.eyebrow) {
  margin: 14px 0 28px;
  color: rgba(255, 255, 255, 0.4);
  font-size: 12px;
}

.error-content button {
  display: inline-flex;
  height: 40px;
  align-items: center;
  gap: 7px;
  padding: 0 15px;
  border: 1px solid var(--acid);
  border-radius: 5px;
  color: var(--ink);
  background: var(--acid);
  font-size: 10px;
  font-weight: 650;
  cursor: pointer;
}

.corner-code {
  position: absolute;
  right: 24px;
  bottom: 20px;
  color: rgba(255, 255, 255, 0.18);
  font-size: 7px;
  letter-spacing: 0.12em;
}
</style>
