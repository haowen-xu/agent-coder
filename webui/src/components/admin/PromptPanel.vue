<template>
  <div class="form-inline-row mb-12">
    <ProjectSelect
      v-model="promptForm.project_key"
      :projects="projects"
      width="240px"
      @change="emit('load')"
    />
    <el-select v-model="promptForm.run_kind" style="width: 120px" @change="emit('runKindChange')">
      <el-option v-for="it in promptRunKindOptions" :key="it" :label="it" :value="it" />
    </el-select>
    <el-select v-model="promptForm.agent_role" style="width: 140px">
      <el-option v-for="it in promptAgentRoleOptions" :key="it" :label="it" :value="it" />
    </el-select>
    <el-button :loading="loading" @click="emit('load')">加载模板</el-button>
    <el-tag :type="promptHasOverride ? 'warning' : 'info'" size="small">
      {{ promptHasOverride ? 'project_override' : 'embedded_default' }}
    </el-tag>
  </div>

  <el-row :gutter="16">
    <el-col :xs="24" :lg="10">
      <el-table :data="projectPrompts" height="360" @row-click="onRowClick">
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
        <el-button type="primary" :loading="loading" @click="emit('save')">保存覆盖</el-button>
        <el-button :loading="loading" @click="emit('reset')">回退默认</el-button>
      </div>
    </el-col>
  </el-row>

  <el-collapse class="mt-12">
    <el-collapse-item title="查看默认模板（embedded）" name="defaults">
      <el-table :data="defaultPrompts" height="260">
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
</template>

<script setup lang="ts">
import type { AdminProjectRow, PromptTemplate } from '../../stores/admin'
import ProjectSelect from '../common/ProjectSelect.vue'

defineProps<{
  projects: AdminProjectRow[]
  projectPrompts: PromptTemplate[]
  defaultPrompts: PromptTemplate[]
  promptForm: {
    project_key: string
    run_kind: string
    agent_role: string
    content: string
  }
  promptRunKindOptions: string[]
  promptAgentRoleOptions: string[]
  promptHasOverride: boolean
  loading: boolean
}>()

const emit = defineEmits<{
  load: []
  runKindChange: []
  save: []
  reset: []
  selectTemplate: [row: PromptTemplate]
}>()

function shortPrompt(text: string) {
  const compact = text.replace(/\s+/g, ' ').trim()
  if (compact.length <= 120) {
    return compact
  }
  return `${compact.slice(0, 120)}...`
}

function onRowClick(row: PromptTemplate) {
  emit('selectTemplate', row)
}
</script>

<style scoped>
.form-inline-row {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.prompt-preview {
  color: #5f6b7a;
  font-size: 12px;
  line-height: 1.4;
}

.mb-12 {
  margin-bottom: 12px;
}

.mt-12 {
  margin-top: 12px;
}
</style>
