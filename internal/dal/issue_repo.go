package db

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// GetIssueByProjectIID 是 *Client 的方法实现。
func (c *Client) GetIssueByProjectIID(ctx context.Context, projectID uint, issueIID int64) (*Issue, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row Issue
	err := c.db.WithContext(ctx).
		Where(&Issue{ProjectID: projectID, IssueIID: issueIID}).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query issue by project+iid")
	}
	return &row, nil
}

// GetIssueByID 是 *Client 的方法实现。
func (c *Client) GetIssueByID(ctx context.Context, issueID uint) (*Issue, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row Issue
	err := c.db.WithContext(ctx).Where("id = ?", issueID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query issue by id")
	}
	return &row, nil
}

// CreateIssue 是 *Client 的方法实现。
func (c *Client) CreateIssue(ctx context.Context, row *Issue) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Create(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "create issue")
	}
	return nil
}

// SaveIssue 是 *Client 的方法实现。
func (c *Client) SaveIssue(ctx context.Context, row *Issue) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Save(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "save issue")
	}
	return nil
}

// ListIssuesByProject 是 *Client 的方法实现。
func (c *Client) ListIssuesByProject(ctx context.Context, projectID uint, limit int) ([]Issue, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	var rows []Issue
	err := c.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list issues by project")
	}
	return rows, nil
}

// ListIssuesByLifecycle 是 *Client 的方法实现。
func (c *Client) ListIssuesByLifecycle(ctx context.Context, status string, limit int) ([]Issue, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	if limit <= 0 {
		limit = 100
	}
	var rows []Issue
	err := c.db.WithContext(ctx).
		Where("lifecycle_status = ?", status).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list issues by lifecycle")
	}
	return rows, nil
}

// ListIssuesForScheduling 是 *Client 的方法实现。
func (c *Client) ListIssuesForScheduling(ctx context.Context, limit int) ([]Issue, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	if limit <= 0 {
		limit = 200
	}
	var rows []Issue
	err := c.db.WithContext(ctx).
		Where("lifecycle_status IN ?", []string{IssueLifecycleRegistered, IssueLifecycleRework, IssueLifecycleVerified}).
		Order("registered_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list issues for scheduling")
	}
	return rows, nil
}

// TouchIssueSync 是 *Client 的方法实现。
func (c *Client) TouchIssueSync(ctx context.Context, issueID uint) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	err := c.db.WithContext(ctx).Model(&Issue{}).Where("id = ?", issueID).Update("last_synced_at", time.Now()).Error
	if err != nil {
		return xerr.Infra.Wrap(err, "touch issue sync")
	}
	return nil
}

// BindIssueRunIfIdle 是 *Client 的方法实现。
func (c *Client) BindIssueRunIfIdle(ctx context.Context, issueID uint, runID uint, branch string) (bool, error) {
	if c == nil || c.db == nil {
		return false, xerr.Infra.New("db is not initialized")
	}
	res := c.db.WithContext(ctx).Model(&Issue{}).
		Where("id = ? AND current_run_id IS NULL AND lifecycle_status IN ?", issueID, []string{
			IssueLifecycleRegistered,
			IssueLifecycleRework,
			IssueLifecycleVerified,
		}).
		Updates(map[string]any{
			"current_run_id":   runID,
			"lifecycle_status": IssueLifecycleRunning,
			"close_reason":     nil,
			"branch_name":      branch,
			"last_synced_at":   time.Now(),
			"updated_at":       time.Now(),
		})
	if res.Error != nil {
		return false, xerr.Infra.Wrap(res.Error, "bind issue run if idle")
	}
	return res.RowsAffected > 0, nil
}
