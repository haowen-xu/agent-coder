<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :xs="24" :md="8">
      <el-card class="panel">
        <template #header>
          <div class="panel-title">项目列表</div>
        </template>
        <AsyncStateAlert :error="board.error" />
        <BoardProjectTable
          :projects="board.projects"
          :selected-project-key="board.selectedProjectKey"
          @select="onProjectRowChange"
        />
      </el-card>
    </el-col>

    <el-col :xs="24" :md="16">
      <el-card class="panel">
        <template #header>
          <div class="panel-title">
            Issue 列表
            <span v-if="board.selectedProjectKey">({{ board.selectedProjectKey }})</span>
          </div>
        </template>
        <BoardIssueTable :issues="board.issues" />
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useBoardStore, type ProjectRow } from '../stores/board'
import { useSessionStore } from '../stores/session'
import BoardIssueTable from '../components/board/IssueTable.vue'
import BoardProjectTable from '../components/board/ProjectTable.vue'
import AsyncStateAlert from '../components/common/AsyncStateAlert.vue'

const session = useSessionStore()
const board = useBoardStore()

async function refreshBoard() {
  if (!session.token) {
    return
  }
  await board.fetchProjects(session.token)
  if (board.selectedProjectKey) {
    await board.fetchIssues(session.token, board.selectedProjectKey)
  }
}

async function onProjectRowChange(row: ProjectRow | null) {
  if (!row || !session.token) {
    return
  }
  await board.fetchIssues(session.token, row.project_key)
}

onMounted(async () => {
  await refreshBoard()
})
</script>

<style scoped>
.panel-row {
  margin-top: 4px;
}

.panel {
  height: calc(100vh - 260px);
}

.panel-title {
  font-weight: 600;
}

@media (max-width: 768px) {
  .panel {
    height: auto;
  }
}
</style>
