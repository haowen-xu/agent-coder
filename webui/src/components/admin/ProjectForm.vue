<template>
  <el-form label-width="140px" class="project-form">
    <el-form-item label="project_key">
      <el-input v-model="projectForm.project_key" :disabled="Boolean(editingProjectKey)" placeholder="demo" />
    </el-form-item>
    <el-form-item label="project_slug">
      <el-input v-model="projectForm.project_slug" placeholder="group/repo" />
    </el-form-item>
    <el-form-item label="name">
      <el-input v-model="projectForm.name" placeholder="项目名" />
    </el-form-item>
    <el-form-item label="provider">
      <el-select v-model="projectForm.provider" style="width: 100%">
        <el-option label="gitlab" value="gitlab" />
      </el-select>
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
    <el-form-item label="sandbox_plan_review">
      <el-switch v-model="projectForm.sandbox_plan_review" />
      <span class="form-help">仅影响 plan/review 角色；dev/merge 固定关闭 sandbox。</span>
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
  </el-form>
</template>

<script setup lang="ts">
import type { UpsertProjectInput } from '../../stores/admin'

defineProps<{
  projectForm: UpsertProjectInput
  editingProjectKey: string
}>()
</script>

<style scoped>
.project-form {
  max-height: 62vh;
  overflow: auto;
  padding-right: 8px;
}

.form-help {
  margin-left: 12px;
  color: #5f6b7a;
  font-size: 12px;
}

@media (max-width: 768px) {
  .project-form {
    max-height: 56vh;
    overflow: visible;
  }
}
</style>
