package db

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// ListProjects 是 *Client 的方法实现。
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var rows []Project
	err := c.db.WithContext(ctx).Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list projects")
	}
	return rows, nil
}

// ListEnabledProjects 是 *Client 的方法实现。
func (c *Client) ListEnabledProjects(ctx context.Context) ([]Project, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var rows []Project
	err := c.db.WithContext(ctx).Where("enabled = ?", true).Order("id ASC").Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list enabled projects")
	}
	return rows, nil
}

// GetProjectByKey 是 *Client 的方法实现。
func (c *Client) GetProjectByKey(ctx context.Context, projectKey string) (*Project, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row Project
	err := c.db.WithContext(ctx).Where("project_key = ?", projectKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query project by key")
	}
	return &row, nil
}

// GetProjectByID 是 *Client 的方法实现。
func (c *Client) GetProjectByID(ctx context.Context, id uint) (*Project, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var row Project
	err := c.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query project by id")
	}
	return &row, nil
}

// CreateProject 是 *Client 的方法实现。
func (c *Client) CreateProject(ctx context.Context, row *Project) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Create(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "create project")
	}
	return nil
}

// SaveProject 是 *Client 的方法实现。
func (c *Client) SaveProject(ctx context.Context, row *Project) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}
	if err := c.db.WithContext(ctx).Save(row).Error; err != nil {
		return xerr.Infra.Wrap(err, "save project")
	}
	return nil
}

// ResetProjectSyncCursorByKey 是 *Client 的方法实现。
func (c *Client) ResetProjectSyncCursorByKey(ctx context.Context, projectKey string) (*Project, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	key := strings.TrimSpace(projectKey)
	if key == "" {
		return nil, xerr.Config.New("project_key is required")
	}
	row, err := c.GetProjectByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	row.LastIssueSyncAt = nil
	if err := c.SaveProject(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}
