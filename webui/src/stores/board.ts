import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiRequest } from '../api'

export interface ProjectRow {
  id: number
  project_key: string
  project_slug: string
  name: string
  provider: string
  enabled: boolean
}

export interface IssueRow {
  id: number
  issue_iid: number
  title: string
  state: string
  lifecycle_status: string
  branch_name?: string
  mr_iid?: number
  mr_url?: string
  updated_at: string
}

interface ProjectsResp {
  items: ProjectRow[]
}

interface IssuesResp {
  project_key: string
  items: IssueRow[]
}

export const useBoardStore = defineStore('board', () => {
  const loading = ref(false)
  const error = ref('')
  const projects = ref<ProjectRow[]>([])
  const issues = ref<IssueRow[]>([])
  const selectedProjectKey = ref('')

  async function fetchProjects(token: string) {
    loading.value = true
    error.value = ''
    try {
      const resp = await apiRequest<ProjectsResp>('/api/v1/board/projects', { token })
      projects.value = resp.items
      if (!selectedProjectKey.value && resp.items.length > 0) {
        selectedProjectKey.value = resp.items[0].project_key
      }
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchIssues(token: string, projectKey: string) {
    if (!projectKey) {
      issues.value = []
      return
    }
    loading.value = true
    error.value = ''
    try {
      selectedProjectKey.value = projectKey
      const resp = await apiRequest<IssuesResp>(`/api/v1/board/projects/${encodeURIComponent(projectKey)}/issues`, {
        token,
      })
      issues.value = resp.items
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
    projects,
    issues,
    selectedProjectKey,
    fetchProjects,
    fetchIssues,
  }
})
