import { apiRequest } from './client'
import type { IssuesResp, ProjectsResp } from '../types/board'

export function listBoardProjectsApi(token: string) {
  return apiRequest<ProjectsResp>('/api/v1/board/projects', { token })
}

export function listBoardIssuesApi(token: string, projectKey: string) {
  return apiRequest<IssuesResp>(`/api/v1/board/projects/${encodeURIComponent(projectKey)}/issues`, {
    token,
  })
}

export type { IssueRow as BoardIssueRow, ProjectRow as BoardProjectRow } from '../types/board'
