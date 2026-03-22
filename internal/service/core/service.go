package core

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/haowen-xu/agent-coder/internal/auth"
	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/prompts"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
	"github.com/haowen-xu/agent-coder/internal/infra/repo/gitlab"
	"github.com/haowen-xu/agent-coder/internal/utils"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

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

// AuthUser 表示数据结构定义。
type AuthUser struct {
	ID       uint   // ID 字段说明。
	Username string // Username 字段说明。
	IsAdmin  bool   // IsAdmin 字段说明。
	Enabled  bool   // Enabled 字段说明。
}

// Login 是 *Service 的方法实现。
func (s *Service) Login(ctx context.Context, username string, password string) (string, time.Time, *AuthUser, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return "", time.Time{}, nil, xerr.Config.New("username/password are required")
	}

	user, err := s.db.GetUserByUsername(ctx, username)
	if err != nil {
		return "", time.Time{}, nil, err
	}
	if user == nil || !user.Enabled {
		return "", time.Time{}, nil, xerr.Config.New("invalid username or password")
	}
	if !auth.VerifyPassword(password, user.PasswordHash) {
		return "", time.Time{}, nil, xerr.Config.New("invalid username or password")
	}

	token, err := auth.NewToken()
	if err != nil {
		return "", time.Time{}, nil, xerr.Infra.Wrap(err, "new session token")
	}
	now := utils.NowUTC()
	expiredAt := now.Add(s.cfg.Auth.SessionTTLDuration())
	sess := &db.UserSession{
		UserID:    user.ID,
		Token:     token,
		ExpiredAt: expiredAt,
	}
	if err := s.db.CreateSession(ctx, sess); err != nil {
		return "", time.Time{}, nil, err
	}
	user.LastLoginAt = &now
	if err := s.db.SaveUser(ctx, user); err != nil {
		return "", time.Time{}, nil, err
	}
	return token, expiredAt, &AuthUser{
		ID:       user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Enabled:  user.Enabled,
	}, nil
}

// AuthByToken 是 *Service 的方法实现。
func (s *Service) AuthByToken(ctx context.Context, token string) (*AuthUser, error) {
	_, user, err := s.db.GetSessionWithUser(ctx, token)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.Enabled {
		return nil, nil
	}
	return &AuthUser{
		ID:       user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Enabled:  user.Enabled,
	}, nil
}

// ListUsers 是 *Service 的方法实现。
func (s *Service) ListUsers(ctx context.Context) ([]db.User, error) {
	return s.db.ListUsers(ctx)
}

// CreateUser 是 *Service 的方法实现。
func (s *Service) CreateUser(ctx context.Context, username string, password string, isAdmin bool, enabled bool) (*db.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, xerr.Config.New("username/password are required")
	}
	existed, err := s.db.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existed != nil {
		return nil, xerr.Config.New("username already exists")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "hash password")
	}
	row := &db.User{
		Username:     username,
		PasswordHash: hash,
		IsAdmin:      isAdmin,
		Enabled:      enabled,
	}
	if err := s.db.CreateUser(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateUser 是 *Service 的方法实现。
func (s *Service) UpdateUser(ctx context.Context, userID uint, password *string, isAdmin *bool, enabled *bool) (*db.User, error) {
	row, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, xerr.Config.New("user not found")
	}
	if password != nil {
		hash, err := auth.HashPassword(*password)
		if err != nil {
			return nil, xerr.Infra.Wrap(err, "hash password")
		}
		row.PasswordHash = hash
	}
	if isAdmin != nil {
		row.IsAdmin = *isAdmin
	}
	if enabled != nil {
		row.Enabled = *enabled
	}
	if err := s.db.SaveUser(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

// ListProjects 是 *Service 的方法实现。
func (s *Service) ListProjects(ctx context.Context) ([]db.Project, error) {
	return s.db.ListProjects(ctx)
}

// ProjectUpsertInput 表示数据结构定义。
type ProjectUpsertInput struct {
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

// NormalizeProjectInput 执行相关逻辑。
func NormalizeProjectInput(in *ProjectUpsertInput) {
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

// ValidateProjectInput 执行相关逻辑。
func ValidateProjectInput(in ProjectUpsertInput) error {
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

// CreateProject 是 *Service 的方法实现。
func (s *Service) CreateProject(ctx context.Context, createdBy uint, in ProjectUpsertInput) (*db.Project, error) {
	NormalizeProjectInput(&in)
	if err := ValidateProjectInput(in); err != nil {
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

// fillIssueProjectIDOnCreate 在创建项目时补全远端 project_id，无法补全时快速失败。
func (s *Service) fillIssueProjectIDOnCreate(ctx context.Context, in *ProjectUpsertInput) error {
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

// UpdateProject 是 *Service 的方法实现。
func (s *Service) UpdateProject(ctx context.Context, projectKey string, in ProjectUpsertInput) (*db.Project, error) {
	projectKey = strings.TrimSpace(projectKey)
	NormalizeProjectInput(&in)
	if in.ProjectKey == "" {
		in.ProjectKey = projectKey
	}
	if err := ValidateProjectInput(in); err != nil {
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

// OpsMetrics 表示数据结构定义。
type OpsMetrics struct {
	Timestamp time.Time         `json:"timestamp"` // Timestamp 字段说明。
	Projects  OpsProjectsMetric `json:"projects"`  // Projects 字段说明。
	Issues    OpsIssueMetric    `json:"issues"`    // Issues 字段说明。
	Runs      OpsRunMetric      `json:"runs"`      // Runs 字段说明。
}

// OpsProjectsMetric 表示数据结构定义。
type OpsProjectsMetric struct {
	Total   int64 `json:"total"`   // Total 字段说明。
	Enabled int64 `json:"enabled"` // Enabled 字段说明。
}

// OpsIssueMetric 表示数据结构定义。
type OpsIssueMetric struct {
	Total       int64            `json:"total"`        // Total 字段说明。
	ByLifecycle map[string]int64 `json:"by_lifecycle"` // ByLifecycle 字段说明。
}

// OpsRunMetric 表示数据结构定义。
type OpsRunMetric struct {
	Total    int64            `json:"total"`     // Total 字段说明。
	ByStatus map[string]int64 `json:"by_status"` // ByStatus 字段说明。
	ByKind   map[string]int64 `json:"by_kind"`   // ByKind 字段说明。
}

// GetOpsMetrics 是 *Service 的方法实现。
func (s *Service) GetOpsMetrics(ctx context.Context) (*OpsMetrics, error) {
	projectTotal, projectEnabled, err := s.db.CountProjects(ctx)
	if err != nil {
		return nil, err
	}
	issueTotal, err := s.db.CountIssues(ctx)
	if err != nil {
		return nil, err
	}
	issueByLifecycle, err := s.db.CountIssuesByLifecycle(ctx)
	if err != nil {
		return nil, err
	}
	runTotal, err := s.db.CountRuns(ctx)
	if err != nil {
		return nil, err
	}
	runByStatus, err := s.db.CountRunsByStatus(ctx)
	if err != nil {
		return nil, err
	}
	runByKind, err := s.db.CountRunsByKind(ctx)
	if err != nil {
		return nil, err
	}
	return &OpsMetrics{
		Timestamp: utils.NowUTC(),
		Projects: OpsProjectsMetric{
			Total:   projectTotal,
			Enabled: projectEnabled,
		},
		Issues: OpsIssueMetric{
			Total:       issueTotal,
			ByLifecycle: issueByLifecycle,
		},
		Runs: OpsRunMetric{
			Total:    runTotal,
			ByStatus: runByStatus,
			ByKind:   runByKind,
		},
	}, nil
}

// GuardAdmin 是 *Service 的方法实现。
func (s *Service) GuardAdmin(user *AuthUser) error {
	if user == nil || !user.IsAdmin {
		return xerr.Config.New("admin required")
	}
	return nil
}

// stringPtr 执行相关逻辑。
func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := strings.TrimSpace(v)
	return &s
}

// Describe 是 *Service 的方法实现。
func (s *Service) Describe() string {
	return fmt.Sprintf("core service env=%s", s.cfg.App.Env)
}
