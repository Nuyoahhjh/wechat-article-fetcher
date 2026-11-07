import { ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '@/services/api'
import type {
  CreateTaskPayload,
  Task,
  TaskEvent,
  TaskListQuery,
  TaskStatus,
} from '@/types'

export const useTaskStore = defineStore('tasks', () => {
  const items = ref<Task[]>([])
  const total = ref(0)
  const loading = ref(false)
  const creating = ref(false)
  const selectedTask = ref<Task | null>(null)
  const events = ref<TaskEvent[]>([])
  const eventsLoading = ref(false)
  const drawerOpen = ref(false)
  const query = ref<TaskListQuery>({
    page: 1,
    pageSize: 10,
    status: 'all',
    keyword: '',
  })

  async function fetchTasks(nextQuery?: Partial<TaskListQuery>) {
    query.value = { ...query.value, ...nextQuery }
    loading.value = true
    try {
      const result = await api.tasks.list(query.value)
      items.value = result.items
      total.value = result.total
    } finally {
      loading.value = false
    }
  }

  async function createTask(payload: CreateTaskPayload, agentId?: string) {
    creating.value = true
    try {
      const task = await api.tasks.create(payload, agentId)
      items.value.unshift(task)
      total.value += 1
      return task
    } finally {
      creating.value = false
    }
  }

  async function openTask(task: Task) {
    selectedTask.value = task
    drawerOpen.value = true
    eventsLoading.value = true
    try {
      const [fresh, taskEvents] = await Promise.all([
        api.tasks.get(task.id),
        api.tasks.events(task.id),
      ])
      selectedTask.value = fresh
      events.value = taskEvents
    } finally {
      eventsLoading.value = false
    }
  }

  async function refreshSelected() {
    if (!selectedTask.value) return
    const task = items.value.find((item) => item.id === selectedTask.value?.id)
    await openTask(task ?? selectedTask.value)
  }

  async function cancelTask(task: Task) {
    const updated = await api.tasks.cancel(task.id)
    const index = items.value.findIndex((item) => item.id === updated.id)
    if (index >= 0) items.value[index] = updated
    if (selectedTask.value?.id === updated.id) {
      selectedTask.value = updated
      events.value = await api.tasks.events(updated.id)
    }
    return updated
  }

  function applyProgress(
    taskId: string,
    progress: number,
    collected: number,
    status: TaskStatus,
  ) {
    const task = items.value.find((item) => item.id === taskId)
    if (task) {
      task.progress = progress
      task.collected = collected
      task.status = status
    }
    if (selectedTask.value?.id === taskId) {
      selectedTask.value.progress = progress
      selectedTask.value.collected = collected
      selectedTask.value.status = status
    }
  }

  return {
    items,
    total,
    loading,
    creating,
    selectedTask,
    events,
    eventsLoading,
    drawerOpen,
    query,
    fetchTasks,
    createTask,
    openTask,
    refreshSelected,
    cancelTask,
    applyProgress,
  }
})
