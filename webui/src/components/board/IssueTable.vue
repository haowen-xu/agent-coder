<template>
  <el-table :data="issues" height="520">
    <el-table-column label="IID" prop="issue_iid" width="90" />
    <el-table-column label="标题" prop="title" min-width="260" />
    <el-table-column label="State" prop="state" width="100" />
    <el-table-column label="Lifecycle" prop="lifecycle_status" width="140" />
    <el-table-column label="MR" width="100">
      <template #default="scope">
        <span v-if="scope.row.mr_iid">{{ scope.row.mr_iid }}</span>
        <span v-else>-</span>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import type { IssueRow } from '../../types/board'

defineProps<{
  issues: IssueRow[]
}>()
</script>
