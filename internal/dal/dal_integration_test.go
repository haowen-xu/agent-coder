package db

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
)

func newTestDBClient(t *testing.T) *Client {
	t.Helper()

	ctx := context.Background()
	cfg := appcfg.DBConfig{
		Enabled:         true,
		Driver:          "sqlite",
		SQLitePath:      filepath.Join(t.TempDir(), "test.db"),
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: "1m",
		AutoMigrate:     true,
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, err := New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("new test db failed: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func mustCreateProject(t *testing.T, c *Client, key string, enabled bool) *Project {
	t.Helper()

	row := &Project{
		ProjectKey:       key,
		ProjectSlug:      "group/" + key,
		Name:             key,
		Provider:         ProviderGitLab,
		ProviderURL:      "https://gitlab.example.com/api/v4",
		RepoURL:          "https://gitlab.example.com/group/" + key,
		DefaultBranch:    "main",
		CredentialRef:    "ref_" + key,
		PollIntervalSec:  60,
		Enabled:          enabled,
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

func mustCreateIssue(t *testing.T, c *Client, projectID uint, iid int64, lifecycle string) *Issue {
	t.Helper()

	now := time.Now()
	row := &Issue{
		ProjectID:       projectID,
		IssueIID:        iid,
		Title:           "issue",
		State:           "opened",
		LabelsJSON:      "[]",
		RegisteredAt:    now,
		LifecycleStatus: lifecycle,
		IssueDir:        "/tmp/issue",
		LastSyncedAt:    now,
	}
	if err := c.CreateIssue(context.Background(), row); err != nil {
		t.Fatalf("create issue failed: %v", err)
	}
	return row
}

func mustCreateRun(t *testing.T, c *Client, issueID uint, runNo int, status, runKind string, queuedAt time.Time) *IssueRun {
	t.Helper()

	row := &IssueRun{
		IssueID:          issueID,
		RunNo:            runNo,
		RunKind:          runKind,
		TriggerType:      TriggerScheduler,
		Status:           status,
		AgentRole:        AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      5,
		QueuedAt:         queuedAt,
		BranchName:       "agent/branch",
		GitTreePath:      "/tmp/git-tree",
		AgentRunDir:      "/tmp/agent-run",
		MaxConflictRetry: 5,
	}
	if err := c.CreateRun(context.Background(), row); err != nil {
		t.Fatalf("create run failed: %v", err)
	}
	return row
}

// TestDBNewAndBasics 用于单元测试。
func TestDBNewAndBasics(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	disabled, err := New(ctx, appcfg.DBConfig{Enabled: false, Driver: "sqlite"}, log)
	if err != nil {
		t.Fatalf("disabled db should not fail: %v", err)
	}
	if disabled.Enabled() {
		t.Fatalf("expected disabled db")
	}
	if disabled.Dialect() != "sqlite" {
		t.Fatalf("unexpected disabled dialect: %s", disabled.Dialect())
	}

	if _, _, err := buildDialector(appcfg.DBConfig{Driver: "postgres"}); err == nil {
		t.Fatalf("expected postgres missing dsn error")
	}
	if _, _, err := buildDialector(appcfg.DBConfig{Driver: "mysql"}); err == nil {
		t.Fatalf("expected unsupported driver error")
	}

	client := newTestDBClient(t)
	if !client.Enabled() {
		t.Fatalf("db should be enabled")
	}
	if client.Dialect() != "sqlite" {
		t.Fatalf("unexpected dialect: %s", client.Dialect())
	}
	if client.DB() == nil || client.SQLDB() == nil {
		t.Fatalf("db handles should not be nil")
	}

	var nilClient *Client
	if nilClient.Enabled() {
		t.Fatalf("nil client should be disabled")
	}
	if nilClient.Dialect() != "" {
		t.Fatalf("nil client dialect should be empty")
	}
	if nilClient.DB() != nil || nilClient.SQLDB() != nil {
		t.Fatalf("nil client db handles should be nil")
	}
	if err := nilClient.Close(); err != nil {
		t.Fatalf("nil client close should not fail: %v", err)
	}
}

// TestProjectAndIssueRepoFlow 用于单元测试。
func TestProjectAndIssueRepoFlow(t *testing.T) {
	ctx := context.Background()
	c := newTestDBClient(t)

	p1 := mustCreateProject(t, c, "p1", true)
	p2 := mustCreateProject(t, c, "p2", true)
	p2.Enabled = false
	if err := c.SaveProject(ctx, p2); err != nil {
		t.Fatalf("disable p2 failed: %v", err)
	}

	allProjects, err := c.ListProjects(ctx)
	if err != nil || len(allProjects) != 2 {
		t.Fatalf("ListProjects mismatch: len=%d err=%v", len(allProjects), err)
	}
	enabledProjects, err := c.ListEnabledProjects(ctx)
	if err != nil || len(enabledProjects) != 1 || enabledProjects[0].ProjectKey != "p1" {
		t.Fatalf("ListEnabledProjects mismatch: %#v err=%v", enabledProjects, err)
	}

	row, err := c.GetProjectByKey(ctx, "p1")
	if err != nil || row == nil {
		t.Fatalf("GetProjectByKey failed: row=%v err=%v", row, err)
	}
	row.Name = "p1-updated"
	row.LastIssueSyncAt = ptrTime(time.Now())
	if err := c.SaveProject(ctx, row); err != nil {
		t.Fatalf("SaveProject failed: %v", err)
	}

	byID, err := c.GetProjectByID(ctx, row.ID)
	if err != nil || byID == nil || byID.Name != "p1-updated" {
		t.Fatalf("GetProjectByID mismatch: row=%#v err=%v", byID, err)
	}

	if _, err := c.ResetProjectSyncCursorByKey(ctx, " "); err == nil {
		t.Fatalf("expected empty project key error")
	}
	notFound, err := c.ResetProjectSyncCursorByKey(ctx, "not-found")
	if err != nil || notFound != nil {
		t.Fatalf("expected nil reset result for missing project, got row=%v err=%v", notFound, err)
	}
	resetRow, err := c.ResetProjectSyncCursorByKey(ctx, "p1")
	if err != nil || resetRow == nil || resetRow.LastIssueSyncAt != nil {
		t.Fatalf("reset cursor mismatch: row=%#v err=%v", resetRow, err)
	}

	i1 := mustCreateIssue(t, c, p1.ID, 101, IssueLifecycleRegistered)
	i2 := mustCreateIssue(t, c, p1.ID, 102, IssueLifecycleRework)
	_ = mustCreateIssue(t, c, p1.ID, 103, IssueLifecycleHumanReview)

	gotIssue, err := c.GetIssueByProjectIID(ctx, p1.ID, 101)
	if err != nil || gotIssue == nil || gotIssue.ID != i1.ID {
		t.Fatalf("GetIssueByProjectIID mismatch: row=%#v err=%v", gotIssue, err)
	}
	gotIssueByID, err := c.GetIssueByID(ctx, i2.ID)
	if err != nil || gotIssueByID == nil || gotIssueByID.IssueIID != 102 {
		t.Fatalf("GetIssueByID mismatch: row=%#v err=%v", gotIssueByID, err)
	}

	issuesByProject, err := c.ListIssuesByProject(ctx, p1.ID, 10)
	if err != nil || len(issuesByProject) != 3 {
		t.Fatalf("ListIssuesByProject mismatch: len=%d err=%v", len(issuesByProject), err)
	}
	if issuesByProject[0].IssueIID != 103 {
		t.Fatalf("issues should be sorted by id desc, first issue_iid=%d", issuesByProject[0].IssueIID)
	}

	issuesByLifecycle, err := c.ListIssuesByLifecycle(ctx, IssueLifecycleRework, 10)
	if err != nil || len(issuesByLifecycle) != 1 || issuesByLifecycle[0].IssueIID != 102 {
		t.Fatalf("ListIssuesByLifecycle mismatch: %#v err=%v", issuesByLifecycle, err)
	}

	schedulingIssues, err := c.ListIssuesForScheduling(ctx, 10)
	if err != nil || len(schedulingIssues) != 2 {
		t.Fatalf("ListIssuesForScheduling mismatch: len=%d err=%v", len(schedulingIssues), err)
	}

	before := gotIssue.LastSyncedAt
	time.Sleep(5 * time.Millisecond)
	if err := c.TouchIssueSync(ctx, i1.ID); err != nil {
		t.Fatalf("TouchIssueSync failed: %v", err)
	}
	after, err := c.GetIssueByID(ctx, i1.ID)
	if err != nil || after == nil || !after.LastSyncedAt.After(before) {
		t.Fatalf("touch issue sync mismatch: after=%#v before=%s err=%v", after, before, err)
	}

	bound, err := c.BindIssueRunIfIdle(ctx, i1.ID, 999, "agent/branch-1")
	if err != nil || !bound {
		t.Fatalf("BindIssueRunIfIdle should bind: bound=%v err=%v", bound, err)
	}
	issueAfterBind, _ := c.GetIssueByID(ctx, i1.ID)
	if issueAfterBind.CurrentRunID == nil || *issueAfterBind.CurrentRunID != 999 {
		t.Fatalf("bind current_run_id mismatch: %#v", issueAfterBind.CurrentRunID)
	}
	boundAgain, err := c.BindIssueRunIfIdle(ctx, i1.ID, 1000, "agent/branch-2")
	if err != nil || boundAgain {
		t.Fatalf("second bind should fail due active run: bound=%v err=%v", boundAgain, err)
	}
}

// TestRunUserMetricPromptRepoFlow 用于单元测试。
func TestRunUserMetricPromptRepoFlow(t *testing.T) {
	ctx := context.Background()
	c := newTestDBClient(t)

	project := mustCreateProject(t, c, "p-run", true)
	issue := mustCreateIssue(t, c, project.ID, 201, IssueLifecycleRegistered)

	maxRunNo, err := c.GetMaxRunNo(ctx, issue.ID)
	if err != nil || maxRunNo != 0 {
		t.Fatalf("GetMaxRunNo empty mismatch: runNo=%d err=%v", maxRunNo, err)
	}

	r1 := mustCreateRun(t, c, issue.ID, 1, RunStatusQueued, RunKindDev, time.Now().Add(-2*time.Minute))
	r2 := mustCreateRun(t, c, issue.ID, 2, RunStatusQueued, RunKindMerge, time.Now().Add(-1*time.Minute))

	maxRunNo, err = c.GetMaxRunNo(ctx, issue.ID)
	if err != nil || maxRunNo != 2 {
		t.Fatalf("GetMaxRunNo mismatch: runNo=%d err=%v", maxRunNo, err)
	}

	nextQueued, err := c.GetNextQueuedRun(ctx)
	if err != nil || nextQueued == nil || nextQueued.ID != r1.ID {
		t.Fatalf("GetNextQueuedRun mismatch: row=%#v err=%v", nextQueued, err)
	}

	claimed, err := c.ClaimNextQueuedRun(ctx)
	if err != nil || claimed == nil || claimed.ID != r1.ID || claimed.Status != RunStatusRunning {
		t.Fatalf("ClaimNextQueuedRun mismatch: row=%#v err=%v", claimed, err)
	}
	active, err := c.GetActiveRunByIssue(ctx, issue.ID)
	if err != nil || active == nil {
		t.Fatalf("GetActiveRunByIssue should return active run: row=%#v err=%v", active, err)
	}

	claimed.Status = RunStatusSucceeded
	if err := c.SaveRun(ctx, claimed); err != nil {
		t.Fatalf("SaveRun failed: %v", err)
	}
	r2.Status = RunStatusFailed
	if err := c.SaveRun(ctx, r2); err != nil {
		t.Fatalf("SaveRun failed for r2: %v", err)
	}

	rowByID, err := c.GetRunByID(ctx, r2.ID)
	if err != nil || rowByID == nil || rowByID.Status != RunStatusFailed {
		t.Fatalf("GetRunByID mismatch: row=%#v err=%v", rowByID, err)
	}

	runList, err := c.ListRunsByIssue(ctx, issue.ID, 10)
	if err != nil || len(runList) != 2 || runList[0].RunNo != 2 {
		t.Fatalf("ListRunsByIssue mismatch: %#v err=%v", runList, err)
	}

	seq, err := c.GetNextRunSeq(ctx, r2.ID)
	if err != nil || seq != 1 {
		t.Fatalf("GetNextRunSeq initial mismatch: seq=%d err=%v", seq, err)
	}
	if err := c.AppendRunLog(ctx, &RunLog{
		RunID:     r2.ID,
		Seq:       1,
		Level:     "INFO",
		Stage:     "agent",
		EventType: "review",
		Message:   "m1",
	}); err != nil {
		t.Fatalf("AppendRunLog failed: %v", err)
	}
	if err := c.AppendRunLog(ctx, &RunLog{
		RunID:     r2.ID,
		Seq:       2,
		Level:     "INFO",
		Stage:     "agent",
		EventType: "review",
		Message:   "m2",
	}); err != nil {
		t.Fatalf("AppendRunLog second failed: %v", err)
	}
	logs, err := c.ListRunLogsByRun(ctx, r2.ID, 10)
	if err != nil || len(logs) != 2 || logs[0].Seq != 1 || logs[1].Seq != 2 {
		t.Fatalf("ListRunLogsByRun should return asc seq: %#v err=%v", logs, err)
	}
	seq, err = c.GetNextRunSeq(ctx, r2.ID)
	if err != nil || seq != 3 {
		t.Fatalf("GetNextRunSeq mismatch after logs: seq=%d err=%v", seq, err)
	}

	cnt, err := c.CountIssueRunsByStatus(ctx, issue.ID, []string{RunStatusSucceeded, RunStatusFailed})
	if err != nil || cnt != 2 {
		t.Fatalf("CountIssueRunsByStatus mismatch: cnt=%d err=%v", cnt, err)
	}
	cnt, err = c.CountIssueRunsByStatusAndKind(ctx, issue.ID, RunKindMerge, []string{RunStatusFailed})
	if err != nil || cnt != 1 {
		t.Fatalf("CountIssueRunsByStatusAndKind mismatch: cnt=%d err=%v", cnt, err)
	}

	if _, err := c.ClaimNextQueuedRun(ctx); err != nil {
		t.Fatalf("ClaimNextQueuedRun should not fail on remaining queued run: %v", err)
	}
	noQueue, err := c.ClaimNextQueuedRun(ctx)
	if err != nil || noQueue != nil {
		t.Fatalf("ClaimNextQueuedRun should return nil when no queued run: row=%#v err=%v", noQueue, err)
	}

	u := &User{Username: "u1", PasswordHash: "h1", Enabled: true}
	if err := c.CreateUser(ctx, u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	users, err := c.ListUsers(ctx)
	if err != nil || len(users) != 1 {
		t.Fatalf("ListUsers mismatch: len=%d err=%v", len(users), err)
	}
	userByName, err := c.GetUserByUsername(ctx, "u1")
	if err != nil || userByName == nil {
		t.Fatalf("GetUserByUsername failed: row=%#v err=%v", userByName, err)
	}
	userByID, err := c.GetUserByID(ctx, userByName.ID)
	if err != nil || userByID == nil {
		t.Fatalf("GetUserByID failed: row=%#v err=%v", userByID, err)
	}
	userByID.Enabled = false
	if err := c.SaveUser(ctx, userByID); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	sess := &UserSession{
		UserID:    userByName.ID,
		Token:     "tok1",
		ExpiredAt: time.Now().Add(time.Hour),
	}
	if err := c.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	gotSess, gotUser, err := c.GetSessionWithUser(ctx, "tok1")
	if err != nil || gotSess == nil || gotUser == nil {
		t.Fatalf("GetSessionWithUser mismatch: sess=%#v user=%#v err=%v", gotSess, gotUser, err)
	}
	expired := &UserSession{
		UserID:    userByName.ID,
		Token:     "tok-expired",
		ExpiredAt: time.Now().Add(-time.Hour),
	}
	if err := c.CreateSession(ctx, expired); err != nil {
		t.Fatalf("CreateSession expired failed: %v", err)
	}
	gotSess, gotUser, err = c.GetSessionWithUser(ctx, "tok-expired")
	if err != nil || gotSess != nil || gotUser != nil {
		t.Fatalf("expired session should return nil: sess=%#v user=%#v err=%v", gotSess, gotUser, err)
	}

	if err := c.EnsureBootstrapAdmin(ctx, "admin", "pass123"); err != nil {
		t.Fatalf("EnsureBootstrapAdmin create failed: %v", err)
	}
	if err := c.EnsureBootstrapAdmin(ctx, "admin", "pass123"); err != nil {
		t.Fatalf("EnsureBootstrapAdmin idempotent failed: %v", err)
	}

	templates, err := c.ListPromptTemplatesByProject(ctx, "p-run")
	if err != nil || len(templates) != 0 {
		t.Fatalf("ListPromptTemplatesByProject initial mismatch: len=%d err=%v", len(templates), err)
	}
	tpl, err := c.UpsertPromptTemplate(ctx, "p-run", RunKindDev, AgentRoleDev, "content-v1")
	if err != nil || tpl == nil || tpl.Content != "content-v1" {
		t.Fatalf("UpsertPromptTemplate create mismatch: tpl=%#v err=%v", tpl, err)
	}
	tpl, err = c.UpsertPromptTemplate(ctx, "p-run", RunKindDev, AgentRoleDev, "content-v2")
	if err != nil || tpl == nil || tpl.Content != "content-v2" {
		t.Fatalf("UpsertPromptTemplate update mismatch: tpl=%#v err=%v", tpl, err)
	}
	if err := c.DeletePromptTemplate(ctx, "p-run", RunKindDev, AgentRoleDev); err != nil {
		t.Fatalf("DeletePromptTemplate failed: %v", err)
	}

	totalProjects, enabledProjects, err := c.CountProjects(ctx)
	if err != nil || totalProjects <= 0 || enabledProjects <= 0 {
		t.Fatalf("CountProjects mismatch: total=%d enabled=%d err=%v", totalProjects, enabledProjects, err)
	}
	totalIssues, err := c.CountIssues(ctx)
	if err != nil || totalIssues <= 0 {
		t.Fatalf("CountIssues mismatch: total=%d err=%v", totalIssues, err)
	}
	totalRuns, err := c.CountRuns(ctx)
	if err != nil || totalRuns <= 0 {
		t.Fatalf("CountRuns mismatch: total=%d err=%v", totalRuns, err)
	}
	issueLifecycle, err := c.CountIssuesByLifecycle(ctx)
	if err != nil || issueLifecycle[IssueLifecycleRegistered] <= 0 {
		t.Fatalf("CountIssuesByLifecycle mismatch: map=%v err=%v", issueLifecycle, err)
	}
	runByStatus, err := c.CountRunsByStatus(ctx)
	if err != nil || runByStatus[RunStatusSucceeded] <= 0 {
		t.Fatalf("CountRunsByStatus mismatch: map=%v err=%v", runByStatus, err)
	}
	runByKind, err := c.CountRunsByKind(ctx)
	if err != nil || runByKind[RunKindDev] <= 0 {
		t.Fatalf("CountRunsByKind mismatch: map=%v err=%v", runByKind, err)
	}

	var bad Client
	if _, err := bad.ListProjects(ctx); err == nil || !strings.Contains(err.Error(), "db is not initialized") {
		t.Fatalf("expected db not initialized error, got: %v", err)
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
