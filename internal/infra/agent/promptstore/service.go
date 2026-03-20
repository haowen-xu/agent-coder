package promptstore

import (
	"context"
	"strings"

	"github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/prompts"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// Service 表示数据结构定义。
type Service struct {
	db *db.Client // db 字段说明。
}

// NewService 执行相关逻辑。
func NewService(dbClient *db.Client) *Service {
	return &Service{db: dbClient}
}

// ListDefaults 是 *Service 的方法实现。
func (s *Service) ListDefaults() ([]prompts.Template, error) {
	return prompts.ListDefaultTemplates()
}

// ListEffectiveByProject 是 *Service 的方法实现。
func (s *Service) ListEffectiveByProject(ctx context.Context, projectKey string) ([]prompts.Template, error) {
	projectKey = strings.TrimSpace(projectKey)
	if projectKey == "" {
		return nil, xerr.Config.New("project_key is required")
	}

	defaults, err := prompts.ListDefaultTemplates()
	if err != nil {
		return nil, err
	}
	if s.db == nil || !s.db.Enabled() {
		return defaults, nil
	}

	rows, err := s.db.ListPromptTemplatesByProject(ctx, projectKey)
	if err != nil {
		return nil, err
	}

	overrideMap := make(map[string]db.PromptTemplate, len(rows))
	for _, row := range rows {
		overrideMap[row.RunKind+":"+row.AgentRole] = row
	}

	out := make([]prompts.Template, 0, len(defaults))
	for _, d := range defaults {
		key := d.RunKind + ":" + d.AgentRole
		if row, ok := overrideMap[key]; ok {
			out = append(out, prompts.Template{
				ProjectKey: row.ProjectKey,
				RunKind:    row.RunKind,
				AgentRole:  row.AgentRole,
				Source:     "project_override",
				Content:    row.Content,
			})
			continue
		}
		d.ProjectKey = projectKey
		out = append(out, d)
	}

	return out, nil
}

// UpsertProjectOverride 是 *Service 的方法实现。
func (s *Service) UpsertProjectOverride(
	ctx context.Context,
	projectKey string,
	runKind string,
	agentRole string,
	content string,
) (*prompts.Template, error) {
	projectKey = strings.TrimSpace(projectKey)
	runKind = strings.TrimSpace(runKind)
	agentRole = strings.TrimSpace(agentRole)
	content = strings.TrimSpace(content)

	if projectKey == "" {
		return nil, xerr.Config.New("project_key is required")
	}
	if content == "" {
		return nil, xerr.Config.New("content is required")
	}
	if err := prompts.ValidateKey(runKind, agentRole); err != nil {
		return nil, err
	}
	if s.db == nil || !s.db.Enabled() {
		return nil, xerr.Config.New("db must be enabled for prompt override")
	}

	row, err := s.db.UpsertPromptTemplate(ctx, projectKey, runKind, agentRole, content)
	if err != nil {
		return nil, err
	}
	return &prompts.Template{
		ProjectKey: row.ProjectKey,
		RunKind:    row.RunKind,
		AgentRole:  row.AgentRole,
		Source:     "project_override",
		Content:    row.Content,
	}, nil
}

// DeleteProjectOverride 是 *Service 的方法实现。
func (s *Service) DeleteProjectOverride(ctx context.Context, projectKey string, runKind string, agentRole string) error {
	projectKey = strings.TrimSpace(projectKey)
	runKind = strings.TrimSpace(runKind)
	agentRole = strings.TrimSpace(agentRole)

	if projectKey == "" {
		return xerr.Config.New("project_key is required")
	}
	if err := prompts.ValidateKey(runKind, agentRole); err != nil {
		return err
	}
	if s.db == nil || !s.db.Enabled() {
		return xerr.Config.New("db must be enabled for prompt override")
	}

	return s.db.DeletePromptTemplate(ctx, projectKey, runKind, agentRole)
}
