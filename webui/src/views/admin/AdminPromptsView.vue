<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :span="24">
      <el-card class="admin-card" shadow="never">
        <template #header>
          <div class="panel-header">
            <span class="panel-title">Prompt 管理</span>
            <el-button type="primary" @click="openCreateDialog">创建覆盖</el-button>
          </div>
        </template>

        <div class="toolbar-row">
          <ProjectSelect
            v-model="selectedProjectKey"
            :projects="projects"
            width="280px"
            @change="loadProjectPrompts"
          />
          <el-button :loading="loading" @click="loadProjectPrompts">刷新</el-button>
        </div>

        <AsyncStateAlert :error="admin.error" />

        <el-table :data="projectPrompts" height="560">
          <el-table-column label="run_kind" prop="run_kind" width="120" />
          <el-table-column label="agent_role" prop="agent_role" width="120" />
          <el-table-column label="source" prop="source" width="170" />
          <el-table-column label="内容预览" min-width="360">
            <template #default="scope">
              <span class="prompt-preview">{{ shortPrompt(scope.row.content) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" min-width="180" fixed="right">
            <template #default="scope">
              <el-button link type="primary" @click="openEditDialog(scope.row)">编辑</el-button>
              <el-button link type="danger" @click="deletePromptOverride(scope.row)">回退默认</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </el-col>
  </el-row>

  <el-dialog
    v-model="promptDialogVisible"
    :title="isEditMode ? '编辑 Prompt 覆盖' : '创建 Prompt 覆盖'"
    width="760px"
    destroy-on-close
  >
    <el-form label-position="top">
      <el-form-item label="项目">
        <ProjectSelect v-model="promptForm.project_key" :projects="projects" width="100%" />
      </el-form-item>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12">
          <el-form-item label="run_kind">
            <el-select v-model="promptForm.run_kind" style="width: 100%">
              <el-option v-for="it in promptRunKindOptions" :key="it" :label="it" :value="it" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :xs="24" :sm="12">
          <el-form-item label="agent_role">
            <el-select v-model="promptForm.agent_role" style="width: 100%">
              <el-option v-for="it in promptAgentRoleOptions" :key="it" :label="it" :value="it" />
            </el-select>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="内容">
        <el-input
          v-model="promptForm.content"
          type="textarea"
          :rows="16"
          placeholder="请输入 Prompt 覆盖内容（markdown）"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="promptDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="loading" @click="savePromptOverride">
        {{ isEditMode ? '保存更新' : '创建覆盖' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useAdminStore, type PromptTemplate } from '../../stores/admin'
import { useSessionStore } from '../../stores/session'
import AsyncStateAlert from '../../components/common/AsyncStateAlert.vue'
import ProjectSelect from '../../components/common/ProjectSelect.vue'

const session = useSessionStore()
const admin = useAdminStore()
const { loading, projects, projectPrompts } = storeToRefs(admin)

const selectedProjectKey = ref('')
const promptDialogVisible = ref(false)
const isEditMode = ref(false)

const promptRunKindOptions = ['dev', 'merge']
const promptForm = reactive({
  project_key: '',
  run_kind: 'dev',
  agent_role: 'dev',
  content: '',
})

const promptAgentRoleOptions = computed(() => {
  if (promptForm.run_kind === 'merge') {
    return ['merge', 'review']
  }
  return ['dev', 'review']
})

watch(
  () => promptForm.run_kind,
  () => {
    if (!promptAgentRoleOptions.value.includes(promptForm.agent_role)) {
      promptForm.agent_role = promptAgentRoleOptions.value[0]
    }
  },
)

function shortPrompt(text: string) {
  const compact = text.replace(/\s+/g, ' ').trim()
  if (compact.length <= 120) {
    return compact
  }
  return `${compact.slice(0, 120)}...`
}

async function loadProjectPrompts() {
  if (!session.token || !selectedProjectKey.value) {
    admin.projectPrompts = []
    return
  }
  await admin.fetchProjectPrompts(session.token, selectedProjectKey.value)
}

function openCreateDialog() {
  promptForm.project_key = selectedProjectKey.value
  promptForm.run_kind = 'dev'
  promptForm.agent_role = 'dev'
  promptForm.content = ''
  isEditMode.value = false
  promptDialogVisible.value = true
}

function openEditDialog(row: PromptTemplate) {
  promptForm.project_key = selectedProjectKey.value
  promptForm.run_kind = row.run_kind
  promptForm.agent_role = row.agent_role
  promptForm.content = row.content
  isEditMode.value = true
  promptDialogVisible.value = true
}

async function savePromptOverride() {
  if (!session.token) {
    return
  }
  if (!promptForm.project_key) {
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
  selectedProjectKey.value = promptForm.project_key
  await loadProjectPrompts()
  promptDialogVisible.value = false
  ElMessage.success(isEditMode.value ? 'Prompt 已更新' : 'Prompt 覆盖已创建')
}

async function deletePromptOverride(row: PromptTemplate) {
  if (!session.token || !selectedProjectKey.value) {
    return
  }
  await ElMessageBox.confirm(
    `确认回退 ${row.run_kind}/${row.agent_role} 的项目覆盖吗？`,
    '回退确认',
    {
      type: 'warning',
      confirmButtonText: '确认回退',
      cancelButtonText: '取消',
    },
  )
  await admin.deleteProjectPrompt(session.token, selectedProjectKey.value, row.run_kind, row.agent_role)
  await loadProjectPrompts()
  ElMessage.success('已回退到默认模板')
}

onMounted(async () => {
  if (!session.token) {
    return
  }
  await admin.fetchProjects(session.token)
  if (projects.value.length > 0) {
    selectedProjectKey.value = projects.value[0].project_key
  }
  await loadProjectPrompts()
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

.toolbar-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.prompt-preview {
  color: #5f6b7a;
  font-size: 12px;
  line-height: 1.4;
}
</style>
