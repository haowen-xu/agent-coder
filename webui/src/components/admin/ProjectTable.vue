<template>
  <el-table
    :data="projects"
    height="520"
    highlight-current-row
    row-key="project_key"
    @current-change="onRowChange"
  >
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
  </el-table>
</template>

<script setup lang="ts">
import type { AdminProjectRow } from '../../stores/admin'

defineProps<{
  projects: AdminProjectRow[]
}>()

const emit = defineEmits<{
  select: [row: AdminProjectRow | null]
}>()

function onRowChange(row: AdminProjectRow | null) {
  emit('select', row)
}
</script>
