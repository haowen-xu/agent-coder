import { apiRequest } from './client'
import type {
  AdminIssueRow,
  DefaultPromptsResp,
  IssueRunsResp,
  OpsMetrics,
  ProjectIssuesResp,
  ProjectPromptsResp,
  ProjectsResp,
  RunLogsResp,
  UsersResp,
} from '../types/admin'

export function listAdminUsersApi(token: string) {
  return apiRequest<UsersResp>('/api/v1/admin/users', { token })
}

export function createAdminUserApi(token: string, body: unknown) {
  return apiRequest('/api/v1/admin/users', { method: 'POST', token, body })
}

export function updateAdminUserApi(token: string, userID: number, body: unknown) {
  return apiRequest(`/api/v1/admin/users/${userID}`, { method: 'PUT', token, body })
}

export function listAdminProjectsApi(token: string) {
  return apiRequest<ProjectsResp>('/api/v1/admin/projects', { token })
}

export function createAdminProjectApi(token: string, body: unknown) {
  return apiRequest('/api/v1/admin/projects', { method: 'POST', token, body })
}

export function updateAdminProjectApi(token: string, projectKey: string, body: unknown) {
  return apiRequest(`/api/v1/admin/projects/${encodeURIComponent(projectKey)}`, {
    method: 'PUT',
    token,
    body,
  })
}

export function listDefaultPromptsApi(token: string) {
  return apiRequest<DefaultPromptsResp>('/api/v1/admin/prompts/defaults', { token })
}

export function listProjectPromptsApi(token: string, projectKey: string) {
  return apiRequest<ProjectPromptsResp>(`/api/v1/admin/projects/${encodeURIComponent(projectKey)}/prompts`, {
    token,
  })
}

export function upsertProjectPromptApi(
  token: string,
  projectKey: string,
  runKind: string,
  agentRole: string,
  content: string,
) {
  return apiRequest(
    `/api/v1/admin/projects/${encodeURIComponent(projectKey)}/prompts/${encodeURIComponent(runKind)}/${encodeURIComponent(agentRole)}`,
    {
      method: 'PUT',
      token,
      body: { content },
    },
  )
}

export function deleteProjectPromptApi(token: string, projectKey: string, runKind: string, agentRole: string) {
  return apiRequest(
    `/api/v1/admin/projects/${encodeURIComponent(projectKey)}/prompts/${encodeURIComponent(runKind)}/${encodeURIComponent(agentRole)}`,
    {
      method: 'DELETE',
      token,
    },
  )
}

export function listAdminProjectIssuesApi(token: string, projectKey: string, limit = 200) {
  return apiRequest<ProjectIssuesResp<AdminIssueRow>>(
    `/api/v1/admin/projects/${encodeURIComponent(projectKey)}/issues?limit=${limit}`,
    { token },
  )
}

export function listIssueRunsApi(token: string, issueID: number, limit = 100) {
  return apiRequest<IssueRunsResp>(`/api/v1/admin/issues/${issueID}/runs?limit=${limit}`, { token })
}

export function listRunLogsApi(token: string, runID: number, limit = 500) {
  return apiRequest<RunLogsResp>(`/api/v1/admin/runs/${runID}/logs?limit=${limit}`, { token })
}

export function retryIssueApi(token: string, issueID: number) {
  return apiRequest(`/api/v1/admin/issues/${issueID}/retry`, {
    method: 'POST',
    token,
  })
}

export function cancelRunApi(token: string, runID: number, reason?: string) {
  return apiRequest(`/api/v1/admin/runs/${runID}/cancel`, {
    method: 'POST',
    token,
    body: reason?.trim() ? { reason: reason.trim() } : {},
  })
}

export function resetProjectSyncCursorApi(token: string, projectKey: string) {
  return apiRequest(`/api/v1/admin/projects/${encodeURIComponent(projectKey)}/reset-sync-cursor`, {
    method: 'POST',
    token,
  })
}

export function fetchAdminMetricsApi(token: string) {
  return apiRequest<OpsMetrics>('/api/v1/admin/metrics', { token })
}
