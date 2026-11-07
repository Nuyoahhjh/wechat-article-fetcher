<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, Search, X } from 'lucide-vue-next'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const router = useRouter()
const keyword = ref('')
const input = ref<HTMLInputElement | null>(null)

const commands = [
  { label: '打开工作台', hint: '全局态势与运行概览', name: 'dashboard', group: '导航' },
  { label: '查看 Agent 网络', hint: '节点状态与微信登录', name: 'agents', group: '导航' },
  { label: '进入采集任务', hint: '任务进度与创建任务', name: 'tasks', group: '导航' },
  { label: '浏览文章资料库', hint: '检索已采集文章', name: 'articles', group: '导航' },
  { label: '平台设置', hint: '通知、刷新与数据策略', name: 'settings', group: '系统' },
]

const filtered = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  if (!query) return commands
  return commands.filter((command) =>
    `${command.label} ${command.hint} ${command.group}`.toLowerCase().includes(query),
  )
})

watch(
  () => props.modelValue,
  async (open) => {
    if (open) {
      keyword.value = ''
      await nextTick()
      input.value?.focus()
    }
  },
)

function close() {
  emit('update:modelValue', false)
}

async function run(name: string) {
  close()
  await router.push({ name })
}
</script>

<template>
  <Teleport to="body">
    <Transition name="palette">
      <div v-if="modelValue" class="palette-backdrop" @mousedown.self="close">
        <section class="palette">
          <header>
            <Search :size="19" :stroke-width="1.7" />
            <input
              ref="input"
              v-model="keyword"
              placeholder="搜索页面与操作"
              @keydown.esc="close"
            />
            <button class="close" title="关闭" @click="close">
              <X :size="16" />
            </button>
          </header>
          <div class="palette-body">
            <span class="group-label">{{ keyword ? '搜索结果' : '快速前往' }}</span>
            <button
              v-for="command in filtered"
              :key="command.name"
              class="command-row"
              @click="run(command.name)"
            >
              <span class="command-index">{{ command.group }}</span>
              <span class="command-text">
                <strong>{{ command.label }}</strong>
                <small>{{ command.hint }}</small>
              </span>
              <ArrowRight :size="15" />
            </button>
            <div v-if="!filtered.length" class="command-empty">没有匹配的页面或操作</div>
          </div>
          <footer>
            <span><kbd>ESC</kbd> 关闭</span>
            <span>拾刊 Agent Console</span>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.palette-backdrop {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 12vh 20px 20px;
  background: rgba(11, 17, 13, 0.54);
  backdrop-filter: blur(6px);
}

.palette {
  width: min(590px, 100%);
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: 0 28px 80px rgba(0, 0, 0, 0.3);
}

.palette header {
  display: flex;
  height: 62px;
  align-items: center;
  gap: 12px;
  padding: 0 18px;
  border-bottom: 1px solid var(--line);
}

.palette header input {
  min-width: 0;
  flex: 1;
  border: 0;
  outline: 0;
  color: var(--ink);
  background: transparent;
  font-size: 16px;
}

.palette header input::placeholder {
  color: #9a9e97;
}

.close {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border: 0;
  border-radius: 5px;
  color: var(--muted);
  background: transparent;
  cursor: pointer;
}

.close:hover {
  background: var(--paper-deep);
}

.palette-body {
  min-height: 240px;
  max-height: 52vh;
  overflow-y: auto;
  padding: 10px;
}

.group-label {
  display: block;
  padding: 9px 10px 7px;
  color: var(--muted);
  font-family: "Bricolage Grotesque Variable", sans-serif;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.15em;
}

.command-row {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 14px;
  padding: 11px 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  text-align: left;
  cursor: pointer;
}

.command-row:hover {
  background: var(--paper-deep);
}

.command-index {
  display: grid;
  width: 46px;
  height: 28px;
  flex: 0 0 46px;
  place-items: center;
  border: 1px solid var(--line);
  border-radius: 5px;
  color: var(--muted);
  font-size: 9px;
  font-weight: 700;
}

.command-text {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.command-text strong {
  font-size: 13px;
  font-weight: 650;
}

.command-text small {
  color: var(--muted);
  font-size: 11px;
}

.command-empty {
  padding: 60px 20px;
  color: var(--muted);
  font-size: 13px;
  text-align: center;
}

.palette footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 18px;
  border-top: 1px solid var(--line);
  color: var(--muted);
  font-size: 10px;
}

kbd {
  padding: 2px 5px;
  border: 1px solid var(--line);
  border-radius: 3px;
  background: var(--paper);
  font-family: inherit;
  font-size: 9px;
}

.palette-enter-active,
.palette-leave-active {
  transition: opacity 0.16s ease;
}

.palette-enter-active .palette,
.palette-leave-active .palette {
  transition: transform 0.18s ease, opacity 0.16s ease;
}

.palette-enter-from,
.palette-leave-to {
  opacity: 0;
}

.palette-enter-from .palette,
.palette-leave-to .palette {
  opacity: 0;
  transform: translateY(-8px) scale(0.99);
}
</style>
