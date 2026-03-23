<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :span="24">
      <el-card class="admin-card" shadow="never">
        <template #header>
          <div class="panel-header">
            <span class="panel-title">用户管理</span>
            <el-button type="primary" @click="openCreateDialog">创建用户</el-button>
          </div>
        </template>

        <AsyncStateAlert :error="admin.error" />
        <UserTable
          :users="admin.users"
          @edit="openEditDialog"
          @toggle-admin="toggleUserAdmin"
          @toggle-enabled="toggleUserEnabled"
        />
      </el-card>
    </el-col>
  </el-row>

  <el-dialog v-model="createDialogVisible" title="创建用户" width="520px" destroy-on-close>
    <el-form label-position="top">
      <el-form-item label="用户名">
        <el-input v-model="createForm.username" placeholder="请输入用户名" />
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="createForm.password" type="password" show-password placeholder="请输入密码" />
      </el-form-item>
      <div class="form-inline-row">
        <el-checkbox v-model="createForm.is_admin">管理员</el-checkbox>
        <el-checkbox v-model="createForm.enabled">启用</el-checkbox>
      </div>
    </el-form>
    <template #footer>
      <el-button @click="createDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="admin.loading" @click="createAdminUser">创建</el-button>
    </template>
  </el-dialog>

  <el-dialog v-model="editDialogVisible" title="编辑用户" width="520px" destroy-on-close>
    <el-form label-position="top">
      <el-form-item label="用户名">
        <el-input :model-value="editingUser?.username ?? ''" disabled />
      </el-form-item>
      <el-form-item label="新密码（可选）">
        <el-input v-model="editForm.password" type="password" show-password placeholder="留空则不修改密码" />
      </el-form-item>
      <div class="form-inline-row">
        <el-checkbox v-model="editForm.is_admin">管理员</el-checkbox>
        <el-checkbox v-model="editForm.enabled">启用</el-checkbox>
      </div>
    </el-form>
    <template #footer>
      <el-button @click="editDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="admin.loading" @click="saveEditedUser">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useAdminStore, type AdminUserRow } from '../../stores/admin'
import { useSessionStore } from '../../stores/session'
import UserTable from '../../components/admin/UserTable.vue'
import AsyncStateAlert from '../../components/common/AsyncStateAlert.vue'

const session = useSessionStore()
const admin = useAdminStore()

const createDialogVisible = ref(false)
const editDialogVisible = ref(false)
const editingUser = ref<AdminUserRow | null>(null)

const createForm = reactive({
  username: '',
  password: '',
  is_admin: false,
  enabled: true,
})

const editForm = reactive({
  password: '',
  is_admin: false,
  enabled: true,
})

async function refresh() {
  if (!session.token) {
    return
  }
  await admin.fetchUsers(session.token)
}

function resetCreateForm() {
  createForm.username = ''
  createForm.password = ''
  createForm.is_admin = false
  createForm.enabled = true
}

function openCreateDialog() {
  resetCreateForm()
  createDialogVisible.value = true
}

async function createAdminUser() {
  if (!session.token) {
    return
  }
  if (!createForm.username.trim() || !createForm.password.trim()) {
    ElMessage.warning('用户名和密码不能为空')
    return
  }

  await admin.createUser(session.token, {
    username: createForm.username.trim(),
    password: createForm.password,
    is_admin: createForm.is_admin,
    enabled: createForm.enabled,
  })
  await admin.fetchUsers(session.token)
  createDialogVisible.value = false
  resetCreateForm()
  ElMessage.success('用户已创建')
}

async function toggleUserEnabled(row: AdminUserRow) {
  if (!session.token) {
    return
  }
  await admin.updateUser(session.token, row.id, {
    enabled: !row.enabled,
  })
  await admin.fetchUsers(session.token)
}

async function toggleUserAdmin(row: AdminUserRow) {
  if (!session.token) {
    return
  }
  await admin.updateUser(session.token, row.id, {
    is_admin: !row.is_admin,
  })
  await admin.fetchUsers(session.token)
}

function openEditDialog(row: AdminUserRow) {
  editingUser.value = row
  editForm.password = ''
  editForm.is_admin = row.is_admin
  editForm.enabled = row.enabled
  editDialogVisible.value = true
}

async function saveEditedUser() {
  if (!session.token || !editingUser.value) {
    return
  }

  const payload: { password?: string; is_admin: boolean; enabled: boolean } = {
    is_admin: editForm.is_admin,
    enabled: editForm.enabled,
  }
  if (editForm.password.trim()) {
    payload.password = editForm.password
  }

  await admin.updateUser(session.token, editingUser.value.id, payload)
  await admin.fetchUsers(session.token)
  editDialogVisible.value = false
  ElMessage.success('用户已更新')
}

onMounted(async () => {
  await refresh()
})
</script>

<style scoped>
.panel-row {
  margin-top: 4px;
}

.panel-title {
  font-weight: 600;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.form-inline-row {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}
</style>
