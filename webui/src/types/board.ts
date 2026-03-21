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

export interface ProjectsResp {
  items: ProjectRow[]
}

export interface IssuesResp {
  project_key: string
  items: IssueRow[]
}
