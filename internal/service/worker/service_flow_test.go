package worker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/base"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

type fakeAgent struct {
	calls     []base.InvokeRequest
	byRole    map[string]base.InvokeResult
	errByRole map[string]error
	onRun     func(base.InvokeRequest)
}

func (f *fakeAgent) Name() string { return "fake-agent" }

func (f *fakeAgent) Run(_ context.Context, req base.InvokeRequest) (*base.InvokeResult, error) {
	f.calls = append(f.calls, req)
	if f.onRun != nil {
		f.onRun(req)
	}
	if err := f.errByRole[req.Role]; err != nil {
		return nil, err
	}
	if res, ok := f.byRole[req.Role]; ok {
		cp := res
		return &cp, nil
	}
	return &base.InvokeResult{
		Decision: base.Decision{
			Role:     req.Role,
			Decision: "pass",
			Summary:  "ok",
		},
	}, nil
}

type workerRepoMock struct {
	listIssuesResp []repocommon.Issue
	listIssuesErr  error

	setLabelsCalls [][]string
	upsertCalls    []string
	createCalls    []string
	closeCalls     int

	ensureMRResp *repocommon.MergeRequest
	ensureMRErr  error
	mergeErr     error
}

func (m *workerRepoMock) ListIssues(_ context.Context, _ db.Project, _ repocommon.ListIssuesOptions) ([]repocommon.Issue, error) {
	if m.listIssuesErr != nil {
		return nil, m.listIssuesErr
	}
	return m.listIssuesResp, nil
}

func (m *workerRepoMock) SetIssueLabels(_ context.Context, _ db.Project, _ int64, labels []string) error {
	cp := make([]string, len(labels))
	copy(cp, labels)
	m.setLabelsCalls = append(m.setLabelsCalls, cp)
	return nil
}

func (m *workerRepoMock) CreateIssueNote(_ context.Context, _ db.Project, _ int64, body string) error {
	m.createCalls = append(m.createCalls, body)
	return nil
}

func (m *workerRepoMock) UpsertIssueNote(_ context.Context, _ db.Project, _ int64, _ string, body string) error {
	m.upsertCalls = append(m.upsertCalls, body)
	return nil
}

func (m *workerRepoMock) CloseIssue(_ context.Context, _ db.Project, _ int64) error {
	m.closeCalls++
	return nil
}

func (m *workerRepoMock) EnsureMergeRequest(_ context.Context, _ db.Project, _ repocommon.CreateOrUpdateMRRequest) (*repocommon.MergeRequest, error) {
	if m.ensureMRErr != nil {
		return nil, m.ensureMRErr
	}
	return m.ensureMRResp, nil
}

func (m *workerRepoMock) MergeMergeRequest(_ context.Context, _ db.Project, _ int64) error {
	return m.mergeErr
}

func newWorkerTestService(t *testing.T) (*Service, *db.Client, *appcfg.Config) {
	t.Helper()
	ctx := context.Background()
	cfg := &appcfg.Config{
		Work: appcfg.WorkConfig{
			WorkDir: filepath.Join(t.TempDir(), "work"),
		},
		Scheduler: appcfg.SchedulerConfig{
			Enabled:         false,
			PollIntervalSec: 1,
			RunEvery:        "20ms",
		},
		RepoProvider: appcfg.RepoProviderConfig{
			HTTPTimeoutSec: 5,
		},
		Agent: appcfg.AgentConfig{
			Codex: appcfg.AgentCodexConfig{
				Binary:      "codex",
				TimeoutSec:  30,
				MaxRetry:    2,
				MaxLoopStep: 2,
			},
		},
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbClient, err := db.New(ctx, appcfg.DBConfig{
		Enabled:         true,
		Driver:          "sqlite",
		SQLitePath:      filepath.Join(t.TempDir(), "worker.db"),
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: "1m",
		AutoMigrate:     true,
	}, log)
	if err != nil {
		t.Fatalf("init worker db failed: %v", err)
	}
	t.Cleanup(func() { _ = dbClient.Close() })
	ps := promptstore.NewService(dbClient)
	svc := New(cfg, log, dbClient, ps, nil)
	return svc, dbClient, cfg
}

func createWorkerProject(t *testing.T, c *db.Client, providerURL, repoURL string) *db.Project {
	t.Helper()
	pid := "42"
	row := &db.Project{
		ProjectKey:       "p-worker",
		ProjectSlug:      "group/p-worker",
		Name:             "worker project",
		Provider:         db.ProviderGitLab,
		ProviderURL:      providerURL,
		RepoURL:          repoURL,
		DefaultBranch:    "main",
		IssueProjectID:   &pid,
		ProjectToken:     stringPtr("token-worker"),
		PollIntervalSec:  1,
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

func createWorkerIssue(t *testing.T, c *db.Client, projectID uint, iid int64, lifecycle string) *db.Issue {
	t.Helper()
	now := time.Now()
	row := &db.Issue{
		ProjectID:       projectID,
		IssueIID:        iid,
		Title:           "issue-" + strconv.FormatInt(iid, 10),
		State:           "opened",
		LabelsJSON:      "[]",
		RegisteredAt:    now,
		LifecycleStatus: lifecycle,
		IssueDir:        filepath.Join(t.TempDir(), "issue-"+strconv.FormatInt(iid, 10)),
		LastSyncedAt:    now,
	}
	if err := c.CreateIssue(context.Background(), row); err != nil {
		t.Fatalf("create issue failed: %v", err)
	}
	return row
}

// TestWorkerRunLoopAndRunOnce 用于单元测试。
func TestWorkerRunLoopAndRunOnce(t *testing.T) {
	svc, dbClient, _ := newWorkerTestService(t)
	ctx := context.Background()

	if err := svc.RunLoop(ctx); err != nil {
		t.Fatalf("RunLoop with scheduler disabled should pass: %v", err)
	}

	now := time.Now()
	run := &db.IssueRun{
		IssueID:          999999,
		RunNo:            1,
		RunKind:          db.RunKindDev,
		TriggerType:      db.TriggerScheduler,
		Status:           db.RunStatusQueued,
		AgentRole:        db.AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      2,
		QueuedAt:         now,
		BranchName:       "agent/ghost",
		GitTreePath:      filepath.Join(t.TempDir(), "ghost-tree"),
		AgentRunDir:      filepath.Join(t.TempDir(), "ghost-agent"),
		MaxConflictRetry: 2,
	}
	if err := dbClient.CreateRun(ctx, run); err != nil {
		t.Fatalf("create queued run failed: %v", err)
	}
	if err := svc.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce should pass: %v", err)
	}
	claimed, _ := dbClient.GetRunByID(ctx, run.ID)
	if claimed == nil || claimed.Status != db.RunStatusRunning {
		t.Fatalf("queued run should be claimed as running, got %#v", claimed)
	}

	svc.cfg.Scheduler.Enabled = true
	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- svc.RunLoop(runCtx)
	}()
	time.Sleep(40 * time.Millisecond)
	cancel()
	err := <-done
	if err == nil || err != context.Canceled {
		t.Fatalf("RunLoop should exit with context canceled, got: %v", err)
	}
}

// TestWorkerSyncAndSchedule 用于单元测试。
func TestWorkerSyncAndSchedule(t *testing.T) {
	svc, dbClient, _ := newWorkerTestService(t)
	ctx := context.Background()

	var issues []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/issues") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(issues)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	project := createWorkerProject(t, dbClient, server.URL, "https://example.com/repo.git")
	issues = []map[string]any{
		{
			"id":         1001,
			"iid":        11,
			"title":      "ready issue",
			"state":      "opened",
			"labels":     []string{"Agent Ready"},
			"web_url":    "https://example.com/11",
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		},
		{
			"id":         1002,
			"iid":        12,
			"title":      "skip issue",
			"state":      "opened",
			"labels":     []string{"Other"},
			"web_url":    "https://example.com/12",
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
	if err := svc.syncProjectIssues(ctx, *project); err != nil {
		t.Fatalf("syncProjectIssues create failed: %v", err)
	}
	list, err := dbClient.ListIssuesByProject(ctx, project.ID, 10)
	if err != nil || len(list) != 1 || list[0].IssueIID != 11 {
		t.Fatalf("sync should create only ready-labeled issue: len=%d err=%v rows=%#v", len(list), err, list)
	}
	if strings.TrimSpace(list[0].IssueDir) == "" {
		t.Fatalf("created issue should have issue dir")
	}

	issues = []map[string]any{
		{
			"id":         1001,
			"iid":        11,
			"title":      "ready issue updated",
			"state":      "opened",
			"labels":     []string{"Verified"},
			"web_url":    "https://example.com/11",
			"updated_at": time.Now().UTC().Add(time.Second).Format(time.RFC3339),
		},
	}
	if err := svc.syncProjectIssues(ctx, *project); err != nil {
		t.Fatalf("syncProjectIssues update failed: %v", err)
	}
	updated, _ := dbClient.GetIssueByProjectIID(ctx, project.ID, 11)
	if updated == nil || updated.Title != "ready issue updated" || updated.LifecycleStatus != db.IssueLifecycleVerified {
		t.Fatalf("updated issue mismatch: %#v", updated)
	}

	issueRegistered := createWorkerIssue(t, dbClient, project.ID, 21, db.IssueLifecycleRegistered)
	issueRework := createWorkerIssue(t, dbClient, project.ID, 22, db.IssueLifecycleRework)
	issueVerified := createWorkerIssue(t, dbClient, project.ID, 23, db.IssueLifecycleVerified)
	issueBound := createWorkerIssue(t, dbClient, project.ID, 24, db.IssueLifecycleRegistered)
	issueBound.CurrentRunID = ptrUint(12345)
	if err := dbClient.SaveIssue(ctx, issueBound); err != nil {
		t.Fatalf("save bound issue failed: %v", err)
	}

	activeRun := &db.IssueRun{
		IssueID:          issueRework.ID,
		RunNo:            1,
		RunKind:          db.RunKindDev,
		TriggerType:      db.TriggerScheduler,
		Status:           db.RunStatusQueued,
		AgentRole:        db.AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      2,
		QueuedAt:         time.Now(),
		BranchName:       "agent/active",
		GitTreePath:      filepath.Join(issueRework.IssueDir, "git-tree"),
		AgentRunDir:      filepath.Join(issueRework.IssueDir, "agent"),
		MaxConflictRetry: 2,
	}
	if err := dbClient.CreateRun(ctx, activeRun); err != nil {
		t.Fatalf("create active run failed: %v", err)
	}

	if err := svc.scheduleRuns(ctx); err != nil {
		t.Fatalf("scheduleRuns failed: %v", err)
	}

	runsRegistered, _ := dbClient.ListRunsByIssue(ctx, issueRegistered.ID, 10)
	if len(runsRegistered) == 0 || runsRegistered[0].RunKind != db.RunKindDev || runsRegistered[0].TriggerType != db.TriggerScheduler {
		t.Fatalf("registered issue run mismatch: %#v", runsRegistered)
	}
	runsVerified, _ := dbClient.ListRunsByIssue(ctx, issueVerified.ID, 10)
	if len(runsVerified) == 0 || runsVerified[0].RunKind != db.RunKindMerge || runsVerified[0].TriggerType != db.TriggerManual {
		t.Fatalf("verified issue should schedule merge run: %#v", runsVerified)
	}
	runsRework, _ := dbClient.ListRunsByIssue(ctx, issueRework.ID, 10)
	if len(runsRework) != 1 {
		t.Fatalf("issue with active run should not enqueue new run: %#v", runsRework)
	}
	runsBound, _ := dbClient.ListRunsByIssue(ctx, issueBound.ID, 10)
	if len(runsBound) == 0 || runsBound[0].Status != db.RunStatusCanceled || runsBound[0].ErrorSummary == nil {
		t.Fatalf("bound issue should produce canceled enqueue run: %#v", runsBound)
	}
}

// TestWorkerFinalizeAndRelatedMethods 用于单元测试。
func TestWorkerFinalizeAndRelatedMethods(t *testing.T) {
	svc, dbClient, cfg := newWorkerTestService(t)
	ctx := context.Background()
	project := createWorkerProject(t, dbClient, "https://gitlab.example.com/api/v4", "https://example.com/repo.git")
	issue := createWorkerIssue(t, dbClient, project.ID, 88, db.IssueLifecycleRunning)
	run := &db.IssueRun{
		IssueID:          issue.ID,
		RunNo:            1,
		RunKind:          db.RunKindDev,
		TriggerType:      db.TriggerScheduler,
		Status:           db.RunStatusRunning,
		AgentRole:        db.AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      2,
		QueuedAt:         time.Now(),
		StartedAt:        ptrTime(time.Now()),
		BranchName:       "agent/88",
		GitTreePath:      filepath.Join(issue.IssueDir, "git-tree"),
		AgentRunDir:      filepath.Join(issue.IssueDir, "agent"),
		MaxConflictRetry: cfg.Agent.Codex.MaxRetry,
	}
	if err := dbClient.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run failed: %v", err)
	}

	repo := &workerRepoMock{}
	if err := svc.finalizeIssue(ctx, repo, *project, issue, run, true, "boom"); err != nil {
		t.Fatalf("finalize failed-path error: %v", err)
	}
	issueAfter, _ := dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleRegistered {
		t.Fatalf("failed finalize should reset lifecycle to registered: %#v", issueAfter)
	}
	if len(repo.upsertCalls) == 0 || !strings.Contains(repo.upsertCalls[0], "agent run failed") {
		t.Fatalf("failed finalize should upsert failure note: %#v", repo.upsertCalls)
	}

	for i := 2; i <= cfg.Agent.Codex.MaxRetry+1; i++ {
		failedRun := &db.IssueRun{
			IssueID:          issue.ID,
			RunNo:            i,
			RunKind:          db.RunKindDev,
			TriggerType:      db.TriggerScheduler,
			Status:           db.RunStatusFailed,
			AgentRole:        db.AgentRoleDev,
			LoopStep:         1,
			MaxLoopStep:      2,
			QueuedAt:         time.Now(),
			BranchName:       "agent/88-fail-" + strconv.Itoa(i),
			GitTreePath:      filepath.Join(issue.IssueDir, "git-tree"),
			AgentRunDir:      filepath.Join(issue.IssueDir, "agent"),
			MaxConflictRetry: cfg.Agent.Codex.MaxRetry,
		}
		_ = dbClient.CreateRun(ctx, failedRun)
	}
	if err := svc.finalizeIssue(ctx, repo, *project, issue, run, true, "boom2"); err != nil {
		t.Fatalf("finalize failed-path max retry error: %v", err)
	}
	issueAfter, _ = dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleFailed {
		t.Fatalf("failed finalize should set failed after max retries: %#v", issueAfter)
	}

	repo.ensureMRErr = errors.New("mr failed")
	runDev := *run
	runDev.RunKind = db.RunKindDev
	runDev.Status = db.RunStatusRunning
	if err := svc.finalizeIssue(ctx, repo, *project, issue, &runDev, false, ""); err != nil {
		t.Fatalf("finalize dev ensure MR err should not return error: %v", err)
	}
	if runDev.Status != db.RunStatusFailed || runDev.ErrorSummary == nil {
		t.Fatalf("run should be marked failed when ensure MR fails: %#v", runDev)
	}

	repo.ensureMRErr = nil
	repo.ensureMRResp = &repocommon.MergeRequest{
		IID:          321,
		WebURL:       "https://example.com/mr/321",
		SourceBranch: "agent/88",
		TargetBranch: "main",
	}
	runDev2 := *run
	runDev2.RunKind = db.RunKindDev
	if err := svc.finalizeIssue(ctx, repo, *project, issue, &runDev2, false, ""); err != nil {
		t.Fatalf("finalize dev success failed: %v", err)
	}
	issueAfter, _ = dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleHumanReview || issueAfter.MRIID == nil || *issueAfter.MRIID != 321 {
		t.Fatalf("dev finalize should set MR and human review: %#v", issueAfter)
	}

	runMerge := *run
	runMerge.RunKind = db.RunKindMerge
	issue.MRIID = ptrInt64(321)
	_ = dbClient.SaveIssue(ctx, issue)
	repo.mergeErr = &repocommon.ErrNeedHumanMerge{Provider: db.ProviderGitLab, StatusCode: 409, Reason: "conflict"}
	if err := svc.finalizeIssue(ctx, repo, *project, issue, &runMerge, false, ""); err != nil {
		t.Fatalf("finalize merge need human should not return error: %v", err)
	}
	issueAfter, _ = dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleClosed || issueAfter.CloseReason == nil || *issueAfter.CloseReason != db.IssueCloseReasonNeedHumanMerge {
		t.Fatalf("need human merge finalize mismatch: %#v", issueAfter)
	}

	repo.mergeErr = nil
	if err := svc.finalizeIssue(ctx, repo, *project, issue, &runMerge, false, ""); err != nil {
		t.Fatalf("finalize merge success failed: %v", err)
	}
	issueAfter, _ = dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfter == nil || issueAfter.CloseReason == nil || *issueAfter.CloseReason != db.IssueCloseReasonMerged {
		t.Fatalf("merge finalize should set merged close reason: %#v", issueAfter)
	}
	if repo.closeCalls == 0 {
		t.Fatalf("merge finalize should call CloseIssue")
	}

	if err := svc.markMergeFailure(ctx, repo, *project, issue, &runMerge, "merge boom"); err != nil {
		t.Fatalf("markMergeFailure failed: %v", err)
	}
	issueAfter, _ = dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleVerified {
		t.Fatalf("markMergeFailure should keep verified before max retries: %#v", issueAfter)
	}

	for i := 1; i <= cfg.Agent.Codex.MaxRetry; i++ {
		failedMergeRun := &db.IssueRun{
			IssueID:          issue.ID,
			RunNo:            100 + i,
			RunKind:          db.RunKindMerge,
			TriggerType:      db.TriggerManual,
			Status:           db.RunStatusFailed,
			AgentRole:        db.AgentRoleMerge,
			LoopStep:         1,
			MaxLoopStep:      2,
			QueuedAt:         time.Now(),
			BranchName:       "agent/88-merge-fail-" + strconv.Itoa(i),
			GitTreePath:      filepath.Join(issue.IssueDir, "git-tree"),
			AgentRunDir:      filepath.Join(issue.IssueDir, "agent"),
			MaxConflictRetry: cfg.Agent.Codex.MaxRetry,
		}
		if err := dbClient.CreateRun(ctx, failedMergeRun); err != nil {
			t.Fatalf("create failed merge run %d failed: %v", i, err)
		}
	}
	if err := svc.markMergeFailure(ctx, repo, *project, issue, &runMerge, "merge boom again"); err != nil {
		t.Fatalf("markMergeFailure after retries failed: %v", err)
	}
	issueAfter, _ = dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleFailed {
		t.Fatalf("markMergeFailure should set failed after max retries: %#v", issueAfter)
	}
}

// TestWorkerPromptLogInvokeAndCommit 用于单元测试。
func TestWorkerPromptLogInvokeAndCommit(t *testing.T) {
	svc, dbClient, _ := newWorkerTestService(t)
	ctx := context.Background()
	project := createWorkerProject(t, dbClient, "https://gitlab.example.com/api/v4", "")
	project.SandboxPlanReview = true
	_ = dbClient.SaveProject(ctx, project)
	issue := createWorkerIssue(t, dbClient, project.ID, 901, db.IssueLifecycleRegistered)
	run := &db.IssueRun{
		IssueID:          issue.ID,
		RunNo:            1,
		RunKind:          db.RunKindDev,
		TriggerType:      db.TriggerScheduler,
		Status:           db.RunStatusQueued,
		AgentRole:        db.AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      2,
		QueuedAt:         time.Now(),
		BranchName:       "agent/901",
		GitTreePath:      filepath.Join(issue.IssueDir, "git-tree"),
		AgentRunDir:      filepath.Join(issue.IssueDir, "agent"),
		MaxConflictRetry: 2,
	}
	if err := dbClient.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run failed: %v", err)
	}
	_, _ = dbClient.UpsertPromptTemplate(ctx, project.ProjectKey, db.RunKindDev, db.AgentRoleDev, "dev prompt")
	_, _ = dbClient.UpsertPromptTemplate(ctx, project.ProjectKey, db.RunKindDev, db.AgentRoleReview, "review prompt")

	prompt, err := svc.loadPrompt(ctx, project.ProjectKey, db.RunKindDev, db.AgentRoleDev)
	if err != nil || prompt != "dev prompt" {
		t.Fatalf("loadPrompt mismatch: prompt=%q err=%v", prompt, err)
	}
	if _, err := svc.loadPrompt(ctx, project.ProjectKey, db.RunKindDev, "nonexistent-role"); err == nil {
		t.Fatalf("loadPrompt should fail for missing template")
	}

	composed := svc.composePrompt("role prompt", *project, *issue, *run, db.AgentRoleDev)
	if !strings.Contains(composed, "role: dev") || !strings.Contains(composed, issue.Title) {
		t.Fatalf("composePrompt should include role and issue title: %s", composed)
	}

	fa := &fakeAgent{
		byRole: map[string]base.InvokeResult{
			db.AgentRoleReview: {
				Decision: base.Decision{
					Role:     db.AgentRoleReview,
					Decision: "pass",
					Summary:  "review pass",
				},
			},
		},
		errByRole: map[string]error{},
	}
	svc.agent = fa
	invokeRes, err := svc.invokeRole(ctx, *project, *issue, *run, db.AgentRoleReview)
	if err != nil || invokeRes == nil || invokeRes.Decision.Decision != "pass" {
		t.Fatalf("invokeRole mismatch: res=%#v err=%v", invokeRes, err)
	}
	if len(fa.calls) == 0 || !fa.calls[0].UseSandbox {
		t.Fatalf("invokeRole review should enable sandbox when configured")
	}

	if err := svc.appendDecisionLog(ctx, run.ID, "agent", "review", base.Decision{
		Role:     db.AgentRoleReview,
		Decision: "pass",
		Summary:  "good",
	}); err != nil {
		t.Fatalf("appendDecisionLog failed: %v", err)
	}
	logs, err := dbClient.ListRunLogsByRun(ctx, run.ID, 10)
	if err != nil || len(logs) == 0 || logs[0].Message != "good" {
		t.Fatalf("run log append mismatch: logs=%#v err=%v", logs, err)
	}

	origin, worktree := setupWorkerGitRepo(t)
	_ = origin
	run.GitTreePath = worktree
	run.BranchName = "feature-901"
	issue.IssueIID = 901
	if err := svc.autoCommitAndPush(ctx, issue, run, ""); err != nil {
		t.Fatalf("autoCommitAndPush no-change should pass: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "new.txt"), []byte("new content\n"), 0o644); err != nil {
		t.Fatalf("write worktree file failed: %v", err)
	}
	if err := svc.autoCommitAndPush(ctx, issue, run, ""); err != nil {
		t.Fatalf("autoCommitAndPush with change failed: %v", err)
	}
	out := gitOut(t, "", "git", "ls-remote", "--heads", origin, "feature-901")
	if strings.TrimSpace(out) == "" {
		t.Fatalf("remote branch feature-901 should exist after push")
	}
}

// TestWorkerExecuteRunEarlyReturns 用于单元测试。
func TestWorkerExecuteRunEarlyReturns(t *testing.T) {
	svc, dbClient, _ := newWorkerTestService(t)
	ctx := context.Background()

	runMissingIssue := &db.IssueRun{IssueID: 999999}
	if err := svc.executeRun(ctx, runMissingIssue); err != nil {
		t.Fatalf("executeRun should return nil when issue missing: %v", err)
	}

	issue := createWorkerIssue(t, dbClient, 999998, 5001, db.IssueLifecycleRegistered)
	runMissingProject := &db.IssueRun{IssueID: issue.ID}
	if err := svc.executeRun(ctx, runMissingProject); err != nil {
		t.Fatalf("executeRun should return nil when project missing: %v", err)
	}
}

// TestWorkerExecuteRunDevSuccess 验证 executeRun 开发流成功并真实提交推送。
func TestWorkerExecuteRunDevSuccess(t *testing.T) {
	svc, dbClient, cfg := newWorkerTestService(t)
	ctx := context.Background()
	origin, _ := setupWorkerGitRepo(t)

	projectID := "42"
	var issuePutBodies []string
	var notePostBodies []string
	var createMRCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/projects/42/issues/1101":
			raw, _ := io.ReadAll(r.Body)
			issuePutBodies = append(issuePutBodies, string(raw))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && r.URL.Path == "/projects/42/issues/1101/notes":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/projects/42/issues/1101/notes":
			raw, _ := io.ReadAll(r.Body)
			notePostBodies = append(notePostBodies, string(raw))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && r.URL.Path == "/projects/42/merge_requests":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/projects/42/merge_requests":
			createMRCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"iid":321,"web_url":"https://example.com/mr/321","source_branch":"agent/1101","target_branch":"main","state":"opened"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	project := createWorkerProject(t, dbClient, server.URL, origin)
	project.IssueProjectID = &projectID
	if err := dbClient.SaveProject(ctx, project); err != nil {
		t.Fatalf("save project failed: %v", err)
	}

	issue := createWorkerIssue(t, dbClient, project.ID, 1101, db.IssueLifecycleRegistered)
	run := &db.IssueRun{
		IssueID:          issue.ID,
		RunNo:            1,
		RunKind:          db.RunKindDev,
		TriggerType:      db.TriggerScheduler,
		Status:           db.RunStatusQueued,
		AgentRole:        db.AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      2,
		QueuedAt:         time.Now(),
		BranchName:       "agent/1101",
		GitTreePath:      filepath.Join(cfg.Work.WorkDir, "issue-1101-tree"),
		AgentRunDir:      filepath.Join(cfg.Work.WorkDir, "issue-1101-run"),
		MaxConflictRetry: cfg.Agent.Codex.MaxRetry,
	}
	if err := dbClient.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run failed: %v", err)
	}
	if _, err := dbClient.UpsertPromptTemplate(ctx, project.ProjectKey, db.RunKindDev, db.AgentRoleDev, "dev prompt"); err != nil {
		t.Fatalf("upsert dev prompt failed: %v", err)
	}
	if _, err := dbClient.UpsertPromptTemplate(ctx, project.ProjectKey, db.RunKindDev, db.AgentRoleReview, "review prompt"); err != nil {
		t.Fatalf("upsert review prompt failed: %v", err)
	}

	svc.agent = &fakeAgent{
		byRole: map[string]base.InvokeResult{
			db.AgentRoleDev: {
				Decision: base.Decision{
					Role:     db.AgentRoleDev,
					Decision: "pass",
					Summary:  "dev pass",
				},
			},
			db.AgentRoleReview: {
				Decision: base.Decision{
					Role:     db.AgentRoleReview,
					Decision: "pass",
					Summary:  "review pass",
				},
			},
		},
		onRun: func(req base.InvokeRequest) {
			if req.Role == db.AgentRoleDev {
				_ = os.WriteFile(filepath.Join(req.WorkDir, "feature.txt"), []byte("feature work\n"), 0o644)
			}
		},
	}

	if err := svc.executeRun(ctx, run); err != nil {
		t.Fatalf("executeRun dev success failed: %v", err)
	}

	runAfter, err := dbClient.GetRunByID(ctx, run.ID)
	if err != nil || runAfter == nil {
		t.Fatalf("get run failed: run=%#v err=%v", runAfter, err)
	}
	if runAfter.Status != db.RunStatusSucceeded || runAfter.FinishedAt == nil {
		t.Fatalf("run should succeed with finished time: %#v", runAfter)
	}

	issueAfter, err := dbClient.GetIssueByID(ctx, issue.ID)
	if err != nil || issueAfter == nil {
		t.Fatalf("get issue failed: issue=%#v err=%v", issueAfter, err)
	}
	if issueAfter.LifecycleStatus != db.IssueLifecycleHumanReview || issueAfter.MRIID == nil || *issueAfter.MRIID != 321 {
		t.Fatalf("issue should move to human review with MR: %#v", issueAfter)
	}

	if createMRCalls != 1 {
		t.Fatalf("expected one MR create call, got %d", createMRCalls)
	}
	if len(issuePutBodies) < 2 {
		t.Fatalf("expected issue label updates at start and finalize, got %d", len(issuePutBodies))
	}
	if !strings.Contains(issuePutBodies[0], project.LabelInProgress) {
		t.Fatalf("first label update should include in-progress label: %s", issuePutBodies[0])
	}
	if !strings.Contains(issuePutBodies[len(issuePutBodies)-1], project.LabelHumanReview) {
		t.Fatalf("final label update should include human-review label: %s", issuePutBodies[len(issuePutBodies)-1])
	}
	if len(notePostBodies) == 0 || !strings.Contains(notePostBodies[0], "MR is ready for human review") {
		t.Fatalf("expected MR-ready note body, got: %#v", notePostBodies)
	}

	remoteBranch := gitOut(t, "", "git", "ls-remote", "--heads", origin, run.BranchName)
	if strings.TrimSpace(remoteBranch) == "" {
		t.Fatalf("expected pushed branch %s to exist in origin", run.BranchName)
	}

	runLogs, err := dbClient.ListRunLogsByRun(ctx, run.ID, 10)
	if err != nil {
		t.Fatalf("list run logs failed: %v", err)
	}
	if len(runLogs) < 2 {
		t.Fatalf("expected at least dev/review decision logs, got %d", len(runLogs))
	}
}

// TestWorkerExecuteRunBlockedAndMergeFailure 验证 executeRun 的阻塞和 merge 失败路径。
func TestWorkerExecuteRunBlockedAndMergeFailure(t *testing.T) {
	t.Run("blocked_dev", func(t *testing.T) {
		svc, dbClient, cfg := newWorkerTestService(t)
		ctx := context.Background()
		origin, _ := setupWorkerGitRepo(t)

		projectID := "42"
		var noteBodies []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/projects/42/issues/1201":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			case r.Method == http.MethodGet && r.URL.Path == "/projects/42/issues/1201/notes":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
			case r.Method == http.MethodPost && r.URL.Path == "/projects/42/issues/1201/notes":
				raw, _ := io.ReadAll(r.Body)
				noteBodies = append(noteBodies, string(raw))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			default:
				t.Fatalf("unexpected blocked-dev request: %s %s", r.Method, r.URL.String())
			}
		}))
		defer server.Close()

		project := createWorkerProject(t, dbClient, server.URL, origin)
		project.IssueProjectID = &projectID
		if err := dbClient.SaveProject(ctx, project); err != nil {
			t.Fatalf("save project failed: %v", err)
		}
		issue := createWorkerIssue(t, dbClient, project.ID, 1201, db.IssueLifecycleRegistered)
		run := &db.IssueRun{
			IssueID:          issue.ID,
			RunNo:            1,
			RunKind:          db.RunKindDev,
			TriggerType:      db.TriggerScheduler,
			Status:           db.RunStatusQueued,
			AgentRole:        db.AgentRoleDev,
			LoopStep:         1,
			MaxLoopStep:      2,
			QueuedAt:         time.Now(),
			BranchName:       "agent/1201",
			GitTreePath:      filepath.Join(cfg.Work.WorkDir, "issue-1201-tree"),
			AgentRunDir:      filepath.Join(cfg.Work.WorkDir, "issue-1201-run"),
			MaxConflictRetry: cfg.Agent.Codex.MaxRetry,
		}
		if err := dbClient.CreateRun(ctx, run); err != nil {
			t.Fatalf("create run failed: %v", err)
		}
		if _, err := dbClient.UpsertPromptTemplate(ctx, project.ProjectKey, db.RunKindDev, db.AgentRoleDev, "dev prompt"); err != nil {
			t.Fatalf("upsert dev prompt failed: %v", err)
		}

		svc.agent = &fakeAgent{
			byRole: map[string]base.InvokeResult{
				db.AgentRoleDev: {
					Decision: base.Decision{
						Role:           db.AgentRoleDev,
						Decision:       "blocked",
						Summary:        "blocked",
						BlockingReason: "missing dependency",
					},
				},
			},
		}
		if err := svc.executeRun(ctx, run); err != nil {
			t.Fatalf("executeRun blocked dev failed: %v", err)
		}

		runAfter, _ := dbClient.GetRunByID(ctx, run.ID)
		if runAfter == nil || runAfter.Status != db.RunStatusFailed || runAfter.ErrorSummary == nil || !strings.Contains(*runAfter.ErrorSummary, "missing dependency") {
			t.Fatalf("blocked run should fail with blocking reason: %#v", runAfter)
		}
		issueAfter, _ := dbClient.GetIssueByID(ctx, issue.ID)
		if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleRegistered {
			t.Fatalf("blocked run should reset lifecycle to registered: %#v", issueAfter)
		}
		if len(noteBodies) == 0 || !strings.Contains(noteBodies[0], "agent run failed") {
			t.Fatalf("blocked run should write failure note: %#v", noteBodies)
		}
	})

	t.Run("merge_try_merge_error", func(t *testing.T) {
		svc, dbClient, cfg := newWorkerTestService(t)
		ctx := context.Background()
		origin, _ := setupWorkerGitRepo(t)

		projectID := "42"
		var noteBodies []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/projects/42/issues/1301":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			case r.Method == http.MethodGet && r.URL.Path == "/projects/42/issues/1301/notes":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
			case r.Method == http.MethodPost && r.URL.Path == "/projects/42/issues/1301/notes":
				raw, _ := io.ReadAll(r.Body)
				noteBodies = append(noteBodies, string(raw))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			default:
				t.Fatalf("unexpected merge-error request: %s %s", r.Method, r.URL.String())
			}
		}))
		defer server.Close()

		project := createWorkerProject(t, dbClient, server.URL, origin)
		project.IssueProjectID = &projectID
		project.DefaultBranch = "missing-main"
		if err := dbClient.SaveProject(ctx, project); err != nil {
			t.Fatalf("save project failed: %v", err)
		}
		issue := createWorkerIssue(t, dbClient, project.ID, 1301, db.IssueLifecycleVerified)
		run := &db.IssueRun{
			IssueID:          issue.ID,
			RunNo:            1,
			RunKind:          db.RunKindMerge,
			TriggerType:      db.TriggerManual,
			Status:           db.RunStatusQueued,
			AgentRole:        db.AgentRoleMerge,
			LoopStep:         1,
			MaxLoopStep:      2,
			QueuedAt:         time.Now(),
			BranchName:       "agent/1301",
			GitTreePath:      filepath.Join(cfg.Work.WorkDir, "issue-1301-tree"),
			AgentRunDir:      filepath.Join(cfg.Work.WorkDir, "issue-1301-run"),
			MaxConflictRetry: cfg.Agent.Codex.MaxRetry,
		}
		if err := dbClient.CreateRun(ctx, run); err != nil {
			t.Fatalf("create run failed: %v", err)
		}
		svc.agent = &fakeAgent{
			byRole: map[string]base.InvokeResult{
				db.AgentRoleMerge:  {Decision: base.Decision{Role: db.AgentRoleMerge, Decision: "pass", Summary: "merge"}},
				db.AgentRoleReview: {Decision: base.Decision{Role: db.AgentRoleReview, Decision: "pass", Summary: "review"}},
			},
		}

		if err := svc.executeRun(ctx, run); err != nil {
			t.Fatalf("executeRun merge error path failed: %v", err)
		}

		runAfter, _ := dbClient.GetRunByID(ctx, run.ID)
		if runAfter == nil || runAfter.Status != db.RunStatusFailed || runAfter.ErrorSummary == nil {
			t.Fatalf("merge error should fail run with error summary: %#v", runAfter)
		}
		issueAfter, _ := dbClient.GetIssueByID(ctx, issue.ID)
		if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleVerified {
			t.Fatalf("single merge failure should keep issue verified: %#v", issueAfter)
		}
		if len(noteBodies) == 0 || !strings.Contains(noteBodies[0], "merge failed") {
			t.Fatalf("merge failure should write merge note: %#v", noteBodies)
		}
	})
}

// TestWorkerExecuteRunReviewLoopExhausted 验证 review 未通过且达到最大循环时的失败分支。
func TestWorkerExecuteRunReviewLoopExhausted(t *testing.T) {
	svc, dbClient, cfg := newWorkerTestService(t)
	ctx := context.Background()
	origin, _ := setupWorkerGitRepo(t)

	projectID := "42"
	var noteBodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/projects/42/issues/1401":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && r.URL.Path == "/projects/42/issues/1401/notes":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/projects/42/issues/1401/notes":
			raw, _ := io.ReadAll(r.Body)
			noteBodies = append(noteBodies, string(raw))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	project := createWorkerProject(t, dbClient, server.URL, origin)
	project.IssueProjectID = &projectID
	if err := dbClient.SaveProject(ctx, project); err != nil {
		t.Fatalf("save project failed: %v", err)
	}
	issue := createWorkerIssue(t, dbClient, project.ID, 1401, db.IssueLifecycleRegistered)
	run := &db.IssueRun{
		IssueID:          issue.ID,
		RunNo:            1,
		RunKind:          db.RunKindDev,
		TriggerType:      db.TriggerScheduler,
		Status:           db.RunStatusQueued,
		AgentRole:        db.AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      1,
		QueuedAt:         time.Now(),
		BranchName:       "agent/1401",
		GitTreePath:      filepath.Join(cfg.Work.WorkDir, "issue-1401-tree"),
		AgentRunDir:      filepath.Join(cfg.Work.WorkDir, "issue-1401-run"),
		MaxConflictRetry: cfg.Agent.Codex.MaxRetry,
	}
	if err := dbClient.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run failed: %v", err)
	}
	if _, err := dbClient.UpsertPromptTemplate(ctx, project.ProjectKey, db.RunKindDev, db.AgentRoleDev, "dev prompt"); err != nil {
		t.Fatalf("upsert dev prompt failed: %v", err)
	}
	if _, err := dbClient.UpsertPromptTemplate(ctx, project.ProjectKey, db.RunKindDev, db.AgentRoleReview, "review prompt"); err != nil {
		t.Fatalf("upsert review prompt failed: %v", err)
	}

	svc.agent = &fakeAgent{
		byRole: map[string]base.InvokeResult{
			db.AgentRoleDev: {
				Decision: base.Decision{
					Role:     db.AgentRoleDev,
					Decision: "pass",
					Summary:  "dev pass",
				},
			},
			db.AgentRoleReview: {
				Decision: base.Decision{
					Role:     db.AgentRoleReview,
					Decision: "rework",
					Summary:  "need changes",
				},
			},
		},
	}

	if err := svc.executeRun(ctx, run); err != nil {
		t.Fatalf("executeRun review loop exhausted failed: %v", err)
	}

	runAfter, _ := dbClient.GetRunByID(ctx, run.ID)
	if runAfter == nil || runAfter.Status != db.RunStatusFailed || runAfter.ErrorSummary == nil {
		t.Fatalf("run should fail when max loop exceeded: %#v", runAfter)
	}
	if !strings.Contains(*runAfter.ErrorSummary, "max_loop_step exceeded") {
		t.Fatalf("unexpected run error summary: %s", *runAfter.ErrorSummary)
	}
	issueAfter, _ := dbClient.GetIssueByID(ctx, issue.ID)
	if issueAfter == nil || issueAfter.LifecycleStatus != db.IssueLifecycleRegistered {
		t.Fatalf("issue should be reset to registered after failure: %#v", issueAfter)
	}
	if len(noteBodies) == 0 || !strings.Contains(noteBodies[0], "agent run failed") {
		t.Fatalf("expected failure issue note, got: %#v", noteBodies)
	}
}

func setupWorkerGitRepo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	seed := filepath.Join(root, "seed")
	worktree := filepath.Join(root, "worktree")

	gitOut(t, "", "git", "init", "--bare", origin)
	gitOut(t, "", "git", "clone", origin, seed)
	gitOut(t, seed, "git", "config", "user.email", "tester@example.com")
	gitOut(t, seed, "git", "config", "user.name", "tester")
	_ = os.WriteFile(filepath.Join(seed, "README.md"), []byte("hello\n"), 0o644)
	gitOut(t, seed, "git", "add", "README.md")
	gitOut(t, seed, "git", "commit", "-m", "init")
	gitOut(t, seed, "git", "push", "-u", "origin", "HEAD:main")

	gitOut(t, "", "git", "clone", origin, worktree)
	gitOut(t, worktree, "git", "checkout", "-b", "feature-901")
	gitOut(t, worktree, "git", "config", "user.email", "tester@example.com")
	gitOut(t, worktree, "git", "config", "user.name", "tester")
	return origin, worktree
}

func gitOut(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, string(out))
	}
	return string(out)
}

func ptrUint(v uint) *uint           { return &v }
func ptrInt64(v int64) *int64        { return &v }
func ptrTime(v time.Time) *time.Time { return &v }
