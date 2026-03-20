package db

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// ListPromptTemplatesByProject 是 *Client 的方法实现。
func (c *Client) ListPromptTemplatesByProject(ctx context.Context, projectKey string) ([]PromptTemplate, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}

	var rows []PromptTemplate
	err := c.db.WithContext(ctx).
		Where("project_key = ?", projectKey).
		Order("run_kind ASC, agent_role ASC").
		Find(&rows).Error
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "list prompt templates by project")
	}
	return rows, nil
}

// UpsertPromptTemplate 是 *Client 的方法实现。
func (c *Client) UpsertPromptTemplate(
	ctx context.Context,
	projectKey string,
	runKind string,
	agentRole string,
	content string,
) (*PromptTemplate, error) {
	if c == nil || c.db == nil {
		return nil, xerr.Infra.New("db is not initialized")
	}

	tx := c.db.WithContext(ctx)
	var row PromptTemplate
	err := tx.Where("project_key = ? AND run_kind = ? AND agent_role = ?", projectKey, runKind, agentRole).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = PromptTemplate{
			ProjectKey: projectKey,
			RunKind:    runKind,
			AgentRole:  agentRole,
			Content:    content,
		}
		if createErr := tx.Create(&row).Error; createErr != nil {
			return nil, xerr.Infra.Wrap(createErr, "create prompt template")
		}
		return &row, nil
	}
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "query prompt template")
	}

	row.Content = content
	if saveErr := tx.Save(&row).Error; saveErr != nil {
		return nil, xerr.Infra.Wrap(saveErr, "update prompt template")
	}
	return &row, nil
}

// DeletePromptTemplate 是 *Client 的方法实现。
func (c *Client) DeletePromptTemplate(
	ctx context.Context,
	projectKey string,
	runKind string,
	agentRole string,
) error {
	if c == nil || c.db == nil {
		return xerr.Infra.New("db is not initialized")
	}

	err := c.db.WithContext(ctx).
		Where("project_key = ? AND run_kind = ? AND agent_role = ?", projectKey, runKind, agentRole).
		Delete(&PromptTemplate{}).Error
	if err != nil {
		return xerr.Infra.Wrap(err, "delete prompt template")
	}
	return nil
}
