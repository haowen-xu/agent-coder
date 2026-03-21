<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :xs="24" :xl="10">
      <el-card class="admin-card" shadow="never">
        <template #header>
          <div class="panel-title">项目列表</div>
        </template>
        <AsyncStateAlert :error="admin.error" />
        <ProjectTable :projects="admin.projects" @select="onProjectRowChange" />
      </el-card>
    </el-col>

    <el-col :xs="24" :xl="14">
      <el-card class="admin-card" shadow="never">
        <template #header>
          <div class="panel-title">项目编辑</div>
        </template>
        <ProjectForm
          :project-form="projectForm"
          :editing-project-key="editingProjectKey"
          :loading="admin.loading"
          @save="saveProject"
          @reset="resetProjectForm"
        />
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
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

async function refresh() {
  if (!session.token) {
    return
  }
  await admin.fetchProjects(session.token)
}

function onProjectRowChange(row: AdminProjectRow | null) {
  if (!row) {
    return
  }
  startEditing(row)
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
</style>
