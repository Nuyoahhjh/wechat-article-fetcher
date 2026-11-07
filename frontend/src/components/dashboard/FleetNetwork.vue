<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowUpRight } from 'lucide-vue-next'
import AgentAvatar from '@/components/common/AgentAvatar.vue'
import type { Agent } from '@/types'

const props = defineProps<{
  agents: Agent[]
}>()

const router = useRouter()
const displayedAgents = computed(() => props.agents.slice(0, 6))

const statusLabel: Record<string, string> = {
  online: '在线',
  busy: '运行中',
  offline: '离线',
  disabled: '已禁用',
  error: '异常',
}

function openAgent(id: string) {
  router.push({ name: 'agent-detail', params: { id } })
}
</script>

<template>
  <section class="fleet-network panel">
    <div class="panel-head">
      <h2 class="panel-title">
        <span class="signal-dot" />
        Agent 网络
      </h2>
      <button class="text-button" @click="router.push({ name: 'agents' })">
        全部节点 <ArrowUpRight :size="13" />
      </button>
    </div>
    <div class="network-stage">
      <div class="stage-grid" />
      <div class="stage-line line-x" />
      <div class="stage-line line-y" />
      <button
        v-for="(agent, index) in displayedAgents"
        :key="agent.id"
        class="agent-node"
        :class="[`node-${index + 1}`, agent.status]"
        @click="openAgent(agent.id)"
      >
        <span class="node-signal">
          <i />
        </span>
        <AgentAvatar :name="agent.wechat.nickname" :size="29" />
        <span class="node-copy">
          <strong>{{ agent.code }}</strong>
          <small>{{ agent.hostname || agent.name }} · {{ statusLabel[agent.status] }}</small>
        </span>
      </button>
      <div class="network-core">
        <span class="core-ring ring-one" />
        <span class="core-ring ring-two" />
        <span class="core-label">拾刊</span>
        <small>CORE</small>
      </div>
      <span class="stage-coord mono">AGENT NODES</span>
      <span class="stage-coord right mono">SERVER DATA</span>
    </div>
  </section>
</template>

<style scoped>
.network-stage {
  position: relative;
  height: 300px;
  overflow: hidden;
  background: #19271f;
}

.stage-grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.036) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.036) 1px, transparent 1px);
  background-size: 34px 34px;
  mask-image: radial-gradient(ellipse at center, #000 20%, transparent 76%);
}

.stage-line {
  position: absolute;
  background: rgba(201, 243, 106, 0.11);
}

.line-x {
  top: 50%;
  left: 10%;
  width: 80%;
  height: 1px;
}

.line-y {
  top: 12%;
  left: 50%;
  width: 1px;
  height: 76%;
}

.network-core {
  position: absolute;
  top: 50%;
  left: 50%;
  display: flex;
  width: 72px;
  height: 72px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  border: 1px solid rgba(201, 243, 106, 0.38);
  border-radius: 50%;
  color: var(--acid);
  background: #19271f;
  transform: translate(-50%, -50%);
}

.core-label {
  font-family: "Microsoft YaHei UI", sans-serif;
  font-size: 12px;
  font-weight: 700;
}

.network-core small {
  margin-top: 2px;
  color: rgba(255, 255, 255, 0.32);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 6px;
  letter-spacing: 0.13em;
}

.core-ring {
  position: absolute;
  border: 1px solid rgba(201, 243, 106, 0.1);
  border-radius: 50%;
}

.ring-one {
  inset: -11px;
}

.ring-two {
  inset: -24px;
  border-color: rgba(201, 243, 106, 0.055);
}

.agent-node {
  position: absolute;
  z-index: 2;
  display: flex;
  min-width: 123px;
  align-items: center;
  gap: 8px;
  padding: 7px 9px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: #edf1eb;
  background: rgba(11, 18, 14, 0.86);
  text-align: left;
  cursor: pointer;
  backdrop-filter: blur(6px);
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.agent-node:hover {
  border-color: rgba(201, 243, 106, 0.38);
  background: rgba(21, 34, 27, 0.98);
  transform: translateY(-2px);
}

.node-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.node-copy strong {
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 9px;
  font-weight: 650;
  letter-spacing: 0.04em;
}

.node-copy small {
  color: rgba(237, 241, 235, 0.38);
  font-size: 8px;
}

.node-signal {
  position: absolute;
  top: -4px;
  right: -4px;
  display: grid;
  width: 9px;
  height: 9px;
  place-items: center;
  border: 2px solid #19271f;
  border-radius: 50%;
  background: var(--acid);
}

.error .node-signal {
  background: #e5a747;
}

.offline .node-signal,
.disabled .node-signal {
  background: #717b73;
}

.node-1 {
  top: 36px;
  left: 7%;
}

.node-2 {
  top: 35px;
  right: 6%;
}

.node-3 {
  top: 50%;
  left: 3%;
  transform: translateY(-50%);
}

.node-3:hover {
  transform: translateY(calc(-50% - 2px));
}

.node-4 {
  top: 50%;
  right: 2%;
  transform: translateY(-50%);
}

.node-4:hover {
  transform: translateY(calc(-50% - 2px));
}

.node-5 {
  bottom: 33px;
  left: 13%;
}

.node-6 {
  right: 12%;
  bottom: 33px;
}

.stage-coord {
  position: absolute;
  bottom: 12px;
  left: 13px;
  color: rgba(255, 255, 255, 0.19);
  font-size: 6px;
  letter-spacing: 0.12em;
}

.stage-coord.right {
  right: 13px;
  left: auto;
}

@media (max-width: 1180px) {
  .agent-node {
    min-width: 105px;
  }

  .agent-node :deep(.agent-avatar) {
    display: none;
  }
}
</style>
