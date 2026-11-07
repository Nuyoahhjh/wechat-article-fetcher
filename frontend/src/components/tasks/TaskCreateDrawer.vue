<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  ArrowRight,
  Bot,
  CalendarRange,
  FileSearch,
  Radio,
} from 'lucide-vue-next'
import AgentAvatar from '@/components/common/AgentAvatar.vue'
import { useAgentStore } from '@/stores/agents'
import { useTaskStore } from '@/stores/tasks'
import type { CreateTaskPayload } from '@/types'

const props = defineProps<{
  modelValue: boolean
  initialAgentId?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  created: [taskId: string]
}>()

const agents = useAgentStore()
const tasks = useTaskStore()
const dateRange = ref<[Date, Date] | null>(null)
const form = reactive({
  agentId: '',
  accountName: '',
  maxArticles: 20,
  priority: 100,
  includeContent: true,
})

const schedulableAgents = computed(() =>
  agents.items.filter((agent) => agent.status !== 'disabled'),
)
const selectedAgent = computed(() =>
  agents.items.find((agent) => agent.id === form.agentId),
)

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    form.agentId = props.initialAgentId || ''
    form.accountName = ''
    form.maxArticles = 20
    form.priority = 100
    form.includeContent = true
    dateRange.value = null
  },
)

function close() {
  emit('update:modelValue', false)
}

async function submit() {
  if (!form.accountName.trim()) {
    ElMessage.warning('请输入公众号名称')
    return
  }
  if (form.maxArticles < 1 || form.maxArticles > 500) {
    ElMessage.warning('采集数量需在 1 到 500 之间')
    return
  }
  const payload: CreateTaskPayload = {
    accountName: form.accountName.trim(),
    maxArticles: form.maxArticles,
    priority: form.priority,
    includeContent: form.includeContent,
    since: dateRange.value?.[0]?.toISOString(),
    until: dateRange.value?.[1]?.toISOString(),
  }
  try {
    const task = await tasks.createTask(payload, form.agentId || undefined)
    ElMessage.success('采集任务已进入队列')
    emit('created', task.id)
    close()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '创建任务失败')
  }
}
</script>

<template>
  <el-drawer
    :model-value="modelValue"
    append-to-body
    size="min(560px, 92vw)"
    :show-close="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #header>
      <div class="drawer-head">
        <div>
          <p class="eyebrow">NEW COLLECTION</p>
          <h2>创建采集任务</h2>
        </div>
        <button class="icon-button" title="关闭" @click="close">×</button>
      </div>
    </template>

    <div class="drawer-body">
      <section class="form-section">
        <div class="section-title">
          <span class="mono">01</span>
          <div>
            <strong>公众号与范围</strong>
            <small>按完整公众号名称搜索，不接受模糊账号</small>
          </div>
        </div>
        <div class="field">
          <label>公众号名称 <i>*</i></label>
          <el-input
            v-model="form.accountName"
            maxlength="200"
            placeholder="输入微信中显示的完整名称"
            clearable
          />
        </div>
        <div class="two-fields">
          <div class="field">
            <label>最多采集</label>
            <el-input-number
              v-model="form.maxArticles"
              :min="1"
              :max="500"
              controls-position="right"
            />
          </div>
          <div class="field">
            <label>优先级</label>
            <el-input-number
              v-model="form.priority"
              :min="1"
              :max="1000"
              controls-position="right"
            />
          </div>
        </div>
        <div class="field">
          <label>发布时间范围（可选）</label>
          <el-date-picker
            v-model="dateRange"
            type="datetimerange"
            start-placeholder="起始时间"
            end-placeholder="结束时间"
            range-separator="至"
            style="width: 100%"
          />
          <small class="field-hint"><CalendarRange :size="12" />未选择时按最新文章采集</small>
        </div>
        <label class="startup-option">
          <span class="switch-copy">
            <FileSearch :size="15" />
            <span>
              <strong>抓取正文</strong>
              <small>关闭后仍会解析文章链接并入库元数据</small>
            </span>
          </span>
          <el-switch v-model="form.includeContent" />
        </label>
      </section>

      <section class="form-section">
        <div class="section-title">
          <span class="mono">02</span>
          <div>
            <strong>执行节点</strong>
            <small>可指定 Agent，也可交给调度器自动分配</small>
          </div>
        </div>
        <button
          class="auto-agent"
          type="button"
          :class="{ selected: !form.agentId }"
          @click="form.agentId = ''"
        >
          <span class="auto-icon"><Bot :size="17" /></span>
          <span>
            <strong>自动调度</strong>
            <small>选择在线、已登录微信且未满载的节点</small>
          </span>
          <i />
        </button>
        <div class="agent-selector">
          <button
            v-for="agent in schedulableAgents"
            :key="agent.id"
            type="button"
            :class="{ selected: form.agentId === agent.id }"
            @click="form.agentId = agent.id"
          >
            <AgentAvatar :name="agent.name" :size="32" />
            <span>
              <strong>{{ agent.name }}</strong>
              <small>{{ agent.hostname || agent.code }} · {{ agent.wechat.loginStatus === 'logged_in' ? '微信已登录' : '微信未登录' }}</small>
            </span>
            <i />
          </button>
        </div>
        <div v-if="selectedAgent" class="agent-capacity">
          <span><Radio :size="12" />{{ selectedAgent.status }}</span>
          <span>并发 {{ selectedAgent.config.maxConcurrency }}</span>
          <span>执行中 {{ selectedAgent.activeTasks }}</span>
        </div>
        <div v-else-if="!schedulableAgents.length" class="agent-empty">
          暂无已注册 Agent，任务会保持待调度状态。
        </div>
      </section>
    </div>

    <template #footer>
      <div class="drawer-footer">
        <div class="queue-note">任务会按服务端队列和租约策略执行</div>
        <el-button @click="close">取消</el-button>
        <el-button type="primary" :loading="tasks.creating" @click="submit">
          创建任务
          <ArrowRight v-if="!tasks.creating" :size="15" />
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<style scoped>
.drawer-head {
  display: flex;
  width: 100%;
  align-items: flex-start;
  justify-content: space-between;
}

.drawer-head h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 680;
  letter-spacing: -0.03em;
}

.drawer-head .icon-button {
  font-size: 20px;
  font-weight: 300;
}

.drawer-body {
  padding: 0 24px 48px;
}

.form-section {
  padding: 23px 0;
  border-bottom: 1px solid var(--line);
}

.form-section:last-child {
  border-bottom: 0;
}

.section-title {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  margin-bottom: 16px;
}

.section-title > span {
  color: var(--green-dark);
  font-size: 9px;
  font-weight: 650;
}

.section-title > div {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.section-title strong {
  font-size: 12px;
  font-weight: 650;
}

.section-title small {
  color: var(--muted);
  font-size: 9px;
}

.field {
  display: flex;
  gap: 7px;
  margin-bottom: 14px;
  flex-direction: column;
}

.field label {
  color: var(--ink-2);
  font-size: 9px;
  font-weight: 600;
}

.field label i {
  color: var(--danger);
  font-style: normal;
}

.field-hint {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--muted);
  font-size: 8px;
}

.two-fields {
  display: grid;
  gap: 10px;
  grid-template-columns: 1fr 1fr;
}

.two-fields :deep(.el-input-number) {
  width: 100%;
}

.startup-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 11px 12px;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--paper);
}

.switch-copy {
  display: flex;
  align-items: center;
  gap: 9px;
}

.switch-copy > span {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.switch-copy strong {
  font-size: 9px;
}

.switch-copy small {
  color: var(--muted);
  font-size: 8px;
}

.auto-agent,
.agent-selector button {
  position: relative;
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  gap: 9px;
  padding: 11px;
  border: 1px solid var(--line);
  border-radius: 6px;
  color: var(--ink);
  background: #fff;
  text-align: left;
  cursor: pointer;
}

.auto-agent {
  margin-bottom: 8px;
}

.auto-agent.selected,
.agent-selector button.selected {
  border-color: var(--green);
  box-shadow: 0 0 0 2px rgba(11, 151, 103, 0.07);
}

.auto-icon {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border-radius: 5px;
  color: var(--green-dark);
  background: rgba(11, 151, 103, 0.08);
}

.auto-agent > span:nth-child(2),
.agent-selector button > span {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.auto-agent strong,
.agent-selector button strong {
  font-size: 10px;
  font-weight: 650;
}

.auto-agent small,
.agent-selector button small {
  overflow: hidden;
  color: var(--muted);
  font-size: 8px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.auto-agent > i,
.agent-selector button > i {
  width: 8px;
  height: 8px;
  border: 1px solid var(--line-dark);
  border-radius: 50%;
}

.auto-agent.selected > i,
.agent-selector button.selected > i {
  border-color: var(--green);
  background: var(--green);
  box-shadow: inset 0 0 0 2px #fff;
}

.agent-selector {
  display: grid;
  gap: 7px;
  grid-template-columns: 1fr 1fr;
}

.agent-capacity {
  display: flex;
  gap: 13px;
  margin-top: 9px;
  padding: 8px 10px;
  border-radius: 5px;
  color: var(--muted);
  background: var(--paper);
  font-size: 8px;
}

.agent-capacity span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.agent-empty {
  padding: 18px;
  border: 1px dashed var(--line-dark);
  border-radius: 6px;
  color: var(--muted);
  font-size: 9px;
  text-align: center;
}

.drawer-footer {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 24px;
  border-top: 1px solid var(--line);
}

.queue-note {
  flex: 1;
  color: var(--muted);
  font-size: 8px;
}

@media (max-width: 520px) {
  .agent-selector {
    grid-template-columns: 1fr;
  }

  .queue-note {
    display: none;
  }
}
</style>
