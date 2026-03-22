package project

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"time"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/prompts"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
	"github.com/haowen-xu/agent-coder/internal/infra/repo/gitlab"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// UpsertInput 表示数据结构定义。
type UpsertInput struct {
	ProjectKey        string  // ProjectKey 字段说明。
	ProjectSlug       string  // ProjectSlug 字段说明。
	Name              string  // Name 字段说明。
	Provider          string  // Provider 字段说明。
	ProviderURL       string  // ProviderURL 字段说明。
	RepoURL           string  // RepoURL 字段说明。
	DefaultBranch     string  // DefaultBranch 字段说明。
	IssueProjectID    *string // IssueProjectID 字段说明。
	CredentialRef     string  // CredentialRef 字段说明。
	ProjectToken      *string // ProjectToken 字段说明。
	SandboxPlanReview bool    // SandboxPlanReview 字段说明。
	PollIntervalSec   int     // PollIntervalSec 字段说明。
	Enabled           bool    // Enabled 字段说明。
	LabelAgentReady   string  // LabelAgentReady 字段说明。
	LabelInProgress   string  // LabelInProgress 字段说明。
	LabelHumanReview  string  // LabelHumanReview 字段说明。
	LabelRework       string  // LabelRework 字段说明。
	LabelVerified     string  // LabelVerified 字段说明。
	LabelMerged       string  // LabelMerged 字段说明。
}

// Service 表示数据结构定义。
type Service struct {
	cfg *appcfg.Config       // cfg 字段说明。
	db  *db.Client           // db 字段说明。
	ps  *promptstore.Service // ps 字段说明。
}

// New 执行相关逻辑。
func New(cfg *appcfg.Config, dbClient *db.Client, ps *promptstore.Service) *Service {
	return &Service{
		cfg: cfg,
		db:  dbClient,
		ps:  ps,
	}
}

// NormalizeUpsertInput 执行相关逻辑。
func NormalizeUpsertInput(in *UpsertInput) {
	in.ProjectKey = strings.TrimSpace(in.ProjectKey)
	in.ProjectSlug = strings.TrimSpace(in.ProjectSlug)
	in.Name = strings.TrimSpace(in.Name)
	in.Provider = strings.TrimSpace(in.Provider)
	in.ProviderURL = strings.TrimSpace(in.ProviderURL)
	in.RepoURL = strings.TrimSpace(in.RepoURL)
	in.DefaultBranch = strings.TrimSpace(in.DefaultBranch)
	in.CredentialRef = strings.TrimSpace(in.CredentialRef)
	if in.ProjectToken != nil {
		v := strings.TrimSpace(*in.ProjectToken)
		if v == "" {
			in.ProjectToken = nil
		} else {
			in.ProjectToken = &v
		}
	}
	if in.Provider == "" {
		in.Provider = db.ProviderGitLab
	}
	if in.DefaultBranch == "" {
		in.DefaultBranch = "main"
	}
	if in.PollIntervalSec <= 0 {
		in.PollIntervalSec = 60
	}
	if in.LabelAgentReady == "" {
		in.LabelAgentReady = "Agent Ready"
	}
	if in.LabelInProgress == "" {
		in.LabelInProgress = "In Progress"
	}
	if in.LabelHumanReview == "" {
		in.LabelHumanReview = "Human Review"
	}
	if in.LabelRework == "" {
		in.LabelRework = "Rework"
	}
	if in.LabelVerified == "" {
		in.LabelVerified = "Verified"
	}
	if in.LabelMerged == "" {
		in.LabelMerged = "Merged"
	}
}

// ValidateUpsertInput 执行相关逻辑。
func ValidateUpsertInput(in UpsertInput) error {
	if in.ProjectKey == "" {
		return xerr.Config.New("project_key is required")
	}
	if in.ProjectSlug == "" {
		return xerr.Config.New("project_slug is required")
	}
	if in.Name == "" {
		return xerr.Config.New("name is required")
	}
	if in.ProviderURL == "" {
		return xerr.Config.New("provider_url is required")
	}
	if in.RepoURL == "" {
		return xerr.Config.New("repo_url is required")
	}
	if in.CredentialRef == "" && (in.ProjectToken == nil || strings.TrimSpace(*in.ProjectToken) == "") {
		return xerr.Config.New("credential_ref or project_token is required")
	}
	return nil
}

// ListProjects 是 *Service 的方法实现。
func (s *Service) ListProjects(ctx context.Context) ([]db.Project, error) {
	return s.db.ListProjects(ctx)
}

// CreateProject 是 *Service 的方法实现。
func (s *Service) CreateProject(ctx context.Context, createdBy uint, in UpsertInput) (*db.Project, error) {
	NormalizeUpsertInput(&in)
	if err := ValidateUpsertInput(in); err != nil {
		return nil, err
	}
	existed, err := s.db.GetProjectByKey(ctx, in.ProjectKey)
	if err != nil {
		return nil, err
	}
	if existed != nil {
		return nil, xerr.Config.New("project_key already exists")
	}
	if err := s.fillIssueProjectIDOnCreate(ctx, &in); err != nil {
		return nil, err
	}
	row := &db.Project{
		ProjectKey:        in.ProjectKey,
		ProjectSlug:       in.ProjectSlug,
		Name:              in.Name,
		Provider:          in.Provider,
		ProviderURL:       in.ProviderURL,
		RepoURL:           in.RepoURL,
		DefaultBranch:     in.DefaultBranch,
		IssueProjectID:    in.IssueProjectID,
		CredentialRef:     in.CredentialRef,
		ProjectToken:      in.ProjectToken,
		SandboxPlanReview: in.SandboxPlanReview,
		PollIntervalSec:   in.PollIntervalSec,
		Enabled:           in.Enabled,
		LabelAgentReady:   in.LabelAgentReady,
		LabelInProgress:   in.LabelInProgress,
		LabelHumanReview:  in.LabelHumanReview,
		LabelRework:       in.LabelRework,
		LabelVerified:     in.LabelVerified,
		LabelMerged:       in.LabelMerged,
		CreatedBy:         createdBy,
	}
	if err := s.db.CreateProject(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateProject 是 *Service 的方法实现。
func (s *Service) UpdateProject(ctx context.Context, projectKey string, in UpsertInput) (*db.Project, error) {
	projectKey = strings.TrimSpace(projectKey)
	NormalizeUpsertInput(&in)
	if in.ProjectKey == "" {
		in.ProjectKey = projectKey
	}
	if err := ValidateUpsertInput(in); err != nil {
		return nil, err
	}

	row, err := s.db.GetProjectByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, xerr.Config.New("project not found")
	}
	row.ProjectKey = in.ProjectKey
	row.ProjectSlug = in.ProjectSlug
	row.Name = in.Name
	row.Provider = in.Provider
	row.ProviderURL = in.ProviderURL
	row.RepoURL = in.RepoURL
	row.DefaultBranch = in.DefaultBranch
	row.IssueProjectID = in.IssueProjectID
	row.CredentialRef = in.CredentialRef
	row.ProjectToken = in.ProjectToken
	row.SandboxPlanReview = in.SandboxPlanReview
	row.PollIntervalSec = in.PollIntervalSec
	row.Enabled = in.Enabled
	row.LabelAgentReady = in.LabelAgentReady
	row.LabelInProgress = in.LabelInProgress
	row.LabelHumanReview = in.LabelHumanReview
	row.LabelRework = in.LabelRework
	row.LabelVerified = in.LabelVerified
	row.LabelMerged = in.LabelMerged
	if err := s.db.SaveProject(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

// ListProjectIssues 是 *Service 的方法实现。
func (s *Service) ListProjectIssues(ctx context.Context, projectKey string, limit int) ([]db.Issue, error) {
	project, err := s.db.GetProjectByKey(ctx, strings.TrimSpace(projectKey))
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, xerr.Config.New("project not found")
	}
	return s.db.ListIssuesByProject(ctx, project.ID, limit)
}

// ResetProjectSyncCursor 是 *Service 的方法实现。
func (s *Service) ResetProjectSyncCursor(ctx context.Context, projectKey string) (*db.Project, error) {
	row, err := s.db.ResetProjectSyncCursorByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, xerr.Config.New("project not found")
	}
	return row, nil
}

// ListDefaultPrompts 是 *Service 的方法实现。
func (s *Service) ListDefaultPrompts() ([]prompts.Template, error) {
	return s.ps.ListDefaults()
}

// ListProjectPrompts 是 *Service 的方法实现。
func (s *Service) ListProjectPrompts(ctx context.Context, projectKey string) ([]prompts.Template, error) {
	return s.ps.ListEffectiveByProject(ctx, projectKey)
}

// UpsertProjectPrompt 是 *Service 的方法实现。
func (s *Service) UpsertProjectPrompt(ctx context.Context, projectKey, runKind, role, content string) (*prompts.Template, error) {
	return s.ps.UpsertProjectOverride(ctx, projectKey, runKind, role, content)
}

// DeleteProjectPrompt 是 *Service 的方法实现。
func (s *Service) DeleteProjectPrompt(ctx context.Context, projectKey, runKind, role string) error {
	return s.ps.DeleteProjectOverride(ctx, projectKey, runKind, role)
}

// fillIssueProjectIDOnCreate 在创建项目时补全远端 project_id，无法补全时快速失败。
func (s *Service) fillIssueProjectIDOnCreate(ctx context.Context, in *UpsertInput) error {
	if in == nil {
		return xerr.Config.New("project input is required")
	}
	provider := strings.ToLower(strings.TrimSpace(in.Provider))
	switch provider {
	case "", db.ProviderGitLab:
	default:
		return xerr.Config.New("unsupported provider: %s", in.Provider)
	}

	if in.ProjectToken == nil || strings.TrimSpace(*in.ProjectToken) == "" {
		return xerr.Config.New("project_token is required to validate repo_url and auto-fill issue_project_id")
	}
	token := strings.TrimSpace(*in.ProjectToken)

	timeout := 30 * time.Second
	if s != nil && s.cfg != nil {
		timeout = time.Duration(s.cfg.RepoHTTPTimeoutSec()) * time.Second
	}
	validator := gitlab.NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), timeout, nil)
	resolved, err := validator.ValidateURL(ctx, repocommon.ValidateURLArgs{
		ProviderURL:  in.ProviderURL,
		RepoURL:      in.RepoURL,
		ProjectToken: token,
	})
	if err != nil {
		return xerr.Config.Wrap(err, "validate repo_url")
	}
	if resolved == nil || strings.TrimSpace(resolved.ProjectID) == "" {
		return xerr.Config.New("failed to resolve issue_project_id from repo_url")
	}

	remoteID := strings.TrimSpace(resolved.ProjectID)
	if in.IssueProjectID != nil && strings.TrimSpace(*in.IssueProjectID) != "" {
		inputID := strings.TrimSpace(*in.IssueProjectID)
		if inputID != remoteID {
			return xerr.Config.New("issue_project_id mismatch with remote project_id: input=%s remote=%s", inputID, remoteID)
		}
	}
	in.IssueProjectID = &remoteID
	return nil
}
