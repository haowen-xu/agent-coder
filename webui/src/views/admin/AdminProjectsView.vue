<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :span="24">
      <el-card class="admin-card" shadow="never">
        <template #header>
          <div class="panel-header">
            <span class="panel-title">项目管理</span>
            <el-button type="primary" @click="openCreateDialog">创建项目</el-button>
          </div>
        </template>
        <AsyncStateAlert :error="admin.error" />
        <ProjectTable :projects="admin.projects" @edit="openEditDialog" @toggle-enabled="toggleProjectEnabled" />
      </el-card>
    </el-col>
  </el-row>

  <el-dialog
    v-model="projectDialogVisible"
    :title="editingProjectKey ? `编辑项目: ${editingProjectKey}` : '创建项目'"
    width="920px"
    destroy-on-close
  >
    <ProjectForm :project-form="projectForm" :editing-project-key="editingProjectKey" />
    <template #footer>
      <el-button @click="closeDialog">取消</el-button>
      <el-button type="primary" :loading="admin.loading" @click="saveProject">
        {{ editingProjectKey ? '保存更新' : '创建' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useBoardStore } from '../../stores/board'
import { useAdminStore, type AdminProjectRow } from '../../stores/admin'
import { useSessionStore } from '../../stores/session'
import { useProjectForm } from '../../composables/useProjectForm'
import ProjectTable from '../../components/admin/ProjectTable.vue'
import ProjectForm from '../../components/admin/ProjectForm.vue'
import AsyncStateAlert from '../../components/common/AsyncStateAlert.vue'

const session = useSessionStore()
const admin = useAdminStore()
const board = useBoardStore()
const { projectForm, editingProjectKey, startEditing, resetProjectForm } = useProjectForm()
const projectDialogVisible = ref(false)

async function refresh() {
  if (!session.token) {
    return
  }
  await admin.fetchProjects(session.token)
}

function openCreateDialog() {
  resetProjectForm()
  projectDialogVisible.value = true
}

function openEditDialog(row: AdminProjectRow) {
  startEditing(row)
  projectDialogVisible.value = true
}

function closeDialog() {
  projectDialogVisible.value = false
  resetProjectForm()
}

function toUpsertProjectInput(row: AdminProjectRow) {
  return {
    project_key: row.project_key,
    project_slug: row.project_slug,
    name: row.name,
    provider: row.provider,
    provider_url: row.provider_url,
    repo_url: row.repo_url,
    default_branch: row.default_branch,
    issue_project_id: row.issue_project_id ?? '',
    credential_ref: row.credential_ref,
    project_token: row.project_token ?? '',
    sandbox_plan_review: row.sandbox_plan_review,
    poll_interval_sec: row.poll_interval_sec,
    enabled: row.enabled,
    label_agent_ready: row.label_agent_ready,
    label_in_progress: row.label_in_progress,
    label_human_review: row.label_human_review,
    label_rework: row.label_rework,
    label_verified: row.label_verified,
    label_merged: row.label_merged,
  }
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
  closeDialog()
}

async function toggleProjectEnabled(row: AdminProjectRow) {
  if (!session.token) {
    return
  }
  const payload = toUpsertProjectInput(row)
  payload.enabled = !row.enabled
  await admin.updateProject(session.token, row.project_key, payload)
  await admin.fetchProjects(session.token)
  await board.fetchProjects(session.token)
}

onMounted(async () => {
  await refresh()
})
</script>

<style scoped>
.panel-row {
  margin-top: 4px;
}

.panel-title {
  font-weight: 600;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
