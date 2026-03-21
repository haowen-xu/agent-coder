<template>
  <el-row :gutter="16" class="panel-row">
    <el-col :span="24">
      <el-card class="admin-card" shadow="never">
        <template #header>
          <div class="panel-title">Prompt 管理</div>
        </template>
        <PromptPanel
          :projects="projects"
          :project-prompts="projectPrompts"
          :default-prompts="defaultPrompts"
          :prompt-form="promptForm"
          :prompt-run-kind-options="promptRunKindOptions"
          :prompt-agent-role-options="promptAgentRoleOptions"
          :prompt-has-override="promptHasOverride"
          :loading="loading"
          @load="loadProjectPrompts"
          @run-kind-change="onPromptRunKindChange"
          @save="savePromptOverride"
          @reset="resetPromptOverride"
          @select-template="fillPromptFromTemplate"
        />
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { onMounted } from 'vue'
import { useAdminStore } from '../../stores/admin'
import { usePromptEditor } from '../../composables/usePromptEditor'
import PromptPanel from '../../components/admin/PromptPanel.vue'

const admin = useAdminStore()
const { loading } = storeToRefs(admin)
const {
  projects,
  projectPrompts,
  defaultPrompts,
  promptRunKindOptions,
  promptForm,
  promptAgentRoleOptions,
  promptHasOverride,
  fillPromptFromTemplate,
  onPromptRunKindChange,
  loadProjectPrompts,
  savePromptOverride,
  resetPromptOverride,
  initPromptEditor,
} = usePromptEditor()

onMounted(async () => {
  await initPromptEditor()
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
