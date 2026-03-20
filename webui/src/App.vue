<template>
  <el-container class="page">
    <el-main class="content">
      <el-card v-if="!session.isLoggedIn" class="login-card" shadow="hover">
        <template #header>
          <div class="login-header">
            <h1>Agent Coder</h1>
            <p>登录后查看项目展板</p>
          </div>
        </template>
        <el-form label-position="top" @submit.prevent="onLogin">
          <el-form-item label="用户名">
            <el-input v-model="loginForm.username" placeholder="admin" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="loginForm.password" type="password" placeholder="******" show-password />
          </el-form-item>
          <el-alert
            v-if="session.error"
            :closable="false"
            type="error"
            show-icon
            :title="session.error"
            class="mb-12"
          />
          <el-button type="primary" :loading="session.loading" @click="onLogin">登录</el-button>
        </el-form>
      </el-card>

      <template v-else>
        <el-card class="topbar" shadow="never">
          <div class="topbar-inner">
            <div>
              <h1>Agent Coder Board</h1>
              <p>当前用户: {{ session.user?.username }} (admin: {{ session.user?.is_admin ? 'yes' : 'no' }})</p>
            </div>
            <div class="actions">
              <el-button :loading="board.loading" @click="refreshBoard">刷新</el-button>
              <el-button type="danger" plain @click="session.logout">退出</el-button>
            </div>
          </div>
        </el-card>

        <el-row :gutter="16" class="panel-row">
          <el-col :xs="24" :md="8">
            <el-card class="panel">
              <template #header>
                <div class="panel-title">项目列表</div>
              </template>
              <el-alert
                v-if="board.error"
                :closable="false"
                type="error"
                show-icon
                :title="board.error"
                class="mb-12"
              />
              <el-table
                :data="board.projects"
                highlight-current-row
                :current-row-key="board.selectedProjectKey"
                row-key="project_key"
                @current-change="onProjectRowChange"
              >
                <el-table-column label="Key" prop="project_key" min-width="120" />
                <el-table-column label="名称" prop="name" min-width="120" />
                <el-table-column label="启用" min-width="80">
                  <template #default="scope">
                    <el-tag :type="scope.row.enabled ? 'success' : 'info'" size="small">
                      {{ scope.row.enabled ? 'Yes' : 'No' }}
                    </el-tag>
                  </template>
                </el-table-column>
              </el-table>
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
              <el-table :data="board.issues" height="460">
                <el-table-column label="IID" prop="issue_iid" width="90" />
                <el-table-column label="标题" prop="title" min-width="260" />
                <el-table-column label="State" prop="state" width="100" />
                <el-table-column label="Lifecycle" prop="lifecycle_status" width="140" />
                <el-table-column label="MR" width="90">
                  <template #default="scope">
                    <span v-if="scope.row.mr_iid">{{ scope.row.mr_iid }}</span>
                    <span v-else>-</span>
                  </template>
                </el-table-column>
              </el-table>
            </el-card>
          </el-col>
        </el-row>
      </template>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { useBoardStore, type ProjectRow } from './stores/board'
import { useSessionStore } from './stores/session'

const session = useSessionStore()
const board = useBoardStore()

const loginForm = reactive({
  username: 'admin',
  password: 'admin123',
})

async function onLogin() {
  await session.login(loginForm.username, loginForm.password)
  await refreshBoard()
}

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
  if (!session.token) {
    return
  }
  await session.fetchMe()
  if (session.isLoggedIn) {
    await refreshBoard()
  }
})
</script>

<style scoped>
.page {
  min-height: 100vh;
  background:
    radial-gradient(circle at 15% 20%, #edf5ff 0, transparent 42%),
    radial-gradient(circle at 85% 0, #e9f8f1 0, transparent 30%),
    linear-gradient(140deg, #f6f9fc 0%, #eef3f9 100%);
}

.content {
  padding: 20px;
}

.login-card {
  width: min(460px, 100%);
  margin: 8vh auto 0;
}

.login-header h1 {
  margin: 0;
  font-size: 28px;
}

.login-header p {
  margin: 6px 0 0;
  color: #5f6b7a;
}

.topbar {
  margin-bottom: 16px;
}

.topbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.topbar-inner h1 {
  margin: 0;
  font-size: 22px;
}

.topbar-inner p {
  margin: 6px 0 0;
  color: #5f6b7a;
}

.actions {
  display: flex;
  gap: 8px;
}

.panel-row {
  margin-top: 4px;
}

.panel {
  height: calc(100vh - 170px);
}

.panel-title {
  font-weight: 600;
}

.mb-12 {
  margin-bottom: 12px;
}
</style>
