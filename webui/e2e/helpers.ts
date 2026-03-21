import { readFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import type { APIRequestContext, Locator, Page } from '@playwright/test'

export interface GitLabRuntimeConfig {
  apiBase: string
  projectID: string
  projectSlug: string
  projectToken: string
  projectURL: string
  repoURL: string
}

export interface RuntimeState {
  baseURL: string
  tmpDir: string
  configPath: string
  serverPid: number
  workerPid: number
  serverLogPath: string
  workerLogPath: string
  adminUsername: string
  adminPassword: string
  e2eProjectKey: string
  gitLab: GitLabRuntimeConfig
}

export interface AdminProjectItem {
  id: number
  project_key: string
  sandbox_plan_review: boolean
}

export interface AdminIssueItem {
  id: number
  issue_iid: number
  title: string
  lifecycle_status: string
  mr_iid?: number
}

export interface AdminRunItem {
  id: number
  run_no: number
  run_kind: string
  status: string
  error_summary?: string
}

export interface RunLogItem {
  id: number
  run_id: number
  seq: number
  message: string
}

export interface GitLabIssue {
  iid: number
  title: string
  web_url: string
}

export interface GitLabIssueNote {
  id: number
  body: string
}

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const STATE_PATH = path.resolve(__dirname, '../.playwright-e2e-state.json')

export async function readRuntimeState(): Promise<RuntimeState> {
  const text = await readFile(STATE_PATH, 'utf-8')
  return JSON.parse(text) as RuntimeState
}

export async function apiRequest<T>(
  request: APIRequestContext,
  baseURL: string,
  pathName: string,
  options: {
    method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
    token?: string
    body?: unknown
  } = {},
): Promise<T> {
  const method = options.method ?? 'GET'
  const headers: Record<string, string> = {}
  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`
  }
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  const resp = await request.fetch(`${baseURL}${pathName}`, {
    method,
    headers,
    data: options.body,
  })

  const text = await resp.text()
  const data = text.trim() ? (JSON.parse(text) as Record<string, unknown>) : {}
  if (!resp.ok()) {
    const message = typeof data.error === 'string' ? data.error : `HTTP ${resp.status()}`
    throw new Error(`${method} ${pathName} failed: ${message}`)
  }
  return data as T
}

export async function loginAdmin(request: APIRequestContext, state: RuntimeState): Promise<string> {
  const row = await apiRequest<{ token: string }>(request, state.baseURL, '/api/v1/auth/login', {
    method: 'POST',
    body: {
      username: state.adminUsername,
      password: state.adminPassword,
    },
  })
  return row.token
}

export async function loginFromUI(page: Page, state: RuntimeState) {
  await page.goto(`${state.baseURL}/login`)
  await page.getByPlaceholder('admin').fill(state.adminUsername)
  await page.getByPlaceholder('******').fill(state.adminPassword)
  await page.getByRole('button', { name: '登录' }).click()
}

export async function listAdminProjects(request: APIRequestContext, state: RuntimeState, token: string) {
  return apiRequest<{ items: AdminProjectItem[] }>(request, state.baseURL, '/api/v1/admin/projects', {
    token,
  })
}

export async function listAdminProjectIssues(
  request: APIRequestContext,
  state: RuntimeState,
  token: string,
  projectKey: string,
) {
  return apiRequest<{ project_key: string; items: AdminIssueItem[] }>(
    request,
    state.baseURL,
    `/api/v1/admin/projects/${encodeURIComponent(projectKey)}/issues?limit=200`,
    { token },
  )
}

export async function listIssueRuns(
  request: APIRequestContext,
  state: RuntimeState,
  token: string,
  issueID: number,
) {
  return apiRequest<{ issue_id: number; items: AdminRunItem[] }>(
    request,
    state.baseURL,
    `/api/v1/admin/issues/${issueID}/runs?limit=100`,
    { token },
  )
}

export async function listRunLogs(
  request: APIRequestContext,
  state: RuntimeState,
  token: string,
  runID: number,
) {
  return apiRequest<{ run_id: number; items: RunLogItem[] }>(
    request,
    state.baseURL,
    `/api/v1/admin/runs/${runID}/logs?limit=1000`,
    { token },
  )
}

export async function setElSwitchState(target: Locator, expected: boolean) {
  const checked = await target.evaluate((el) => el.classList.contains('is-checked'))
  if (checked !== expected) {
    await target.click()
  }
}

export async function selectProjectByKey(selectLocator: Locator, projectKey: string) {
  const text = (await selectLocator.textContent()) ?? ''
  if (text.includes(projectKey)) {
    return
  }

  await selectLocator.click()
  const option = selectLocator.page().locator('.el-select-dropdown__item').filter({ hasText: projectKey }).first()
  await option.click()
}

async function gitLabRequest<T>(
  state: RuntimeState,
  pathName: string,
  init: RequestInit,
): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('PRIVATE-TOKEN', state.gitLab.projectToken)
  if (!headers.has('Content-Type') && init.body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }

  const url = `${state.gitLab.apiBase}${pathName}`
  const resp = await fetch(url, {
    ...init,
    headers,
  })

  const text = await resp.text()
  const data = text.trim() ? (JSON.parse(text) as Record<string, unknown>) : {}
  if (!resp.ok) {
    const msg = typeof data.message === 'string' ? data.message : `HTTP ${resp.status}`
    throw new Error(`${init.method ?? 'GET'} ${url} failed: ${msg}`)
  }

  return data as T
}

export async function createGitLabIssue(
  state: RuntimeState,
  title: string,
  description: string,
  labels: string[],
): Promise<GitLabIssue> {
  const pathName = `/projects/${encodeURIComponent(state.gitLab.projectID)}/issues`
  return gitLabRequest<GitLabIssue>(state, pathName, {
    method: 'POST',
    body: JSON.stringify({
      title,
      description,
      labels: labels.join(','),
    }),
  })
}

export async function closeGitLabIssue(state: RuntimeState, issueIID: number) {
  const params = new URLSearchParams({ state_event: 'close' })
  const pathName = `/projects/${encodeURIComponent(state.gitLab.projectID)}/issues/${issueIID}?${params.toString()}`
  await gitLabRequest<Record<string, unknown>>(state, pathName, {
    method: 'PUT',
  })
}

export async function listGitLabIssueNotes(state: RuntimeState, issueIID: number): Promise<GitLabIssueNote[]> {
  const pathName = `/projects/${encodeURIComponent(state.gitLab.projectID)}/issues/${issueIID}/notes?per_page=100`
  return gitLabRequest<GitLabIssueNote[]>(state, pathName, {
    method: 'GET',
  })
}

export async function pollUntil<T>(
  action: () => Promise<T>,
  done: (value: T) => boolean,
  options: {
    timeoutMs: number
    intervalMs: number
    description: string
  },
): Promise<T> {
  const startedAt = Date.now()
  let lastValue: T | null = null

  while (Date.now() - startedAt < options.timeoutMs) {
    const value = await action()
    lastValue = value
    if (done(value)) {
      return value
    }
    await new Promise((resolve) => setTimeout(resolve, options.intervalMs))
  }

  throw new Error(`timeout waiting for ${options.description}; last value=${JSON.stringify(lastValue)}`)
}
