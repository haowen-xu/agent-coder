import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  cancelRunApi,
  createAdminProjectApi,
  createAdminUserApi,
  deleteProjectPromptApi,
  fetchAdminMetricsApi,
  listAdminProjectIssuesApi,
  listAdminProjectsApi,
  listAdminUsersApi,
  listDefaultPromptsApi,
  listIssueRunsApi,
  listProjectPromptsApi,
  listRunLogsApi,
  resetProjectSyncCursorApi,
  retryIssueApi,
  updateAdminProjectApi,
  updateAdminUserApi,
  upsertProjectPromptApi,
} from '../api/admin'
import type {
  AdminIssueRow,
  AdminProjectRow,
  AdminUserRow,
  CreateUserInput,
  IssueRunRow,
  OpsMetrics,
  PromptTemplate,
  RunLogRow,
  UpdateUserInput,
  UpsertProjectInput,
} from '../types/admin'

export type {
  AdminIssueRow,
  AdminProjectRow,
  AdminUserRow,
  CreateUserInput,
  IssueRunRow,
  OpsMetrics,
  PromptTemplate,
  RunLogRow,
  UpdateUserInput,
  UpsertProjectInput,
} from '../types/admin'

function buildProjectPayload(inProject: UpsertProjectInput) {
  return {
    ...inProject,
    issue_project_id: inProject.issue_project_id?.trim() ? inProject.issue_project_id.trim() : null,
    project_token: inProject.project_token?.trim() ? inProject.project_token.trim() : null,
  }
}

export const useAdminStore = defineStore('admin', () => {
  const loading = ref(false)
  const error = ref('')
  const users = ref<AdminUserRow[]>([])
  const projects = ref<AdminProjectRow[]>([])
  const defaultPrompts = ref<PromptTemplate[]>([])
  const projectPrompts = ref<PromptTemplate[]>([])
  const projectIssues = ref<AdminIssueRow[]>([])
  const issueRuns = ref<IssueRunRow[]>([])
  const runLogs = ref<RunLogRow[]>([])
  const metrics = ref<OpsMetrics | null>(null)

  async function fetchUsers(token: string) {
    loading.value = true
    error.value = ''
    try {
      const resp = await listAdminUsersApi(token)
      users.value = resp.items as AdminUserRow[]
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
      await createAdminUserApi(token, inUser)
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
      await updateAdminUserApi(token, userID, inUser)
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
      const resp = await listAdminProjectsApi(token)
      projects.value = resp.items as AdminProjectRow[]
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
      await createAdminProjectApi(token, buildProjectPayload(inProject))
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
      await updateAdminProjectApi(token, projectKey, buildProjectPayload(inProject))
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
      const resp = await listDefaultPromptsApi(token)
      defaultPrompts.value = resp.items as PromptTemplate[]
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
      const resp = await listProjectPromptsApi(token, projectKey)
      projectPrompts.value = resp.items as PromptTemplate[]
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
      await upsertProjectPromptApi(token, projectKey, runKind, agentRole, content)
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
      await deleteProjectPromptApi(token, projectKey, runKind, agentRole)
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchProjectIssues(token: string, projectKey: string, limit = 200) {
    loading.value = true
    error.value = ''
    try {
      const resp = await listAdminProjectIssuesApi(token, projectKey, limit)
      projectIssues.value = resp.items as AdminIssueRow[]
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchIssueRuns(token: string, issueID: number, limit = 100) {
    loading.value = true
    error.value = ''
    try {
      const resp = await listIssueRunsApi(token, issueID, limit)
      issueRuns.value = resp.items as IssueRunRow[]
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchRunLogs(token: string, runID: number, limit = 500) {
    loading.value = true
    error.value = ''
    try {
      const resp = await listRunLogsApi(token, runID, limit)
      runLogs.value = resp.items as RunLogRow[]
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function retryIssue(token: string, issueID: number) {
    loading.value = true
    error.value = ''
    try {
      await retryIssueApi(token, issueID)
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function cancelRun(token: string, runID: number, reason?: string) {
    loading.value = true
    error.value = ''
    try {
      await cancelRunApi(token, runID, reason)
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function resetProjectSyncCursor(token: string, projectKey: string) {
    loading.value = true
    error.value = ''
    try {
      await resetProjectSyncCursorApi(token, projectKey)
    } catch (err) {
      error.value = (err as Error).message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchMetrics(token: string) {
    loading.value = true
    error.value = ''
    try {
      metrics.value = (await fetchAdminMetricsApi(token)) as OpsMetrics
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
    projectIssues,
    issueRuns,
    runLogs,
    metrics,
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
    fetchProjectIssues,
    fetchIssueRuns,
    fetchRunLogs,
    retryIssue,
    cancelRun,
    resetProjectSyncCursor,
    fetchMetrics,
  }
})
