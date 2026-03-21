import { defineStore } from 'pinia'
import { ref } from 'vue'
import { listBoardIssuesApi, listBoardProjectsApi } from '../api/board'
import type { IssueRow, ProjectRow } from '../types/board'

export type { IssueRow, ProjectRow } from '../types/board'

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
      const resp = await listBoardProjectsApi(token)
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
      const resp = await listBoardIssuesApi(token, projectKey)
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
