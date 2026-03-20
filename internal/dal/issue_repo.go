package db

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

func (c *Client) GetIssueByProjectIID(ctx context.Context, projectID uint, issueIID int64) (*Issue, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row Issue
	err := c.db.WithContext(ctx).
		Where("project_id = ? AND issue_iid = ?", projectID, issueIID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query issue by project+iid")
	}
	return &row, nil
}

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

func (c *Client) CreateIssue(ctx context.Context, row *Issue) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Create(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "create issue")
	}
	return nil
}

func (c *Client) SaveIssue(ctx context.Context, row *Issue) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Save(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "save issue")
	}
	return nil
}

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
