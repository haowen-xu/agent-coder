package db

import (
	"context"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type metricCount struct {
	Name  string
	Count int64
}

func (c *Client) CountProjects(ctx context.Context) (total int64, enabled int64, err error) {
	if c == nil || c.db == nil {
		return 0, 0, xerr.Infra.New("db is not initialized")
	}
	if err = c.db.WithContext(ctx).Model(&Project{}).Count(&total).Error; err != nil {
		return 0, 0, xerr.Infra.Wrap(err, "count projects total")
	}
	if err = c.db.WithContext(ctx).Model(&Project{}).Where("enabled = ?", true).Count(&enabled).Error; err != nil {
		return 0, 0, xerr.Infra.Wrap(err, "count projects enabled")
	}
	return total, enabled, nil
}

func (c *Client) CountIssues(ctx context.Context) (int64, error) {
	if c == nil || c.db == nil {
		return 0, xerr.Infra.New("db is not initialized")
	}
	var total int64
	if err := c.db.WithContext(ctx).Model(&Issue{}).Count(&total).Error; err != nil {
		return 0, xerr.Infra.Wrap(err, "count issues total")
	}
	return total, nil
}

func (c *Client) CountRuns(ctx context.Context) (int64, error) {
	if c == nil || c.db == nil {
		return 0, xerr.Infra.New("db is not initialized")
	}
	var total int64
	if err := c.db.WithContext(ctx).Model(&IssueRun{}).Count(&total).Error; err != nil {
		return 0, xerr.Infra.Wrap(err, "count runs total")
	}
	return total, nil
}

func (c *Client) CountIssuesByLifecycle(ctx context.Context) (map[string]int64, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var rows []metricCount
	if err := c.db.WithContext(ctx).
		Model(&Issue{}).
		Select("lifecycle_status AS name, COUNT(*) AS count").
		Group("lifecycle_status").
		Scan(&rows).Error; err != nil {
		return nil, xerr.Infra.Wrap(err, "count issues by lifecycle")
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Name] = row.Count
	}
	return out, nil
}

func (c *Client) CountRunsByStatus(ctx context.Context) (map[string]int64, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var rows []metricCount
	if err := c.db.WithContext(ctx).
		Model(&IssueRun{}).
		Select("status AS name, COUNT(*) AS count").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, xerr.Infra.Wrap(err, "count runs by status")
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Name] = row.Count
	}
	return out, nil
}

func (c *Client) CountRunsByKind(ctx context.Context) (map[string]int64, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}
	var rows []metricCount
	if err := c.db.WithContext(ctx).
		Model(&IssueRun{}).
		Select("run_kind AS name, COUNT(*) AS count").
		Group("run_kind").
		Scan(&rows).Error; err != nil {
		return nil, xerr.Infra.Wrap(err, "count runs by kind")
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Name] = row.Count
	}
	return out, nil
}
