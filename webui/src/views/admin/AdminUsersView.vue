<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :span="24">
      <el-card class="admin-card" shadow="never">
        <template #header>
          <div class="panel-title">用户管理</div>
        </template>

        <UserCreateForm :model="newUserForm" :loading="admin.loading" @submit="createAdminUser" />
        <AsyncStateAlert :error="admin.error" />

        <UserTable :users="admin.users" @toggle-admin="toggleUserAdmin" @toggle-enabled="toggleUserEnabled" />
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { useAdminStore, type AdminUserRow } from '../../stores/admin'
import { useSessionStore } from '../../stores/session'
import UserCreateForm from '../../components/admin/UserCreateForm.vue'
import UserTable from '../../components/admin/UserTable.vue'
import AsyncStateAlert from '../../components/common/AsyncStateAlert.vue'

const session = useSessionStore()
const admin = useAdminStore()

const newUserForm = reactive({
  username: '',
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

async function createAdminUser() {
  if (!session.token) {
    return
  }
  await admin.createUser(session.token, newUserForm)
  await admin.fetchUsers(session.token)
  newUserForm.username = ''
  newUserForm.password = ''
  newUserForm.is_admin = false
  newUserForm.enabled = true
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
</style>
