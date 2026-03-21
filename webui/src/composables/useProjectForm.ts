import { reactive, ref } from 'vue'
import type { AdminProjectRow, UpsertProjectInput } from '../stores/admin'

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
    sandbox_plan_review: false,
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

export function useProjectForm() {
  const editingProjectKey = ref('')
  const projectForm = reactive<UpsertProjectInput>(defaultProjectForm())

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
    projectForm.sandbox_plan_review = row.sandbox_plan_review
    projectForm.poll_interval_sec = row.poll_interval_sec
    projectForm.enabled = row.enabled
    projectForm.label_agent_ready = row.label_agent_ready
    projectForm.label_in_progress = row.label_in_progress
    projectForm.label_human_review = row.label_human_review
    projectForm.label_rework = row.label_rework
    projectForm.label_verified = row.label_verified
    projectForm.label_merged = row.label_merged
  }

  function startEditing(row: AdminProjectRow) {
    editingProjectKey.value = row.project_key
    fillProjectForm(row)
  }

  function resetProjectForm() {
    Object.assign(projectForm, defaultProjectForm())
    editingProjectKey.value = ''
  }

  return {
    projectForm,
    editingProjectKey,
    startEditing,
    resetProjectForm,
  }
}
