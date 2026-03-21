package core

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/haowen-xu/agent-coder/internal/auth"
	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
)

func newCoreServiceForTest(t *testing.T) (*Service, *db.Client) {
	t.Helper()
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbClient, err := db.New(ctx, appcfg.DBConfig{
		Enabled:         true,
		Driver:          "sqlite",
		SQLitePath:      filepath.Join(t.TempDir(), "core.db"),
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: "1m",
		AutoMigrate:     true,
	}, log)
	if err != nil {
		t.Fatalf("init db failed: %v", err)
	}
	t.Cleanup(func() { _ = dbClient.Close() })

	cfg := &appcfg.Config{
		App: appcfg.AppConfig{Env: "test"},
		Auth: appcfg.AuthConfig{
			SessionTTL: "1h",
		},
	}
	ps := promptstore.NewService(dbClient)
	return New(cfg, dbClient, ps), dbClient
}

func createProjectForCoreTest(t *testing.T, c *db.Client, key string) *db.Project {
	t.Helper()
	row := &db.Project{
		ProjectKey:       key,
		ProjectSlug:      "group/" + key,
		Name:             key,
		Provider:         db.ProviderGitLab,
		ProviderURL:      "https://gitlab.example.com/api/v4",
		RepoURL:          "https://gitlab.example.com/group/" + key,
		DefaultBranch:    "main",
		CredentialRef:    "credential-" + key,
		PollIntervalSec:  60,
		Enabled:          true,
		LabelAgentReady:  "Agent Ready",
		LabelInProgress:  "In Progress",
		LabelHumanReview: "Human Review",
		LabelRework:      "Rework",
		LabelVerified:    "Verified",
		LabelMerged:      "Merged",
		CreatedBy:        1,
	}
	if err := c.CreateProject(context.Background(), row); err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	return row
}

// TestServiceAuthAndUserFlow 用于单元测试。
func TestServiceAuthAndUserFlow(t *testing.T) {
	ctx := context.Background()
	svc, dbClient := newCoreServiceForTest(t)

	if _, _, _, err := svc.Login(ctx, "", "x"); err == nil {
		t.Fatalf("expected login validation error")
	}
	if _, _, _, err := svc.Login(ctx, "u1", "x"); err == nil {
		t.Fatalf("expected login failure for missing user")
	}

	u, err := svc.CreateUser(ctx, "u1", "p1", true, true)
	if err != nil || u == nil {
		t.Fatalf("CreateUser failed: user=%#v err=%v", u, err)
	}
	if _, err := svc.CreateUser(ctx, "u1", "p1", true, true); err == nil {
		t.Fatalf("expected duplicate username error")
	}

	token, expiredAt, authUser, err := svc.Login(ctx, "u1", "p1")
	if err != nil {
		t.Fatalf("login should succeed: %v", err)
	}
	if token == "" || !expiredAt.After(time.Now()) || authUser.Username != "u1" {
		t.Fatalf("unexpected login payload: token=%q expiredAt=%s user=%#v", token, expiredAt, authUser)
	}

	byToken, err := svc.AuthByToken(ctx, token)
	if err != nil || byToken == nil || byToken.Username != "u1" {
		t.Fatalf("AuthByToken mismatch: user=%#v err=%v", byToken, err)
	}

	if _, err := svc.UpdateUser(ctx, 99999, nil, nil, nil); err == nil {
		t.Fatalf("expected update non-existing user error")
	}
	enabled := false
	isAdmin := false
	newPassword := "p2"
	updatedUser, err := svc.UpdateUser(ctx, u.ID, &newPassword, &isAdmin, &enabled)
	if err != nil || updatedUser.Enabled || updatedUser.IsAdmin {
		t.Fatalf("UpdateUser mismatch: user=%#v err=%v", updatedUser, err)
	}
	if _, _, _, err := svc.Login(ctx, "u1", "p2"); err == nil {
		t.Fatalf("disabled user should not be able to login")
	}

	rows, err := svc.ListUsers(ctx)
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListUsers mismatch: len=%d err=%v", len(rows), err)
	}

	// validate AuthByToken with expired session
	hash, _ := auth.HashPassword("pw")
	u2 := &db.User{Username: "u2", PasswordHash: hash, Enabled: true, IsAdmin: true}
	if err := dbClient.CreateUser(ctx, u2); err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}
	if err := dbClient.CreateSession(ctx, &db.UserSession{
		UserID:    u2.ID,
		Token:     "expired-token",
		ExpiredAt: time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("create expired session failed: %v", err)
	}
	byToken, err = svc.AuthByToken(ctx, "expired-token")
	if err != nil || byToken != nil {
		t.Fatalf("expired token should not authenticate: user=%#v err=%v", byToken, err)
	}
}

// TestServiceProjectPromptAndOpsFlow 用于单元测试。
func TestServiceProjectPromptAndOpsFlow(t *testing.T) {
	ctx := context.Background()
	svc, dbClient := newCoreServiceForTest(t)

	gitlabServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.EscapedPath() == "/api/v4/projects/group%2Fp1" {
			if r.Header.Get("PRIVATE-TOKEN") != "token-x" {
				t.Fatalf("missing PRIVATE-TOKEN header")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":123,"path_with_namespace":"group/p1"}`))
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
	}))
	defer gitlabServer.Close()

	invalid := ProjectUpsertInput{}
	NormalizeProjectInput(&invalid)
	if err := ValidateProjectInput(invalid); err == nil {
		t.Fatalf("expected invalid project input")
	}

	if _, err := svc.CreateProject(ctx, 1, ProjectUpsertInput{ProjectKey: "p1"}); err == nil {
		t.Fatalf("expected create project validation error")
	}

	in := ProjectUpsertInput{
		ProjectKey:      "p1",
		ProjectSlug:     "group/p1",
		Name:            "Project One",
		ProviderURL:     gitlabServer.URL + "/api/v4",
		RepoURL:         "https://gitlab.example.com/group/p1",
		ProjectToken:    ptrString("  token-x "),
		Enabled:         true,
		PollIntervalSec: 0,
	}
	project, err := svc.CreateProject(ctx, 1, in)
	if err != nil || project == nil {
		t.Fatalf("CreateProject failed: row=%#v err=%v", project, err)
	}
	if project.DefaultBranch != "main" || project.Provider != db.ProviderGitLab || project.ProjectToken == nil || *project.ProjectToken != "token-x" {
		t.Fatalf("project normalization mismatch: %#v", project)
	}
	if project.IssueProjectID == nil || *project.IssueProjectID != "123" {
		t.Fatalf("issue_project_id should be auto-filled: %#v", project)
	}
	if _, err := svc.CreateProject(ctx, 1, in); err == nil {
		t.Fatalf("expected duplicate project create error")
	}

	if _, err := svc.CreateProject(ctx, 1, ProjectUpsertInput{
		ProjectKey:      "p-no-token",
		ProjectSlug:     "group/p-no-token",
		Name:            "Project No Token",
		ProviderURL:     gitlabServer.URL + "/api/v4",
		RepoURL:         "https://gitlab.example.com/group/p-no-token",
		CredentialRef:   "secret-ref",
		Enabled:         true,
		PollIntervalSec: 30,
	}); err == nil || !strings.Contains(err.Error(), "project_token is required") {
		t.Fatalf("expected fast-fail when project_token missing, err=%v", err)
	}

	updated, err := svc.UpdateProject(ctx, "p1", ProjectUpsertInput{
		ProjectSlug:   "group/p1",
		ProjectKey:    "p1",
		Name:          "Project One Updated",
		ProviderURL:   gitlabServer.URL + "/api/v4",
		RepoURL:       "https://gitlab.example.com/group/p1",
		CredentialRef: "secret-ref",
		Enabled:       true,
	})
	if err != nil || updated.Name != "Project One Updated" {
		t.Fatalf("UpdateProject failed: row=%#v err=%v", updated, err)
	}

	projects, err := svc.ListProjects(ctx)
	if err != nil || len(projects) != 1 {
		t.Fatalf("ListProjects mismatch: len=%d err=%v", len(projects), err)
	}

	if _, err := svc.ListProjectIssues(ctx, "missing", 10); err == nil {
		t.Fatalf("expected project not found for ListProjectIssues")
	}
	issue := &db.Issue{
		ProjectID:       project.ID,
		IssueIID:        101,
		Title:           "issue-101",
		State:           "opened",
		LabelsJSON:      "[]",
		RegisteredAt:    time.Now(),
		LifecycleStatus: db.IssueLifecycleRegistered,
		IssueDir:        "/tmp/issue-101",
		LastSyncedAt:    time.Now(),
	}
	if err := dbClient.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("create issue failed: %v", err)
	}

	projectIssues, err := svc.ListProjectIssues(ctx, "p1", 10)
	if err != nil || len(projectIssues) != 1 {
		t.Fatalf("ListProjectIssues mismatch: len=%d err=%v", len(projectIssues), err)
	}

	defaultPrompts, err := svc.ListDefaultPrompts()
	if err != nil || len(defaultPrompts) == 0 {
		t.Fatalf("ListDefaultPrompts mismatch: len=%d err=%v", len(defaultPrompts), err)
	}
	projectPrompts, err := svc.ListProjectPrompts(ctx, "p1")
	if err != nil || len(projectPrompts) == 0 {
		t.Fatalf("ListProjectPrompts mismatch: len=%d err=%v", len(projectPrompts), err)
	}
	override, err := svc.UpsertProjectPrompt(ctx, "p1", db.RunKindDev, db.AgentRoleDev, "override content")
	if err != nil || override == nil || override.Source != "project_override" {
		t.Fatalf("UpsertProjectPrompt mismatch: row=%#v err=%v", override, err)
	}
	if err := svc.DeleteProjectPrompt(ctx, "p1", db.RunKindDev, db.AgentRoleDev); err != nil {
		t.Fatalf("DeleteProjectPrompt failed: %v", err)
	}

	if _, err := svc.ListIssueRuns(ctx, 99999, 10); err == nil {
		t.Fatalf("expected missing issue error")
	}
	if _, err := svc.ListRunLogs(ctx, 99999, 10); err == nil {
		t.Fatalf("expected missing run error")
	}
	run := &db.IssueRun{
		IssueID:          issue.ID,
		RunNo:            1,
		RunKind:          db.RunKindDev,
		TriggerType:      db.TriggerScheduler,
		Status:           db.RunStatusQueued,
		AgentRole:        db.AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      5,
		QueuedAt:         time.Now(),
		BranchName:       "agent/issue-101",
		GitTreePath:      "/tmp/git-tree",
		AgentRunDir:      "/tmp/agent-run",
		MaxConflictRetry: 3,
	}
	if err := dbClient.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run failed: %v", err)
	}
	if err := dbClient.AppendRunLog(ctx, &db.RunLog{
		RunID:     run.ID,
		Seq:       1,
		Level:     "INFO",
		Stage:     "agent",
		EventType: "dev",
		Message:   "log-1",
	}); err != nil {
		t.Fatalf("append run log failed: %v", err)
	}
	issueRuns, err := svc.ListIssueRuns(ctx, issue.ID, 10)
	if err != nil || len(issueRuns) != 1 {
		t.Fatalf("ListIssueRuns mismatch: len=%d err=%v", len(issueRuns), err)
	}
	runLogs, err := svc.ListRunLogs(ctx, run.ID, 10)
	if err != nil || len(runLogs) != 1 {
		t.Fatalf("ListRunLogs mismatch: len=%d err=%v", len(runLogs), err)
	}

	if _, err := svc.RetryIssue(ctx, 999999); err == nil {
		t.Fatalf("expected retry missing issue error")
	}
	issue.LifecycleStatus = db.IssueLifecycleClosed
	if err := dbClient.SaveIssue(ctx, issue); err != nil {
		t.Fatalf("save closed issue failed: %v", err)
	}
	if _, err := svc.RetryIssue(ctx, issue.ID); err == nil {
		t.Fatalf("expected closed issue retry error")
	}
	issue.LifecycleStatus = db.IssueLifecycleRegistered
	issue.CurrentRunID = &run.ID
	if err := dbClient.SaveIssue(ctx, issue); err != nil {
		t.Fatalf("save issue current run failed: %v", err)
	}
	if _, err := svc.RetryIssue(ctx, issue.ID); err == nil {
		t.Fatalf("expected active run retry error")
	}
	run.Status = db.RunStatusSucceeded
	if err := dbClient.SaveRun(ctx, run); err != nil {
		t.Fatalf("save run status failed: %v", err)
	}
	issue.CurrentRunID = &run.ID
	issue.LifecycleStatus = db.IssueLifecycleRework
	issue.CloseReason = ptrString(db.IssueCloseReasonManual)
	if err := dbClient.SaveIssue(ctx, issue); err != nil {
		t.Fatalf("save issue rework failed: %v", err)
	}
	retried, err := svc.RetryIssue(ctx, issue.ID)
	if err != nil || retried.LifecycleStatus != db.IssueLifecycleRegistered || retried.CloseReason != nil {
		t.Fatalf("RetryIssue mismatch: issue=%#v err=%v", retried, err)
	}

	if _, err := svc.CancelRun(ctx, 999999, "reason"); err == nil {
		t.Fatalf("expected cancel missing run error")
	}
	run2 := &db.IssueRun{
		IssueID:          issue.ID,
		RunNo:            2,
		RunKind:          db.RunKindMerge,
		TriggerType:      db.TriggerManual,
		Status:           db.RunStatusQueued,
		AgentRole:        db.AgentRoleMerge,
		LoopStep:         1,
		MaxLoopStep:      5,
		QueuedAt:         time.Now(),
		BranchName:       "agent/issue-101-merge",
		GitTreePath:      "/tmp/git-tree2",
		AgentRunDir:      "/tmp/agent-run2",
		MaxConflictRetry: 3,
	}
	if err := dbClient.CreateRun(ctx, run2); err != nil {
		t.Fatalf("create run2 failed: %v", err)
	}
	issue.CurrentRunID = &run2.ID
	issue.LifecycleStatus = db.IssueLifecycleRunning
	if err := dbClient.SaveIssue(ctx, issue); err != nil {
		t.Fatalf("save issue running failed: %v", err)
	}
	canceled, err := svc.CancelRun(ctx, run2.ID, "  need stop ")
	if err != nil || canceled.Status != db.RunStatusCanceled || canceled.ErrorSummary == nil || !strings.Contains(*canceled.ErrorSummary, "need stop") {
		t.Fatalf("CancelRun mismatch: run=%#v err=%v", canceled, err)
	}
	issueAfterCancel, _ := dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfterCancel == nil || issueAfterCancel.CurrentRunID != nil || issueAfterCancel.LifecycleStatus != db.IssueLifecycleVerified {
		t.Fatalf("issue should be unbound and verified after merge run cancel: %#v", issueAfterCancel)
	}
	if _, err := svc.CancelRun(ctx, run2.ID, "again"); err == nil {
		t.Fatalf("expected non-cancelable error for terminal run")
	}

	if _, err := svc.ResetProjectSyncCursor(ctx, "missing"); err == nil {
		t.Fatalf("expected reset missing project error")
	}
	project.LastIssueSyncAt = ptrTime(time.Now())
	if err := dbClient.SaveProject(ctx, project); err != nil {
		t.Fatalf("save project sync cursor failed: %v", err)
	}
	resetProject, err := svc.ResetProjectSyncCursor(ctx, project.ProjectKey)
	if err != nil || resetProject == nil || resetProject.LastIssueSyncAt != nil {
		t.Fatalf("ResetProjectSyncCursor mismatch: row=%#v err=%v", resetProject, err)
	}

	metrics, err := svc.GetOpsMetrics(ctx)
	if err != nil || metrics == nil || metrics.Projects.Total <= 0 || metrics.Runs.Total <= 0 || metrics.Issues.Total <= 0 {
		t.Fatalf("GetOpsMetrics mismatch: metrics=%#v err=%v", metrics, err)
	}

	if err := svc.GuardAdmin(nil); err == nil {
		t.Fatalf("expected GuardAdmin nil user error")
	}
	if err := svc.GuardAdmin(&AuthUser{IsAdmin: false}); err == nil {
		t.Fatalf("expected GuardAdmin non-admin error")
	}
	if err := svc.GuardAdmin(&AuthUser{IsAdmin: true}); err != nil {
		t.Fatalf("GuardAdmin admin should pass: %v", err)
	}

	if !strings.Contains(svc.Describe(), "env=test") {
		t.Fatalf("Describe should include env, got: %s", svc.Describe())
	}
}

func ptrString(v string) *string {
	s := v
	return &s
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
