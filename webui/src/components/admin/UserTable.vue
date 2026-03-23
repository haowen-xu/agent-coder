<template>
  <el-table :data="users" height="560">
    <el-table-column label="ID" prop="id" width="70" />
    <el-table-column label="用户名" prop="username" min-width="120" />
    <el-table-column label="最近登录" min-width="170">
      <template #default="scope">
        {{ scope.row.last_login_at || '-' }}
      </template>
    </el-table-column>
    <el-table-column label="Admin" min-width="90">
      <template #default="scope">
        <el-tag :type="scope.row.is_admin ? 'warning' : 'info'" size="small">
          {{ scope.row.is_admin ? 'Yes' : 'No' }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="启用" min-width="90">
      <template #default="scope">
        <el-tag :type="scope.row.enabled ? 'success' : 'danger'" size="small">
          {{ scope.row.enabled ? 'Yes' : 'No' }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="操作" min-width="180">
      <template #default="scope">
        <el-button link type="primary" @click="emit('edit', scope.row)">编辑</el-button>
        <el-button link type="primary" @click="emit('toggleAdmin', scope.row)">
          {{ scope.row.is_admin ? '取消管理员' : '设为管理员' }}
        </el-button>
        <el-button link type="primary" @click="emit('toggleEnabled', scope.row)">
          {{ scope.row.enabled ? '禁用' : '启用' }}
        </el-button>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import type { AdminUserRow } from '../../stores/admin'

defineProps<{
  users: AdminUserRow[]
}>()

const emit = defineEmits<{
  edit: [row: AdminUserRow]
  toggleAdmin: [row: AdminUserRow]
  toggleEnabled: [row: AdminUserRow]
}>()
</script>
