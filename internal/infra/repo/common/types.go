package common

import (
	"time"
)

// Issue 描述远端 issue 的核心字段快照。
type Issue struct {
	IID       int64      // IID 是项目内 issue 编号。
	UID       string     // UID 是平台全局唯一标识（若平台提供）。
	Title     string     // Title 是 issue 标题。
	State     string     // State 是远端状态（如 open/closed）。
	Labels    []string   // Labels 是当前标签列表。
	WebURL    string     // WebURL 是 issue 页面地址。
	ClosedAt  *time.Time // ClosedAt 是远端关闭时间。
	UpdatedAt time.Time  // UpdatedAt 是远端最后更新时间。
}

// ListIssuesOptions 定义 issue 列表查询参数。
type ListIssuesOptions struct {
	State        string     // State 是远端筛选状态（如 opened/all）。
	UpdatedAfter *time.Time // UpdatedAfter 用于增量同步游标过滤。
	PerPage      int        // PerPage 是单页数量上限。
	MaxPages     int        // MaxPages 是最大翻页数。
}

// MergeRequest 描述远端 MR 的关键字段。
type MergeRequest struct {
	IID          int64  // IID 是项目内 MR 编号。
	WebURL       string // WebURL 是 MR 页面地址。
	SourceBranch string // SourceBranch 是源分支。
	TargetBranch string // TargetBranch 是目标分支。
	State        string // State 是 MR 当前状态。
}

// CreateOrUpdateMRRequest 描述创建或复用 MR 的输入参数。
type CreateOrUpdateMRRequest struct {
	SourceBranch string // SourceBranch 是源分支。
	TargetBranch string // TargetBranch 是目标分支。
	Title        string // Title 是 MR 标题。
	Description  string // Description 是 MR 描述。
}

// ValidateURLArgs 定义 repo URL 校验输入参数。
type ValidateURLArgs struct {
	ProviderURL  string // ProviderURL 是仓库协作平台 API endpoint。
	RepoURL      string // RepoURL 是代码仓库地址。
	ProjectToken string // ProjectToken 是用于访问平台 API 的 token。
}

// ValidateURLResult 定义 repo URL 校验结果。
type ValidateURLResult struct {
	ProjectID   string // ProjectID 是平台项目 ID（例如 GitLab 的 project id）。
	ProjectSlug string // ProjectSlug 是平台项目路径（例如 group/repo）。
}
