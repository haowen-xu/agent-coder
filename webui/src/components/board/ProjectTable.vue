<template>
  <el-table
    :data="projects"
    highlight-current-row
    :current-row-key="selectedProjectKey"
    row-key="project_key"
    @current-change="onRowChange"
  >
    <el-table-column label="Key" prop="project_key" min-width="120" />
    <el-table-column label="名称" prop="name" min-width="120" />
    <el-table-column label="启用" min-width="90">
      <template #default="scope">
        <el-tag :type="scope.row.enabled ? 'success' : 'info'" size="small">
          {{ scope.row.enabled ? 'Yes' : 'No' }}
        </el-tag>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import type { ProjectRow } from '../../types/board'

defineProps<{
  projects: ProjectRow[]
  selectedProjectKey: string
}>()

const emit = defineEmits<{
  select: [row: ProjectRow | null]
}>()

function onRowChange(row: ProjectRow | null) {
  emit('select', row)
}
</script>
