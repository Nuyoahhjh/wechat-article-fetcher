<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Pencil, Plus, RefreshCw, Trash2, UsersRound } from 'lucide-vue-next'
import { api } from '@/services/api'
import type { ManagedUser, UserRole } from '@/types'
import { useAuthStore } from '@/stores/auth'
import StatusPill from '@/components/common/StatusPill.vue'
import { formatDateTime } from '@/utils/format'

const auth = useAuthStore()
const users = ref<ManagedUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const saving = ref(false)
const createOpen = ref(false)
const editOpen = ref(false)
const editingId = ref('')

const createForm = reactive({
  username: '',
  displayName: '',
  password: '',
  role: 'operator' as UserRole,
})

const editForm = reactive({
  displayName: '',
  role: 'operator' as UserRole,
  status: 'active' as ManagedUser['status'],
  password: '',
})

async function load(nextPage = page.value) {
  page.value = nextPage
  loading.value = true
  try {
    const result = await api.users.list(page.value, pageSize)
    users.value = result.items
    total.value = result.total
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '用户列表加载失败')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  createForm.username = ''
  createForm.displayName = ''
  createForm.password = ''
  createForm.role = 'operator'
  createOpen.value = true
}

async function createUser() {
  if (!createForm.username.trim() || createForm.password.length < 8) {
    ElMessage.warning('用户名必填，密码至少 8 位')
    return
  }
  saving.value = true
  try {
    await api.users.create({
      username: createForm.username.trim(),
      displayName: createForm.displayName.trim() || createForm.username.trim(),
      password: createForm.password,
      role: createForm.role,
    })
    createOpen.value = false
    ElMessage.success('用户已创建')
    await load(1)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '创建用户失败')
  } finally {
    saving.value = false
  }
}

function openEdit(user: ManagedUser) {
  editingId.value = user.id
  editForm.displayName = user.displayName
  editForm.role = user.role
  editForm.status = user.status
  editForm.password = ''
  editOpen.value = true
}

async function updateUser() {
  saving.value = true
  try {
    await api.users.update(editingId.value, {
      displayName: editForm.displayName.trim(),
      role: editForm.role,
      status: editForm.status,
      password: editForm.password || undefined,
    })
    if (editingId.value === auth.user?.id) {
      auth.setDisplayName(editForm.displayName.trim())
    }
    editOpen.value = false
    ElMessage.success('用户已更新')
    await load()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '更新用户失败')
  } finally {
    saving.value = false
  }
}

async function removeUser(user: ManagedUser) {
  try {
    await ElMessageBox.confirm(
      `删除用户 ${user.displayName || user.username}？`,
      '删除用户',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    )
    await api.users.remove(user.id)
    ElMessage.success('用户已删除')
    await load(users.value.length === 1 && page.value > 1 ? page.value - 1 : page.value)
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : '删除用户失败')
  }
}

onMounted(() => load(1))
</script>

<template>
  <div class="page-shell">
    <header class="page-header">
      <div>
        <p class="eyebrow">ACCESS CONTROL</p>
        <h1 class="page-title">用户与权限</h1>
        <p class="page-lead">管理平台账号、角色与启停状态。角色权限由服务端强制校验。</p>
      </div>
      <div class="header-actions">
        <el-button :loading="loading" @click="load()">
          <RefreshCw v-if="!loading" :size="15" />
          刷新
        </el-button>
        <el-button type="primary" @click="openCreate">
          <Plus :size="15" />
          新建用户
        </el-button>
      </div>
    </header>

    <section class="user-summary panel">
      <span class="summary-icon"><UsersRound :size="20" /></span>
      <div>
        <span class="panel-kicker">PLATFORM ACCOUNTS</span>
        <strong class="mono">{{ total }}</strong>
        <p>名平台用户</p>
      </div>
      <div class="role-note">
        <span>admin 管理用户与 Agent</span>
        <span>operator 创建任务与扫码</span>
        <span>viewer 只读查看</span>
      </div>
    </section>

    <section class="panel table-panel">
      <el-table v-loading="loading" :data="users" row-key="id">
        <el-table-column label="用户" min-width="220">
          <template #default="{ row }">
            <div class="user-cell">
              <span class="avatar">{{ (row.displayName || row.username).slice(0, 1) }}</span>
              <span>
                <strong>{{ row.displayName || row.username }}</strong>
                <small class="mono">{{ row.username }}</small>
              </span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="120">
          <template #default="{ row }">
            <el-tag effect="plain" size="small">{{ row.role }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <StatusPill
              :status="row.status === 'active' ? 'online' : 'disabled'"
              :label="row.status === 'active' ? '启用' : '停用'"
            />
          </template>
        </el-table-column>
        <el-table-column
          label="最近登录"
          min-width="170"
          class-name="users-recent-column"
          label-class-name="users-recent-column"
        >
          <template #default="{ row }">
            <span class="muted">{{ row.lastLoginAt ? formatDateTime(row.lastLoginAt) : '尚未登录' }}</span>
          </template>
        </el-table-column>
        <el-table-column
          label="创建时间"
          min-width="170"
          class-name="users-created-column"
          label-class-name="users-created-column"
        >
          <template #default="{ row }">
            <span class="muted">{{ formatDateTime(row.createdAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="" width="110" align="right">
          <template #default="{ row }">
            <el-button text size="small" title="编辑用户" @click="openEdit(row)">
              <Pencil :size="14" />
            </el-button>
            <el-button
              text
              size="small"
              type="danger"
              title="删除用户"
              :disabled="row.id === auth.user?.id"
              @click="removeUser(row)"
            >
              <Trash2 :size="14" />
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-row">
        <span>共 {{ total }} 名用户</span>
        <el-pagination
          size="small"
          background
          layout="prev, pager, next"
          :total="total"
          :page-size="pageSize"
          :current-page="page"
          @current-change="load"
        />
      </div>
    </section>

    <el-dialog v-model="createOpen" title="新建用户" width="min(440px, 92vw)">
      <el-form label-position="top">
        <el-form-item label="用户名"><el-input v-model="createForm.username" /></el-form-item>
        <el-form-item label="显示名称"><el-input v-model="createForm.displayName" /></el-form-item>
        <el-form-item label="初始密码"><el-input v-model="createForm.password" type="password" show-password /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="createForm.role" style="width: 100%">
            <el-option label="operator" value="operator" />
            <el-option label="viewer" value="viewer" />
            <el-option label="admin" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createOpen = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="createUser">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="editOpen" title="编辑用户" width="min(440px, 92vw)">
      <el-form label-position="top">
        <el-form-item label="显示名称"><el-input v-model="editForm.displayName" /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="editForm.role" style="width: 100%">
            <el-option label="admin" value="admin" />
            <el-option label="operator" value="operator" />
            <el-option label="viewer" value="viewer" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="editForm.status" style="width: 100%">
            <el-option label="启用" value="active" />
            <el-option label="停用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="重置密码（留空不修改）">
          <el-input v-model="editForm.password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editOpen = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="updateUser">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 24px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.user-summary {
  display: flex;
  min-height: 112px;
  align-items: center;
  gap: 16px;
  margin-bottom: 18px;
  padding: 20px;
}

.summary-icon {
  display: grid;
  width: 46px;
  height: 46px;
  flex: 0 0 46px;
  place-items: center;
  border-radius: 7px;
  color: var(--green-dark);
  background: rgba(11, 151, 103, 0.08);
}

.user-summary > div:nth-child(2) {
  min-width: 130px;
}

.user-summary strong {
  display: block;
  margin-top: 5px;
  font-size: 27px;
}

.user-summary p {
  margin: 0;
  color: var(--muted);
  font-size: 9px;
}

.role-note {
  display: flex;
  gap: 8px;
  margin-left: auto;
  flex-wrap: wrap;
}

.role-note span {
  padding: 7px 9px;
  border: 1px solid var(--line);
  border-radius: 4px;
  color: var(--muted);
  background: var(--paper);
  font-size: 8px;
}

.table-panel {
  overflow: hidden;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.avatar {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  place-items: center;
  border-radius: 6px;
  color: var(--ink);
  background: var(--acid);
  font-size: 12px;
  font-weight: 700;
}

.user-cell > span:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.user-cell strong {
  font-size: 11px;
}

.user-cell small,
.muted {
  color: var(--muted);
  font-size: 8px;
}

.pagination-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
}

.pagination-row span {
  color: var(--muted);
  font-size: 9px;
}

@media (max-width: 760px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .user-summary {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .role-note {
    width: 100%;
    margin-left: 0;
  }
}

@media (max-width: 1000px) {
  :deep(.users-recent-column),
  :deep(.users-created-column) {
    display: none;
  }
}
</style>
