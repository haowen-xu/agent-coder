package issue

import (
	"context"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// Service 表示数据结构定义。
type Service struct {
	db *db.Client // db 字段说明。
}

// New 执行相关逻辑。
func New(dbClient *db.Client) *Service {
	return &Service{db: dbClient}
}

// RetryIssue 是 *Service 的方法实现。
func (s *Service) RetryIssue(ctx context.Context, issueID uint) (*db.Issue, error) {
	issue, err := s.db.GetIssueByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, xerr.Config.New("issue not found")
	}
	if issue.LifecycleStatus == db.IssueLifecycleClosed {
		return nil, xerr.Config.New("issue is already closed")
	}
	active, err := s.db.GetActiveRunByIssue(ctx, issue.ID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, xerr.Config.New("issue has active run")
	}

	issue.CurrentRunID = nil
	issue.LifecycleStatus = db.IssueLifecycleRegistered
	issue.CloseReason = nil
	if err := s.db.SaveIssue(ctx, issue); err != nil {
		return nil, err
	}
	return issue, nil
}
