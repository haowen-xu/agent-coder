<template>
  <el-table :data="projects" height="560" row-key="project_key">
    <el-table-column label="Key" prop="project_key" min-width="140" />
    <el-table-column label="Slug" prop="project_slug" min-width="180" />
    <el-table-column label="名称" prop="name" min-width="120" />
    <el-table-column label="启用" min-width="90">
      <template #default="scope">
        <el-tag :type="scope.row.enabled ? 'success' : 'info'" size="small">
          {{ scope.row.enabled ? 'Yes' : 'No' }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="Plan/Review 沙盒" min-width="130">
      <template #default="scope">
        <el-tag :type="scope.row.sandbox_plan_review ? 'warning' : 'info'" size="small">
          {{ scope.row.sandbox_plan_review ? 'On' : 'Off' }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column label="操作" min-width="170" fixed="right">
      <template #default="scope">
        <el-button link type="primary" @click="emit('edit', scope.row)">编辑</el-button>
        <el-button link type="primary" @click="emit('toggleEnabled', scope.row)">
          {{ scope.row.enabled ? '禁用' : '启用' }}
        </el-button>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import type { AdminProjectRow } from '../../stores/admin'

defineProps<{
  projects: AdminProjectRow[]
}>()

const emit = defineEmits<{
  edit: [row: AdminProjectRow]
  toggleEnabled: [row: AdminProjectRow]
}>()
</script>
