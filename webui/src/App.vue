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
                    <el-form-item label="project_token">
                      <el-input
                        v-model="projectForm.project_token"
                        type="password"
                        show-password
                        placeholder="glpat-... (可选，优先于 credential_ref)"
                      />
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

            <el-row :gutter="16" class="panel-row">
              <el-col :span="24">
                <el-card class="admin-card" shadow="never">
                  <template #header>
                    <div class="panel-title">Prompt 管理</div>
                  </template>

                  <div class="form-inline-row mb-12">
                    <el-select v-model="promptForm.project_key" placeholder="选择项目" filterable style="width: 240px" @change="loadProjectPrompts">
                      <el-option
                        v-for="row in admin.projects"
                        :key="row.project_key"
                        :label="`${row.project_key} (${row.name})`"
                        :value="row.project_key"
                      />
                    </el-select>
                    <el-select v-model="promptForm.run_kind" style="width: 120px" @change="onPromptRunKindChange">
                      <el-option v-for="it in promptRunKindOptions" :key="it" :label="it" :value="it" />
                    </el-select>
                    <el-select v-model="promptForm.agent_role" style="width: 140px">
                      <el-option v-for="it in promptAgentRoleOptions" :key="it" :label="it" :value="it" />
                    </el-select>
                    <el-button :loading="admin.loading" @click="loadProjectPrompts">加载模板</el-button>
                    <el-tag :type="promptHasOverride ? 'warning' : 'info'" size="small">
                      {{ promptHasOverride ? 'project_override' : 'embedded_default' }}
                    </el-tag>
                  </div>

                  <el-row :gutter="16">
                    <el-col :xs="24" :lg="10">
                      <el-table :data="admin.projectPrompts" height="320" @row-click="fillPromptFromTemplate">
                        <el-table-column label="run_kind" prop="run_kind" width="100" />
                        <el-table-column label="agent_role" prop="agent_role" width="110" />
                        <el-table-column label="source" prop="source" width="150" />
                        <el-table-column label="内容预览" min-width="200">
                          <template #default="scope">
                            <span class="prompt-preview">{{ shortPrompt(scope.row.content) }}</span>
                          </template>
                        </el-table-column>
                      </el-table>
                    </el-col>
                    <el-col :xs="24" :lg="14">
                      <el-input
                        v-model="promptForm.content"
                        type="textarea"
                        :rows="18"
                        placeholder="Prompt 内容（markdown）"
                      />
                      <div class="form-inline-row mt-12">
                        <el-button type="primary" :loading="admin.loading" @click="savePromptOverride">保存覆盖</el-button>
                        <el-button :loading="admin.loading" @click="resetPromptOverride">回退默认</el-button>
                      </div>
                    </el-col>
                  </el-row>

                  <el-collapse class="mt-12">
                    <el-collapse-item title="查看默认模板（embedded）" name="defaults">
                      <el-table :data="admin.defaultPrompts" height="220">
                        <el-table-column label="run_kind" prop="run_kind" width="100" />
                        <el-table-column label="agent_role" prop="agent_role" width="110" />
                        <el-table-column label="内容预览" min-width="240">
                          <template #default="scope">
                            <span class="prompt-preview">{{ shortPrompt(scope.row.content) }}</span>
                          </template>
                        </el-table-column>
                      </el-table>
                    </el-collapse-item>
                  </el-collapse>
                </el-card>
              </el-col>
            </el-row>

            <el-row :gutter="16" class="panel-row">
              <el-col :span="24">
                <el-card class="admin-card" shadow="never">
                  <template #header>
                    <div class="panel-title">运维与日志</div>
                  </template>

                  <div class="form-inline-row mb-12">
                    <el-select
                      v-model="opsForm.project_key"
                      placeholder="选择项目"
                      filterable
                      style="width: 260px"
                      @change="onOpsProjectChange"
                    >
                      <el-option
                        v-for="row in admin.projects"
                        :key="row.project_key"
                        :label="`${row.project_key} (${row.name})`"
                        :value="row.project_key"
                      />
                    </el-select>
                    <el-button :loading="admin.loading" @click="refreshOpsMetrics">刷新指标</el-button>
                    <el-button :loading="admin.loading" @click="resetOpsProjectSyncCursor">重置同步游标</el-button>
                    <el-input-number v-model="opsForm.issue_id" :min="1" placeholder="issue id" />
                    <el-button :loading="admin.loading" @click="retryOpsIssue">重试 Issue</el-button>
                    <el-button :loading="admin.loading" @click="loadOpsIssueRuns">加载 Runs</el-button>
                    <el-input-number v-model="opsForm.run_id" :min="1" placeholder="run id" />
                    <el-button :loading="admin.loading" @click="loadOpsRunLogs">加载 Logs</el-button>
                  </div>

                  <div class="form-inline-row mb-12">
                    <el-input
                      v-model="opsForm.cancel_reason"
                      style="width: 360px"
                      placeholder="取消 run 的原因（可选）"
                    />
                    <el-button type="warning" :loading="admin.loading" @click="cancelOpsRun">取消 Run</el-button>
                  </div>

                  <el-row :gutter="16">
                    <el-col :xs="24" :lg="8">
                      <el-card shadow="never" class="ops-sub-card">
                        <template #header>
                          <div class="panel-title">项目 Issues</div>
                        </template>
                        <el-table :data="admin.projectIssues" height="260" @row-click="onOpsIssueRowClick">
                          <el-table-column label="ID" prop="id" width="80" />
                          <el-table-column label="IID" prop="issue_iid" width="90" />
                          <el-table-column label="Lifecycle" prop="lifecycle_status" min-width="140" />
                        </el-table>
                      </el-card>
                    </el-col>
                    <el-col :xs="24" :lg="8">
                      <el-card shadow="never" class="ops-sub-card">
                        <template #header>
                          <div class="panel-title">Issue Runs</div>
                        </template>
                        <el-table :data="admin.issueRuns" height="260" @row-click="onOpsRunRowClick">
                          <el-table-column label="RunID" prop="id" width="90" />
                          <el-table-column label="No" prop="run_no" width="70" />
                          <el-table-column label="Kind" prop="run_kind" width="90" />
                          <el-table-column label="Status" prop="status" min-width="120" />
                        </el-table>
                      </el-card>
                    </el-col>
                    <el-col :xs="24" :lg="8">
                      <el-card shadow="never" class="ops-sub-card">
                        <template #header>
                          <div class="panel-title">运行指标</div>
                        </template>
                        <el-descriptions v-if="admin.metrics" :column="1" size="small" border>
                          <el-descriptions-item label="项目">{{ admin.metrics.projects.total }} / enabled {{ admin.metrics.projects.enabled }}</el-descriptions-item>
                          <el-descriptions-item label="Issues">{{ admin.metrics.issues.total }}</el-descriptions-item>
                          <el-descriptions-item label="Runs">{{ admin.metrics.runs.total }}</el-descriptions-item>
                        </el-descriptions>
                        <pre v-if="admin.metrics" class="metrics-json">{{ JSON.stringify(admin.metrics, null, 2) }}</pre>
                      </el-card>
                    </el-col>
                  </el-row>

                  <el-table :data="admin.runLogs" height="280" class="mt-12">
                    <el-table-column label="Seq" prop="seq" width="80" />
                    <el-table-column label="At" prop="at" width="190" />
                    <el-table-column label="Level" prop="level" width="90" />
                    <el-table-column label="Stage" prop="stage" width="120" />
                    <el-table-column label="Event" prop="event_type" width="130" />
                    <el-table-column label="Message" prop="message" min-width="260" />
                  </el-table>
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
  type IssueRunRow,
  type PromptTemplate,
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
const promptRunKindOptions = ['dev', 'merge']
const promptForm = reactive({
  project_key: '',
  run_kind: 'dev',
  agent_role: 'dev',
  content: '',
})
const opsForm = reactive({
  project_key: '',
  issue_id: 0,
  run_id: 0,
  cancel_reason: 'manual cancel',
})

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
    project_token: '',
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
const promptAgentRoleOptions = computed(() => {
  if (promptForm.run_kind === 'merge') {
    return ['merge', 'review']
  }
  return ['dev', 'review']
})
const activePrompt = computed(() =>
  admin.projectPrompts.find(
    (p) => p.run_kind === promptForm.run_kind && p.agent_role === promptForm.agent_role,
  ),
)
const promptHasOverride = computed(() => activePrompt.value?.source === 'project_override')

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
  await admin.fetchDefaultPrompts(session.token)
  await admin.fetchMetrics(session.token)
  if (!promptForm.project_key && admin.projects.length > 0) {
    promptForm.project_key = admin.projects[0].project_key
  }
  if (!opsForm.project_key && admin.projects.length > 0) {
    opsForm.project_key = admin.projects[0].project_key
    await admin.fetchProjectIssues(session.token, opsForm.project_key, 200)
  }
  await loadProjectPrompts()
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

function ensurePromptRole() {
  if (!promptAgentRoleOptions.value.includes(promptForm.agent_role)) {
    promptForm.agent_role = promptAgentRoleOptions.value[0]
  }
}

function onPromptRunKindChange() {
  ensurePromptRole()
}

function shortPrompt(text: string) {
  const compact = text.replace(/\s+/g, ' ').trim()
  if (compact.length <= 120) {
    return compact
  }
  return `${compact.slice(0, 120)}...`
}

function fillPromptFromTemplate(row: PromptTemplate) {
  promptForm.run_kind = row.run_kind
  promptForm.agent_role = row.agent_role
  promptForm.content = row.content
  ensurePromptRole()
}

async function loadProjectPrompts() {
  if (!session.token || !promptForm.project_key) {
    admin.projectPrompts = []
    return
  }
  await admin.fetchProjectPrompts(session.token, promptForm.project_key)
  const exact = admin.projectPrompts.find(
    (p) => p.run_kind === promptForm.run_kind && p.agent_role === promptForm.agent_role,
  )
  if (exact) {
    promptForm.content = exact.content
    return
  }
  if (admin.projectPrompts.length > 0) {
    fillPromptFromTemplate(admin.projectPrompts[0])
  } else {
    promptForm.content = ''
  }
}

async function savePromptOverride() {
  if (!session.token || !promptForm.project_key) {
    ElMessage.warning('请先选择项目')
    return
  }
  if (!promptForm.content.trim()) {
    ElMessage.warning('Prompt 内容不能为空')
    return
  }
  await admin.upsertProjectPrompt(
    session.token,
    promptForm.project_key,
    promptForm.run_kind,
    promptForm.agent_role,
    promptForm.content,
  )
  await loadProjectPrompts()
  ElMessage.success('Prompt 覆盖已保存')
}

async function resetPromptOverride() {
  if (!session.token || !promptForm.project_key) {
    ElMessage.warning('请先选择项目')
    return
  }
  if (!promptHasOverride.value) {
    ElMessage.warning('当前组合没有项目覆盖，已是默认模板')
    return
  }
  await admin.deleteProjectPrompt(session.token, promptForm.project_key, promptForm.run_kind, promptForm.agent_role)
  await loadProjectPrompts()
  ElMessage.success('已回退到默认模板')
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
  projectForm.project_token = row.project_token ?? ''
  projectForm.poll_interval_sec = row.poll_interval_sec
  projectForm.enabled = row.enabled
  projectForm.label_agent_ready = row.label_agent_ready
  projectForm.label_in_progress = row.label_in_progress
  projectForm.label_human_review = row.label_human_review
  projectForm.label_rework = row.label_rework
  projectForm.label_verified = row.label_verified
  projectForm.label_merged = row.label_merged
}

async function onAdminProjectRowChange(row: AdminProjectRow | null) {
  if (!row) {
    return
  }
  editingProjectKey.value = row.project_key
  fillProjectForm(row)
  promptForm.project_key = row.project_key
  opsForm.project_key = row.project_key
  await admin.fetchProjectIssues(session.token ?? '', row.project_key, 200)
  await loadProjectPrompts()
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
  if (!promptForm.project_key && admin.projects.length > 0) {
    promptForm.project_key = admin.projects[0].project_key
  }
  if (!opsForm.project_key && admin.projects.length > 0) {
    opsForm.project_key = admin.projects[0].project_key
  }
}

async function onOpsProjectChange() {
  if (!session.token || !opsForm.project_key) {
    return
  }
  await admin.fetchProjectIssues(session.token, opsForm.project_key, 200)
}

function onOpsIssueRowClick(row: any) {
  opsForm.issue_id = Number(row?.id ?? 0)
}

function onOpsRunRowClick(row: IssueRunRow) {
  opsForm.run_id = row.id
}

async function loadOpsIssueRuns() {
  if (!session.token || !opsForm.issue_id) {
    ElMessage.warning('请先选择 Issue')
    return
  }
  await admin.fetchIssueRuns(session.token, opsForm.issue_id, 200)
}

async function loadOpsRunLogs() {
  if (!session.token || !opsForm.run_id) {
    ElMessage.warning('请先选择 Run')
    return
  }
  await admin.fetchRunLogs(session.token, opsForm.run_id, 1000)
}

async function retryOpsIssue() {
  if (!session.token || !opsForm.issue_id) {
    ElMessage.warning('请先选择 Issue')
    return
  }
  await admin.retryIssue(session.token, opsForm.issue_id)
  ElMessage.success('Issue 已重试入队')
  await admin.fetchProjectIssues(session.token, opsForm.project_key, 200)
}

async function cancelOpsRun() {
  if (!session.token || !opsForm.run_id) {
    ElMessage.warning('请先选择 Run')
    return
  }
  await admin.cancelRun(session.token, opsForm.run_id, opsForm.cancel_reason)
  ElMessage.success('Run 已取消')
  if (opsForm.issue_id) {
    await admin.fetchIssueRuns(session.token, opsForm.issue_id, 200)
  }
}

async function resetOpsProjectSyncCursor() {
  if (!session.token || !opsForm.project_key) {
    ElMessage.warning('请先选择项目')
    return
  }
  await admin.resetProjectSyncCursor(session.token, opsForm.project_key)
  ElMessage.success('项目同步游标已重置')
}

async function refreshOpsMetrics() {
  if (!session.token) {
    return
  }
  await admin.fetchMetrics(session.token)
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

.ops-sub-card {
  margin-bottom: 12px;
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

.prompt-preview {
  color: #5f6b7a;
  font-size: 12px;
  line-height: 1.4;
}

.metrics-json {
  margin: 12px 0 0;
  padding: 10px;
  max-height: 220px;
  overflow: auto;
  border-radius: 6px;
  background: #f6f8fb;
  border: 1px solid #e3e8f0;
  color: #3b4556;
  font-size: 12px;
  line-height: 1.45;
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
