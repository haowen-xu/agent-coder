package issue_run

import (
	"context"
	"strings"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/utils"
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

// ListIssueRuns 是 *Service 的方法实现。
func (s *Service) ListIssueRuns(ctx context.Context, issueID uint, limit int) ([]db.IssueRun, error) {
	issue, err := s.db.GetIssueByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, xerr.Config.New("issue not found")
	}
	return s.db.ListRunsByIssue(ctx, issueID, limit)
}

// ListRunLogs 是 *Service 的方法实现。
func (s *Service) ListRunLogs(ctx context.Context, runID uint, limit int) ([]db.RunLog, error) {
	run, err := s.db.GetRunByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, xerr.Config.New("run not found")
	}
	return s.db.ListRunLogsByRun(ctx, runID, limit)
}

// CancelRun 是 *Service 的方法实现。
func (s *Service) CancelRun(ctx context.Context, runID uint, reason string) (*db.IssueRun, error) {
	row, err := s.db.GetRunByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, xerr.Config.New("run not found")
	}
	if row.Status != db.RunStatusQueued && row.Status != db.RunStatusRunning {
		return nil, xerr.Config.New("run is not cancelable")
	}
	now := utils.NowUTC()
	row.Status = db.RunStatusCanceled
	row.FinishedAt = &now
	if strings.TrimSpace(reason) != "" {
		row.ErrorSummary = stringPtr("canceled by admin: " + strings.TrimSpace(reason))
	} else {
		row.ErrorSummary = stringPtr("canceled by admin")
	}
	if err := s.db.SaveRun(ctx, row); err != nil {
		return nil, err
	}

	issue, err := s.db.GetIssueByID(ctx, row.IssueID)
	if err != nil {
		return nil, err
	}
	if issue != nil && issue.CurrentRunID != nil && *issue.CurrentRunID == row.ID {
		issue.CurrentRunID = nil
		switch row.RunKind {
		case db.RunKindMerge:
			issue.LifecycleStatus = db.IssueLifecycleVerified
		default:
			issue.LifecycleStatus = db.IssueLifecycleRegistered
		}
		issue.CloseReason = nil
		if err := s.db.SaveIssue(ctx, issue); err != nil {
			return nil, err
		}
	}
	return row, nil
}

func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := strings.TrimSpace(v)
	return &s
}
