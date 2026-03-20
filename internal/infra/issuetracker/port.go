package issuetracker

import (
	"context"

	"github.com/haowen-xu/agent-coder/internal/dal"
)

type Client interface {
	ListIssues(ctx context.Context, project db.Project, opt ListIssuesOptions) ([]Issue, error)
	SetIssueLabels(ctx context.Context, project db.Project, issueIID int64, labels []string) error
	CreateIssueNote(ctx context.Context, project db.Project, issueIID int64, body string) error
	CloseIssue(ctx context.Context, project db.Project, issueIID int64) error
	EnsureMergeRequest(ctx context.Context, project db.Project, req CreateOrUpdateMRRequest) (*MergeRequest, error)
	MergeMergeRequest(ctx context.Context, project db.Project, mrIID int64) error
}
