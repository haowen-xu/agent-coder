import { spawn } from 'node:child_process'
import { access, mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises'
import { constants as fsConstants, openSync } from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'
import YAML from 'yaml'

interface GitLabRuntimeConfig {
  apiBase: string
  projectID: string
  projectSlug: string
  projectToken: string
  projectURL: string
  repoURL: string
}

interface RuntimeState {
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

interface AdminProjectItem {
  project_key: string
}

interface GitLabProjectInfo {
  default_branch?: string
}

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const STATE_PATH = path.resolve(__dirname, '../.playwright-e2e-state.json')

function shellQuote(input: string): string {
  return `'${input.replace(/'/g, `'"'"'`)}'`
}

async function waitForHealthz(baseURL: string, timeoutMs = 90_000) {
  const startedAt = Date.now()
  let lastError = ''

  while (Date.now() - startedAt < timeoutMs) {
    try {
      const resp = await fetch(`${baseURL}/healthz`)
      if (resp.ok) {
        return
      }
      lastError = `healthz status=${resp.status}`
    } catch (err) {
      lastError = err instanceof Error ? err.message : String(err)
    }
    await new Promise((resolve) => setTimeout(resolve, 1_000))
  }

  throw new Error(`server health check timeout: ${lastError}`)
}

function spawnDetachedProcess(cmd: string, cwd: string, logPath: string): number {
  const fd = openSync(logPath, 'a')
  const child = spawn('zsh', ['-lc', cmd], {
    cwd,
    detached: true,
    stdio: ['ignore', fd, fd],
    env: process.env,
  })
  child.unref()
  if (!child.pid) {
    throw new Error(`failed to start process: ${cmd}`)
  }
  return child.pid
}

async function ensureExecutable(name: string) {
  const candidates = (process.env.PATH ?? '').split(path.delimiter)
  for (const dir of candidates) {
    if (!dir) {
      continue
    }
    const fullPath = path.join(dir, name)
    try {
      await access(fullPath, fsConstants.X_OK)
      return
    } catch {
      // continue
    }
  }
  throw new Error(`required executable not found in PATH: ${name}`)
}

async function loadDotEnvIfExists(repoRoot: string) {
  const envPath = path.join(repoRoot, '.env')
  let raw = ''
  try {
    raw = await readFile(envPath, 'utf-8')
  } catch (err) {
    const code =
      typeof err === 'object' && err !== null && 'code' in err ? String((err as { code?: string }).code ?? '') : ''
    if (code === 'ENOENT') {
      return
    }
    throw err
  }

  for (const row of raw.split('\n')) {
    const line = row.trim()
    if (!line || line.startsWith('#')) {
      continue
    }
    const idx = line.indexOf('=')
    if (idx <= 0) {
      continue
    }
    const key = line.slice(0, idx).trim()
    if (!key || process.env[key] !== undefined) {
      continue
    }
    const value = line
      .slice(idx + 1)
      .trim()
      .replace(/^['"]/, '')
      .replace(/['"]$/, '')
    process.env[key] = value
  }
}

async function requestJSON<T>(url: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type') && init.body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }

  const resp = await fetch(url, {
    ...init,
    headers,
  })

  const text = await resp.text()
  const data = text.trim() ? (JSON.parse(text) as Record<string, unknown>) : {}
  if (!resp.ok) {
    const msg = typeof data.error === 'string' ? data.error : `HTTP ${resp.status}`
    throw new Error(`${init.method ?? 'GET'} ${url} failed: ${msg}`)
  }
  return data as T
}

async function fetchGitLabDefaultBranch(gitLab: GitLabRuntimeConfig): Promise<string> {
  const row = await requestJSON<GitLabProjectInfo>(
    `${gitLab.apiBase}/projects/${encodeURIComponent(gitLab.projectID)}`,
    {
      headers: {
        'PRIVATE-TOKEN': gitLab.projectToken,
      },
    },
  )
  const branch = row.default_branch?.trim()
  if (!branch) {
    throw new Error(`gitlab project ${gitLab.projectID} has empty default_branch`)
  }
  return branch
}

function gitLabConfigFromEnv(): GitLabRuntimeConfig {
  const rawURL = (process.env.GITLAB_TESTBED_URL ?? 'https://git.ccf-quant.com/ai-agents/agent-coder-testbed').trim()
  const projectID = (process.env.GITLAB_TESTBED_PRJ_ID ?? '365').trim()
  const projectToken = (
    process.env.GITLAB_TESTBED_PRJ_TOKEN ??
    process.env.CODEX_GITLAB_TOKEN ??
    'glpat-p2hVy2Z6AyoMGjAJwbyeXG86MQp1OjFrCA.01.0y0sz78gm'
  ).trim()
  const configuredSlug = (process.env.GITLAB_TESTBED_PRJ_SLUG ?? '').trim()

  let apiBase = ''
  let projectSlug = configuredSlug
  let projectURL = ''

  if (/\/api\/v4\/?$/i.test(rawURL)) {
    apiBase = rawURL.replace(/\/+$/g, '')
    const projectURLBase = apiBase.replace(/\/api\/v4\/?$/i, '')
    if (!projectSlug) {
      projectSlug = 'ai-agents/agent-coder-testbed'
    }
    projectURL = `${projectURLBase}/${projectSlug}`
  } else {
    const parsed = new URL(rawURL)
    apiBase = `${parsed.protocol}//${parsed.host}/api/v4`
    if (!projectSlug) {
      projectSlug = parsed.pathname.replace(/^\/+|\/+$/g, '')
    }
    if (!projectSlug) {
      projectSlug = 'ai-agents/agent-coder-testbed'
    }
    projectURL = `${parsed.protocol}//${parsed.host}/${projectSlug}`
  }

  return {
    apiBase,
    projectID,
    projectSlug,
    projectToken,
    projectURL,
    repoURL: `${projectURL}.git`,
  }
}

async function ensureE2EProject(baseURL: string, adminToken: string, projectKey: string, gitLab: GitLabRuntimeConfig) {
  const gitLabDefaultBranch = await fetchGitLabDefaultBranch(gitLab)
  const headers = {
    Authorization: `Bearer ${adminToken}`,
    'Content-Type': 'application/json',
  }

  const payload = {
    project_key: projectKey,
    project_slug: gitLab.projectSlug,
    name: 'e2e-testbed',
    provider: 'gitlab',
    provider_url: gitLab.apiBase,
    repo_url: gitLab.repoURL,
    default_branch: gitLabDefaultBranch,
    issue_project_id: gitLab.projectID,
    credential_ref: '',
    project_token: gitLab.projectToken,
    sandbox_plan_review: false,
    poll_interval_sec: 5,
    enabled: true,
    label_agent_ready: 'Agent Ready',
    label_in_progress: 'In Progress',
    label_human_review: 'Human Review',
    label_rework: 'Rework',
    label_verified: 'Verified',
    label_merged: 'Merged',
  }

  const projectsResp = await requestJSON<{ items: AdminProjectItem[] }>(`${baseURL}/api/v1/admin/projects`, {
    headers,
  })
  const existed = projectsResp.items.find((it) => it.project_key === projectKey)

  if (existed) {
    await requestJSON(`${baseURL}/api/v1/admin/projects/${encodeURIComponent(projectKey)}`, {
      method: 'PUT',
      headers,
      body: JSON.stringify(payload),
    })
    return
  }

  await requestJSON(`${baseURL}/api/v1/admin/projects`, {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
  })
}

export default async function globalSetup() {
  await ensureExecutable('go')
  await ensureExecutable('codex')

  const repoRoot = path.resolve(__dirname, '../..')
  await loadDotEnvIfExists(repoRoot)
  const tmpDir = await mkdtemp(path.join(os.tmpdir(), 'agent-coder-pw-e2e-'))
  const dbPath = path.join(tmpDir, 'e2e.db')
  const workDir = path.join(tmpDir, 'workdir')
  const configPath = path.join(tmpDir, 'config.e2e.yaml')
  const serverLogPath = path.join(tmpDir, 'server.log')
  const workerLogPath = path.join(tmpDir, 'worker.log')
  const baseURL = process.env.PLAYWRIGHT_BASE_URL?.trim() || 'http://127.0.0.1:18080'

  const adminUsername = 'admin'
  const adminPassword = 'admin123'
  const e2eProjectKey = process.env.PLAYWRIGHT_E2E_PROJECT_KEY?.trim() || 'e2e-testbed'
  const gitLab = gitLabConfigFromEnv()

  await mkdir(workDir, { recursive: true })

  const config = {
    app: {
      name: 'agent-coder',
      env: 'e2e',
    },
    server: {
      host: '127.0.0.1',
      port: 18080,
      read_timeout: '30s',
      write_timeout: '30s',
      shutdown_timeout: '10s',
    },
    log: {
      level: 'info',
      format: 'text',
      add_source: false,
    },
    db: {
      enabled: true,
      driver: 'sqlite',
      sqlite_path: dbPath,
      postgres_dsn: '',
      max_open_conns: 20,
      max_idle_conns: 10,
      conn_max_lifetime: '30m',
      auto_migrate: true,
    },
    secret: {
      provider: 'env',
      env_prefix: 'AGENT_CODER_SECRET_',
    },
    auth: {
      session_ttl: '72h',
    },
    work: {
      work_dir: workDir,
    },
    agent: {
      provider: 'codex',
      codex: {
        binary: 'codex',
        sandbox: true,
        timeout_sec: 600,
        max_retry: 2,
        max_loop_step: 2,
      },
    },
    scheduler: {
      enabled: true,
      poll_interval_sec: 5,
      run_every: '5s',
    },
    repo_provider: {
      http_timeout_sec: 60,
    },
    issue_provider: {
      http_timeout_sec: 60,
    },
    bootstrap: {
      admin_username: adminUsername,
      admin_password: adminPassword,
    },
  }

  await writeFile(configPath, YAML.stringify(config), 'utf-8')

  const serverCmd = `go run ./cmds server --config ${shellQuote(configPath)}`
  const workerCmd = `go run ./cmds worker --config ${shellQuote(configPath)}`

  const serverPid = spawnDetachedProcess(serverCmd, repoRoot, serverLogPath)
  try {
    await waitForHealthz(baseURL)
  } catch (err) {
    try {
      process.kill(-serverPid, 'SIGTERM')
    } catch {
      process.kill(serverPid, 'SIGTERM')
    }
    throw err
  }

  const workerPid = spawnDetachedProcess(workerCmd, repoRoot, workerLogPath)

  const loginResp = await requestJSON<{ token: string }>(`${baseURL}/api/v1/auth/login`, {
    method: 'POST',
    body: JSON.stringify({
      username: adminUsername,
      password: adminPassword,
    }),
  })

  await ensureE2EProject(baseURL, loginResp.token, e2eProjectKey, gitLab)

  const state: RuntimeState = {
    baseURL,
    tmpDir,
    configPath,
    serverPid,
    workerPid,
    serverLogPath,
    workerLogPath,
    adminUsername,
    adminPassword,
    e2eProjectKey,
    gitLab,
  }

  await writeFile(STATE_PATH, JSON.stringify(state, null, 2), 'utf-8')
}
