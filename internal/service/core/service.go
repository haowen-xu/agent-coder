package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/haowen-xu/agent-coder/internal/auth"
	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/prompts"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type Service struct {
	cfg *appcfg.Config
	db  *db.Client
	ps  *promptstore.Service
}

func New(cfg *appcfg.Config, dbClient *db.Client, ps *promptstore.Service) *Service {
	return &Service{
		cfg: cfg,
		db:  dbClient,
		ps:  ps,
	}
}

type AuthUser struct {
	ID       uint
	Username string
	IsAdmin  bool
	Enabled  bool
}

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
	now := time.Now()
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

func (s *Service) ListUsers(ctx context.Context) ([]db.User, error) {
	return s.db.ListUsers(ctx)
}

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

func (s *Service) ListProjects(ctx context.Context) ([]db.Project, error) {
	return s.db.ListProjects(ctx)
}

type ProjectUpsertInput struct {
	ProjectKey       string
	ProjectSlug      string
	Name             string
	Provider         string
	ProviderURL      string
	RepoURL          string
	DefaultBranch    string
	IssueProjectID   *string
	CredentialRef    string
	ProjectToken     *string
	PollIntervalSec  int
	Enabled          bool
	LabelAgentReady  string
	LabelInProgress  string
	LabelHumanReview string
	LabelRework      string
	LabelVerified    string
	LabelMerged      string
}

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
	row := &db.Project{
		ProjectKey:       in.ProjectKey,
		ProjectSlug:      in.ProjectSlug,
		Name:             in.Name,
		Provider:         in.Provider,
		ProviderURL:      in.ProviderURL,
		RepoURL:          in.RepoURL,
		DefaultBranch:    in.DefaultBranch,
		IssueProjectID:   in.IssueProjectID,
		CredentialRef:    in.CredentialRef,
		ProjectToken:     in.ProjectToken,
		PollIntervalSec:  in.PollIntervalSec,
		Enabled:          in.Enabled,
		LabelAgentReady:  in.LabelAgentReady,
		LabelInProgress:  in.LabelInProgress,
		LabelHumanReview: in.LabelHumanReview,
		LabelRework:      in.LabelRework,
		LabelVerified:    in.LabelVerified,
		LabelMerged:      in.LabelMerged,
		CreatedBy:        createdBy,
	}
	if err := s.db.CreateProject(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

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

func (s *Service) ListDefaultPrompts() ([]prompts.Template, error) {
	return s.ps.ListDefaults()
}

func (s *Service) ListProjectPrompts(ctx context.Context, projectKey string) ([]prompts.Template, error) {
	return s.ps.ListEffectiveByProject(ctx, projectKey)
}

func (s *Service) UpsertProjectPrompt(ctx context.Context, projectKey, runKind, role, content string) (*prompts.Template, error) {
	return s.ps.UpsertProjectOverride(ctx, projectKey, runKind, role, content)
}

func (s *Service) DeleteProjectPrompt(ctx context.Context, projectKey, runKind, role string) error {
	return s.ps.DeleteProjectOverride(ctx, projectKey, runKind, role)
}

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
	now := time.Now()
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

type OpsMetrics struct {
	Timestamp time.Time         `json:"timestamp"`
	Projects  OpsProjectsMetric `json:"projects"`
	Issues    OpsIssueMetric    `json:"issues"`
	Runs      OpsRunMetric      `json:"runs"`
}

type OpsProjectsMetric struct {
	Total   int64 `json:"total"`
	Enabled int64 `json:"enabled"`
}

type OpsIssueMetric struct {
	Total       int64            `json:"total"`
	ByLifecycle map[string]int64 `json:"by_lifecycle"`
}

type OpsRunMetric struct {
	Total    int64            `json:"total"`
	ByStatus map[string]int64 `json:"by_status"`
	ByKind   map[string]int64 `json:"by_kind"`
}

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
		Timestamp: time.Now(),
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

func (s *Service) GuardAdmin(user *AuthUser) error {
	if user == nil || !user.IsAdmin {
		return xerr.Config.New("admin required")
	}
	return nil
}

func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := strings.TrimSpace(v)
	return &s
}

func (s *Service) Describe() string {
	return fmt.Sprintf("core service env=%s", s.cfg.App.Env)
}
