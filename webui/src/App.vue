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
              <h1>Agent Coder</h1>
              <p>当前用户: {{ session.user?.username }} (admin: {{ session.user?.is_admin ? 'yes' : 'no' }})</p>
            </div>
            <div class="actions">
              <el-button :loading="board.loading || admin.loading" @click="refreshAll">刷新</el-button>
              <el-button type="danger" plain @click="session.logout">退出</el-button>
            </div>
          </div>
        </el-card>

        <el-tabs v-model="activeTab" class="main-tabs">
          <el-tab-pane label="展板" name="board">
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
                    @current-change="onBoardProjectRowChange"
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
                  <el-table :data="board.issues" height="520">
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
                </el-card>
              </el-col>
            </el-row>
          </el-tab-pane>

          <el-tab-pane v-if="isAdmin" label="管理台" name="admin">
            <el-row :gutter="16" class="panel-row">
              <el-col :xs="24" :xl="10">
                <el-card class="admin-card" shadow="never">
                  <template #header>
                    <div class="panel-title">用户管理</div>
                  </template>

                  <el-form label-position="top" class="compact-form" @submit.prevent="createAdminUser">
                    <el-form-item label="用户名">
                      <el-input v-model="newUserForm.username" placeholder="用户名" />
                    </el-form-item>
                    <el-form-item label="密码">
                      <el-input v-model="newUserForm.password" type="password" show-password placeholder="密码" />
                    </el-form-item>
                    <div class="form-inline-row">
                      <el-checkbox v-model="newUserForm.is_admin">管理员</el-checkbox>
                      <el-checkbox v-model="newUserForm.enabled">启用</el-checkbox>
                    </div>
                    <el-button type="primary" :loading="admin.loading" @click="createAdminUser">创建用户</el-button>
                  </el-form>

                  <el-alert
                    v-if="admin.error"
                    :closable="false"
                    type="error"
                    show-icon
                    :title="admin.error"
                    class="mb-12"
                  />

                  <el-table :data="admin.users" height="360" class="mt-12">
                    <el-table-column label="ID" prop="id" width="70" />
                    <el-table-column label="用户名" prop="username" min-width="120" />
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
                        <el-button link type="primary" @click="toggleUserAdmin(scope.row)">
                          {{ scope.row.is_admin ? '取消管理员' : '设为管理员' }}
                        </el-button>
                        <el-button link type="primary" @click="toggleUserEnabled(scope.row)">
                          {{ scope.row.enabled ? '禁用' : '启用' }}
                        </el-button>
                      </template>
                    </el-table-column>
                  </el-table>
                </el-card>
              </el-col>

              <el-col :xs="24" :xl="14">
                <el-card class="admin-card" shadow="never">
                  <template #header>
                    <div class="panel-title">项目管理</div>
                  </template>

                  <el-table
                    :data="admin.projects"
                    height="220"
                    highlight-current-row
                    row-key="project_key"
                    @current-change="onAdminProjectRowChange"
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
                  </el-table>

                  <el-divider />

                  <el-form label-width="140px" class="project-form" @submit.prevent="saveProject">
                    <el-form-item label="project_key">
                      <el-input v-model="projectForm.project_key" placeholder="demo" />
                    </el-form-item>
                    <el-form-item label="project_slug">
                      <el-input v-model="projectForm.project_slug" placeholder="group/repo" />
                    </el-form-item>
                    <el-form-item label="name">
                      <el-input v-model="projectForm.name" placeholder="项目名" />
                    </el-form-item>
                    <el-form-item label="provider">
                      <el-input v-model="projectForm.provider" placeholder="gitlab" />
                    </el-form-item>
                    <el-form-item label="provider_url">
                      <el-input v-model="projectForm.provider_url" placeholder="https://gitlab.example.com/api/v4" />
                    </el-form-item>
                    <el-form-item label="repo_url">
                      <el-input v-model="projectForm.repo_url" placeholder="git@..." />
                    </el-form-item>
                    <el-form-item label="default_branch">
                      <el-input v-model="projectForm.default_branch" placeholder="main" />
                    </el-form-item>
                    <el-form-item label="issue_project_id">
                      <el-input v-model="projectForm.issue_project_id" placeholder="可选" />
                    </el-form-item>
                    <el-form-item label="credential_ref">
                      <el-input v-model="projectForm.credential_ref" placeholder="gitlab_demo_token" />
                    </el-form-item>
                    <el-form-item label="poll_interval_sec">
                      <el-input-number v-model="projectForm.poll_interval_sec" :min="10" :max="3600" />
                    </el-form-item>
                    <el-form-item label="enabled">
                      <el-switch v-model="projectForm.enabled" />
                    </el-form-item>
                    <el-form-item label="label_agent_ready">
                      <el-input v-model="projectForm.label_agent_ready" />
                    </el-form-item>
                    <el-form-item label="label_in_progress">
                      <el-input v-model="projectForm.label_in_progress" />
                    </el-form-item>
                    <el-form-item label="label_human_review">
                      <el-input v-model="projectForm.label_human_review" />
                    </el-form-item>
                    <el-form-item label="label_rework">
                      <el-input v-model="projectForm.label_rework" />
                    </el-form-item>
                    <el-form-item label="label_verified">
                      <el-input v-model="projectForm.label_verified" />
                    </el-form-item>
                    <el-form-item label="label_merged">
                      <el-input v-model="projectForm.label_merged" />
                    </el-form-item>
                    <el-form-item>
                      <div class="form-inline-row">
                        <el-button type="primary" :loading="admin.loading" @click="saveProject">
                          {{ editingProjectKey ? '更新项目' : '创建项目' }}
                        </el-button>
                        <el-button @click="resetProjectForm">重置</el-button>
                        <span v-if="editingProjectKey" class="editing-tip">当前编辑: {{ editingProjectKey }}</span>
                      </div>
                    </el-form-item>
                  </el-form>
                </el-card>
              </el-col>
            </el-row>
          </el-tab-pane>
        </el-tabs>
      </template>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import { useBoardStore, type ProjectRow } from './stores/board'
import {
  useAdminStore,
  type AdminProjectRow,
  type AdminUserRow,
  type UpsertProjectInput,
} from './stores/admin'
import { useSessionStore } from './stores/session'

const session = useSessionStore()
const board = useBoardStore()
const admin = useAdminStore()

const activeTab = ref<'board' | 'admin'>('board')
const isAdmin = computed(() => !!session.user?.is_admin)
const editingProjectKey = ref('')

const loginForm = reactive({
  username: 'admin',
  password: 'admin123',
})

const newUserForm = reactive({
  username: '',
  password: '',
  is_admin: false,
  enabled: true,
})

function defaultProjectForm(): UpsertProjectInput {
  return {
    project_key: '',
    project_slug: '',
    name: '',
    provider: 'gitlab',
    provider_url: '',
    repo_url: '',
    default_branch: 'main',
    issue_project_id: '',
    credential_ref: '',
    poll_interval_sec: 60,
    enabled: true,
    label_agent_ready: 'Agent Ready',
    label_in_progress: 'In Progress',
    label_human_review: 'Human Review',
    label_rework: 'Rework',
    label_verified: 'Verified',
    label_merged: 'Merged',
  }
}

const projectForm = reactive<UpsertProjectInput>(defaultProjectForm())

async function onLogin() {
  await session.login(loginForm.username, loginForm.password)
  await refreshAll()
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

async function refreshAdmin() {
  if (!session.token || !isAdmin.value) {
    return
  }
  await admin.fetchUsers(session.token)
  await admin.fetchProjects(session.token)
}

async function refreshAll() {
  await refreshBoard()
  await refreshAdmin()
}

async function onBoardProjectRowChange(row: ProjectRow | null) {
  if (!row || !session.token) {
    return
  }
  await board.fetchIssues(session.token, row.project_key)
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

function fillProjectForm(row: AdminProjectRow) {
  projectForm.project_key = row.project_key
  projectForm.project_slug = row.project_slug
  projectForm.name = row.name
  projectForm.provider = row.provider
  projectForm.provider_url = row.provider_url
  projectForm.repo_url = row.repo_url
  projectForm.default_branch = row.default_branch
  projectForm.issue_project_id = row.issue_project_id ?? ''
  projectForm.credential_ref = row.credential_ref
  projectForm.poll_interval_sec = row.poll_interval_sec
  projectForm.enabled = row.enabled
  projectForm.label_agent_ready = row.label_agent_ready
  projectForm.label_in_progress = row.label_in_progress
  projectForm.label_human_review = row.label_human_review
  projectForm.label_rework = row.label_rework
  projectForm.label_verified = row.label_verified
  projectForm.label_merged = row.label_merged
}

function onAdminProjectRowChange(row: AdminProjectRow | null) {
  if (!row) {
    return
  }
  editingProjectKey.value = row.project_key
  fillProjectForm(row)
}

function resetProjectForm() {
  Object.assign(projectForm, defaultProjectForm())
  editingProjectKey.value = ''
}

async function saveProject() {
  if (!session.token) {
    return
  }
  if (editingProjectKey.value) {
    await admin.updateProject(session.token, editingProjectKey.value, projectForm)
    ElMessage.success('项目已更新')
  } else {
    await admin.createProject(session.token, projectForm)
    ElMessage.success('项目已创建')
  }
  await admin.fetchProjects(session.token)
  await board.fetchProjects(session.token)
}

onMounted(async () => {
  if (!session.token) {
    return
  }
  await session.fetchMe()
  if (session.isLoggedIn) {
    await refreshAll()
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

.main-tabs {
  margin-top: 8px;
}

.panel-row {
  margin-top: 4px;
}

.panel {
  height: calc(100vh - 210px);
}

.panel-title {
  font-weight: 600;
}

.admin-card {
  margin-bottom: 16px;
}

.compact-form {
  margin-bottom: 12px;
}

.project-form {
  max-height: 480px;
  overflow: auto;
  padding-right: 8px;
}

.form-inline-row {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.editing-tip {
  color: #5f6b7a;
  font-size: 12px;
}

.mb-12 {
  margin-bottom: 12px;
}

.mt-12 {
  margin-top: 12px;
}

@media (max-width: 768px) {
  .panel {
    height: auto;
  }

  .project-form {
    max-height: none;
    overflow: visible;
  }
}
</style>
