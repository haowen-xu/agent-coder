package common

import (
	"context"

	"github.com/haowen-xu/agent-coder/internal/dal"
)

// Client 定义仓库协作平台接口（覆盖 Issue/MR/Merge 能力），供 worker 统一调用。
// 具体实现可按平台差异扩展（如 GitLab），但语义需保持一致。
type Client interface {
	// ListIssues 按筛选条件拉取项目 issue 列表。
	// 返回数据需包含稳定标识，便于上层执行本地 upsert。
	ListIssues(ctx context.Context, project db.Project, opt ListIssuesOptions) ([]Issue, error)

	// SetIssueLabels 覆盖设置远端 issue 的标签集合。
	SetIssueLabels(ctx context.Context, project db.Project, issueIID int64, labels []string) error

	// CreateIssueNote 在远端 issue 下追加一条评论/备注。
	CreateIssueNote(ctx context.Context, project db.Project, issueIID int64, body string) error

	// UpsertIssueNote 使用 marker 幂等更新 issue 评论。
	// 若 marker 对应评论不存在，则创建新评论。
	UpsertIssueNote(ctx context.Context, project db.Project, issueIID int64, marker string, body string) error

	// CloseIssue 关闭远端 issue。
	CloseIssue(ctx context.Context, project db.Project, issueIID int64) error

	// EnsureMergeRequest 获取或创建对应分支的 MR。
	// 相同请求重复调用应保持幂等。
	EnsureMergeRequest(ctx context.Context, project db.Project, req CreateOrUpdateMRRequest) (*MergeRequest, error)

	// MergeMergeRequest 尝试合并指定 MR。
	// 若平台不允许自动合并，应返回 ErrNeedHumanMerge。
	MergeMergeRequest(ctx context.Context, project db.Project, mrIID int64) error
}
