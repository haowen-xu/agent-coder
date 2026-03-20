import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiRequest } from '../api'

export interface AdminUserRow {
  id: number
  username: string
  is_admin: boolean
  enabled: boolean
  last_login_at?: string
  created_at?: string
  updated_at?: string
}

export interface AdminProjectRow {
  id: number
  project_key: string
  project_slug: string
  name: string
  provider: string
  provider_url: string
  repo_url: string
  default_branch: string
  issue_project_id?: string
  credential_ref: string
  poll_interval_sec: number
  enabled: boolean
  last_issue_sync_at?: string
  label_agent_ready: string
  label_in_progress: string
  label_human_review: string
  label_rework: string
  label_verified: string
  label_merged: string
  created_by: number
  created_at?: string
  updated_at?: string
}

export interface PromptTemplate {
  project_key?: string
  run_kind: string
  agent_role: string
  source: string
  content: string
}

export interface CreateUserInput {
  username: string
  password: string
  is_admin: boolean
  enabled: boolean
}

export interface UpdateUserInput {
  password?: string
  is_admin?: boolean
  enabled?: boolean
}

export interface UpsertProjectInput {
  project_key: string
  project_slug: string
  name: string
  provider: string
  provider_url: string
  repo_url: string
  default_branch: string
  issue_project_id?: string
  credential_ref: string
  poll_interval_sec: number
  enabled: boolean
  label_agent_ready: string
  label_in_progress: string
  label_human_review: string
  label_rework: string
  label_verified: string
  label_merged: string
}

interface UsersResp {
  items: AdminUserRow[]
}

interface ProjectsResp {
  items: AdminProjectRow[]
}

interface DefaultPromptsResp {
  items: PromptTemplate[]
}

interface ProjectPromptsResp {
  project_key: string
  items: PromptTemplate[]
}

function buildProjectPayload(inProject: UpsertProjectInput) {
  return {
    ...inProject,
    issue_project_id: inProject.issue_project_id?.trim() ? inProject.issue_project_id.trim() : null,
  }
}

export const useAdminStore = defineStore('admin', () => {
  const loading = ref(false)
  const error = ref('')
  const users = ref<AdminUserRow[]>([])
  const projects = ref<AdminProjectRow[]>([])
  const defaultPrompts = ref<PromptTemplate[]>([])
  const projectPrompts = ref<PromptTemplate[]>([])

  async function fetchUsers(token: string) {
    loading.value = true
    error.value = ''
    try {
      const resp = await apiRequest<UsersResp>('/api/v1/admin/users', { token })
      users.value = resp.items
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createUser(token: string, inUser: CreateUserInput) {
    loading.value = true
    error.value = ''
    try {
      await apiRequest('/api/v1/admin/users', {
        method: 'POST',
        token,
        body: inUser,
      })
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateUser(token: string, userID: number, inUser: UpdateUserInput) {
    loading.value = true
    error.value = ''
    try {
      await apiRequest(`/api/v1/admin/users/${userID}`, {
        method: 'PUT',
        token,
        body: inUser,
      })
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchProjects(token: string) {
    loading.value = true
    error.value = ''
    try {
      const resp = await apiRequest<ProjectsResp>('/api/v1/admin/projects', { token })
      projects.value = resp.items
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createProject(token: string, inProject: UpsertProjectInput) {
    loading.value = true
    error.value = ''
    try {
      await apiRequest('/api/v1/admin/projects', {
        method: 'POST',
        token,
        body: buildProjectPayload(inProject),
      })
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateProject(token: string, projectKey: string, inProject: UpsertProjectInput) {
    loading.value = true
    error.value = ''
    try {
      await apiRequest(`/api/v1/admin/projects/${encodeURIComponent(projectKey)}`, {
        method: 'PUT',
        token,
        body: buildProjectPayload(inProject),
      })
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchDefaultPrompts(token: string) {
    loading.value = true
    error.value = ''
    try {
      const resp = await apiRequest<DefaultPromptsResp>('/api/v1/admin/prompts/defaults', { token })
      defaultPrompts.value = resp.items
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchProjectPrompts(token: string, projectKey: string) {
    loading.value = true
    error.value = ''
    try {
      const resp = await apiRequest<ProjectPromptsResp>(`/api/v1/admin/projects/${encodeURIComponent(projectKey)}/prompts`, {
        token,
      })
      projectPrompts.value = resp.items
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function upsertProjectPrompt(
    token: string,
    projectKey: string,
    runKind: string,
    agentRole: string,
    content: string,
  ) {
    loading.value = true
    error.value = ''
    try {
      await apiRequest(`/api/v1/admin/projects/${encodeURIComponent(projectKey)}/prompts/${encodeURIComponent(runKind)}/${encodeURIComponent(agentRole)}`, {
        method: 'PUT',
        token,
        body: { content },
      })
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteProjectPrompt(token: string, projectKey: string, runKind: string, agentRole: string) {
    loading.value = true
    error.value = ''
    try {
      await apiRequest(`/api/v1/admin/projects/${encodeURIComponent(projectKey)}/prompts/${encodeURIComponent(runKind)}/${encodeURIComponent(agentRole)}`, {
        method: 'DELETE',
        token,
      })
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    users,
    projects,
    defaultPrompts,
    projectPrompts,
    fetchUsers,
    createUser,
    updateUser,
    fetchProjects,
    createProject,
    updateProject,
    fetchDefaultPrompts,
    fetchProjectPrompts,
    upsertProjectPrompt,
    deleteProjectPrompt,
  }
})
