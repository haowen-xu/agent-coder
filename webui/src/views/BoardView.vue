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
          <div class="panel-header">
            <div class="panel-title">
              Issue 列表
              <span v-if="board.selectedProjectKey">({{ board.selectedProjectKey }})</span>
            </div>
            <el-button size="small" :loading="board.loading" @click="refreshBoard">刷新</el-button>
          </div>
        </template>
        <BoardIssueTable :issues="board.issues" />
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useBoardStore, type ProjectRow } from '../stores/board'
import { useSessionStore } from '../stores/session'
import BoardIssueTable from '../components/board/IssueTable.vue'
import BoardProjectTable from '../components/board/ProjectTable.vue'
import AsyncStateAlert from '../components/common/AsyncStateAlert.vue'

const session = useSessionStore()
const board = useBoardStore()
const issuesRefreshEveryMs = 15_000
let issuesRefreshTimer: ReturnType<typeof setInterval> | null = null

async function refreshBoard() {
  if (!session.token) {
    return
  }
  try {
    await board.fetchProjects(session.token)
    if (board.selectedProjectKey) {
      await board.fetchIssues(session.token, board.selectedProjectKey)
    }
  } catch {
    // error state 已由 store 维护
  }
}

async function onProjectRowChange(row: ProjectRow | null) {
  if (!row || !session.token) {
    return
  }
  await board.fetchIssues(session.token, row.project_key)
}

function setupAutoRefresh() {
  if (issuesRefreshTimer !== null) {
    clearInterval(issuesRefreshTimer)
  }
  issuesRefreshTimer = setInterval(() => {
    void refreshSelectedProjectIssues()
  }, issuesRefreshEveryMs)
}

async function refreshSelectedProjectIssues() {
  if (!session.token || !board.selectedProjectKey) {
    return
  }
  try {
    await board.fetchIssues(session.token, board.selectedProjectKey)
  } catch {
    // error state 已由 store 维护
  }
}

onMounted(async () => {
  await refreshBoard()
  setupAutoRefresh()
})

onUnmounted(() => {
  if (issuesRefreshTimer !== null) {
    clearInterval(issuesRefreshTimer)
    issuesRefreshTimer = null
  }
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

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

@media (max-width: 768px) {
  .panel {
    height: auto;
  }
}
</style>
