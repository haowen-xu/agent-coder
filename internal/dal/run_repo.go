package db

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/haowen-xu/agent-coder/internal/utils"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// GetMaxRunNo 是方法实现。
func (c *Client) GetMaxRunNo(ctx context.Context, issueID uint) (int, error) {
	if c == nil || c.db == nil {
		return 0, xerr.Infra.New("db is not initialized")
	}
	var row IssueRun
	err := c.db.WithContext(ctx).
		Where("issue_id = ?", issueID).
		Order("run_no DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, xerr.Infra.Wrap(err, "get max run_no")
	}
	return row.RunNo, nil
}

// GetActiveRunByIssue 是方法实现。
func (c *Client) GetActiveRunByIssue(ctx context.Context, issueID uint) (*IssueRun, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row IssueRun
	err := c.db.WithContext(ctx).
		Where("issue_id = ? AND status IN ?", issueID, []string{RunStatusQueued, RunStatusRunning}).
		Order("id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "get active run by issue")
	}
	return &row, nil
}

// CreateRun 是方法实现。
func (c *Client) CreateRun(ctx context.Context, row *IssueRun) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Create(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "create issue run")
	}
	return nil
}

// GetRunByID 是方法实现。
func (c *Client) GetRunByID(ctx context.Context, id uint) (*IssueRun, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row IssueRun
	err := c.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query run by id")
	}
	return &row, nil
}

// GetNextQueuedRun 是方法实现。
func (c *Client) GetNextQueuedRun(ctx context.Context) (*IssueRun, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row IssueRun
	err := c.db.WithContext(ctx).
		Where("status = ?", RunStatusQueued).
		Order("queued_at ASC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "get next queued run")
	}
	return &row, nil
}

// ClaimNextQueuedRun 是方法实现。
func (c *Client) ClaimNextQueuedRun(ctx context.Context) (*IssueRun, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}

	for attempt := 0; attempt < 5; attempt++ {
		var claimed *IssueRun
		lostRace := false

		err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var row IssueRun
			queryErr := tx.
				Where("status = ?", RunStatusQueued).
				Order("queued_at ASC").
				First(&row).Error
			if errors.Is(queryErr, gorm.ErrRecordNotFound) {
				return nil
			}
			if queryErr != nil {
				return queryErr
			}

			now := utils.NowUTC()
			res := tx.Model(&IssueRun{}).
				Where("id = ? AND status = ?", row.ID, RunStatusQueued).
				Updates(map[string]any{
					"status":     RunStatusRunning,
					"started_at": &now,
					"updated_at": now,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				lostRace = true
				return nil
			}

			row.Status = RunStatusRunning
			row.StartedAt = &now
			claimed = &row
			return nil
		})
		if err != nil {
			return nil, xerr.Infra.Wrap(err, "claim next queued run")
		}
		if claimed != nil {
			return claimed, nil
		}
		if !lostRace {
			return nil, nil
		}
	}

	return nil, nil
}

// SaveRun 是方法实现。
func (c *Client) SaveRun(ctx context.Context, row *IssueRun) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Save(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "save issue run")
	}
	return nil
}

// AppendRunLog 是方法实现。
func (c *Client) AppendRunLog(ctx context.Context, row *RunLog) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if row.At.IsZero() {
		row.At = utils.NowUTC()
	}
	if err := c.db.WithContext(ctx).Create(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "append run log")
	}
	return nil
}

// GetNextRunSeq 是方法实现。
func (c *Client) GetNextRunSeq(ctx context.Context, runID uint) (int, error) {
	if c == nil || c.db == nil {
		return 0, xerr.Infra.New("db is not initialized")
	}
	var row RunLog
	err := c.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("seq DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 1, nil
	}
	if err != nil {
		return 0, xerr.Infra.Wrap(err, "get next run log seq")
	}
	return row.Seq + 1, nil
}

// CountIssueRunsByStatus 是方法实现。
func (c *Client) CountIssueRunsByStatus(ctx context.Context, issueID uint, statuses []string) (int64, error) {
	if c == nil || c.db == nil {
		return 0, xerr.Infra.New("db is not initialized")
	}
	var cnt int64
	err := c.db.WithContext(ctx).
		Model(&IssueRun{}).
		Where("issue_id = ? AND status IN ?", issueID, statuses).
		Count(&cnt).Error
	if err != nil {
		return 0, xerr.Infra.Wrap(err, "count issue runs by status")
	}
	return cnt, nil
}

// CountIssueRunsByStatusAndKind 是方法实现。
func (c *Client) CountIssueRunsByStatusAndKind(ctx context.Context, issueID uint, runKind string, statuses []string) (int64, error) {
	if c == nil || c.db == nil {
		return 0, xerr.Infra.New("db is not initialized")
	}
	var cnt int64
	err := c.db.WithContext(ctx).
		Model(&IssueRun{}).
		Where("issue_id = ? AND run_kind = ? AND status IN ?", issueID, runKind, statuses).
		Count(&cnt).Error
	if err != nil {
		return 0, xerr.Infra.Wrap(err, "count issue runs by status and kind")
	}
	return cnt, nil
}

// ListRunsByIssue 是方法实现。
func (c *Client) ListRunsByIssue(ctx context.Context, issueID uint, limit int) ([]IssueRun, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	var rows []IssueRun
	err := c.db.WithContext(ctx).
		Where("issue_id = ?", issueID).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list runs by issue")
	}
	return rows, nil
}

// ListRunLogsByRun 是方法实现。
func (c *Client) ListRunLogsByRun(ctx context.Context, runID uint, limit int) ([]RunLog, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	if limit <= 0 {
		limit = 500
	}
	var rows []RunLog
	err := c.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("seq DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list run logs by run")
	}
	// Return in ascending seq for UI timeline.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, nil
}
