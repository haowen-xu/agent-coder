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
  project_token?: string
  sandbox_plan_review: boolean
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

export interface AdminIssueRow {
  id: number
  issue_iid: number
  lifecycle_status: string
  title?: string
  state?: string
}

export interface IssueRunRow {
  id: number
  issue_id: number
  run_no: number
  run_kind: string
  trigger_type: string
  status: string
  agent_role: string
  loop_step: number
  max_loop_step: number
  queued_at: string
  started_at?: string
  finished_at?: string
  branch_name: string
  mr_iid?: number
  mr_url?: string
  conflict_retry_count: number
  max_conflict_retry: number
  error_summary?: string
  created_at: string
  updated_at: string
}

export interface RunLogRow {
  id: number
  run_id: number
  seq: number
  at: string
  level: string
  stage: string
  event_type: string
  message: string
  payload_json?: string
}

export interface OpsMetrics {
  timestamp: string
  projects: {
    total: number
    enabled: number
  }
  issues: {
    total: number
    by_lifecycle: Record<string, number>
  }
  runs: {
    total: number
    by_status: Record<string, number>
    by_kind: Record<string, number>
  }
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
  project_token?: string
  sandbox_plan_review: boolean
  poll_interval_sec: number
  enabled: boolean
  label_agent_ready: string
  label_in_progress: string
  label_human_review: string
  label_rework: string
  label_verified: string
  label_merged: string
}

export interface UsersResp {
  items: AdminUserRow[]
}

export interface ProjectsResp {
  items: AdminProjectRow[]
}

export interface DefaultPromptsResp {
  items: PromptTemplate[]
}

export interface ProjectPromptsResp {
  project_key: string
  items: PromptTemplate[]
}

export interface ProjectIssuesResp<TIssue> {
  project_key: string
  items: TIssue[]
}

export interface IssueRunsResp {
  issue_id: number
  items: IssueRunRow[]
}

export interface RunLogsResp {
  run_id: number
  items: RunLogRow[]
}
