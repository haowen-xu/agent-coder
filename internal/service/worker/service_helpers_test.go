package worker

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

// TestInitialRoleAndHelpers 用于单元测试。
func TestInitialRoleAndHelpers(t *testing.T) {
	t.Parallel()

	if got := initialRole(db.RunKindDev); got != db.AgentRoleDev {
		t.Fatalf("unexpected initial role for dev: %s", got)
	}
	if got := initialRole(db.RunKindMerge); got != db.AgentRoleMerge {
		t.Fatalf("unexpected initial role for merge: %s", got)
	}

	if !containsLabel([]string{" Agent Ready ", "Other"}, "agent ready") {
		t.Fatalf("containsLabel should be case-insensitive and trim spaces")
	}
	if containsLabel([]string{"A"}, "") {
		t.Fatalf("containsLabel should reject empty target")
	}

	if got := withIssueNoteMarker("<!-- m -->", "body"); !strings.Contains(got, "<!-- m -->") {
		t.Fatalf("marker should be inserted, got: %q", got)
	}
	if got := withIssueNoteMarker("<!-- m -->", "<!-- m -->\nbody"); strings.Count(got, "<!-- m -->") != 1 {
		t.Fatalf("marker should not duplicate, got: %q", got)
	}
	if got := withIssueNoteMarker("", "body"); got != "body" {
		t.Fatalf("empty marker should not alter body")
	}

	if stringPtr("  ") != nil {
		t.Fatalf("stringPtr should return nil for blank input")
	}
	if v := stringPtr("ok"); v == nil || *v != "ok" {
		t.Fatalf("stringPtr result mismatch")
	}

	if !isUniqueConstraintErr(errors.New("duplicate key value violates unique constraint")) {
		t.Fatalf("expected unique constraint detection")
	}
	if isUniqueConstraintErr(errors.New("plain error")) {
		t.Fatalf("unexpected unique constraint detection")
	}
}

// TestMapLifecycleByRemote 用于单元测试。
func TestMapLifecycleByRemote(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	project := db.Project{
		LabelMerged:      "Merged",
		LabelVerified:    "Verified",
		LabelRework:      "Rework",
		LabelHumanReview: "Human Review",
	}

	cases := []struct {
		name       string
		current    string
		close      *string
		labels     []string
		issueState string
		wantStatus string
		wantClose  *string
	}{
		{
			name:       "merged label wins",
			labels:     []string{"Merged"},
			issueState: "opened",
			wantStatus: db.IssueLifecycleClosed,
			wantClose:  stringPtr(db.IssueCloseReasonMerged),
		},
		{
			name:       "closed issue becomes manual closed",
			labels:     nil,
			issueState: "closed",
			wantStatus: db.IssueLifecycleClosed,
			wantClose:  stringPtr(db.IssueCloseReasonManual),
		},
		{
			name:       "verified label",
			labels:     []string{"Verified"},
			issueState: "opened",
			wantStatus: db.IssueLifecycleVerified,
			wantClose:  nil,
		},
		{
			name:       "rework label",
			labels:     []string{"Rework"},
			issueState: "opened",
			wantStatus: db.IssueLifecycleRework,
			wantClose:  nil,
		},
		{
			name:       "human review label",
			labels:     []string{"Human Review"},
			issueState: "opened",
			wantStatus: db.IssueLifecycleHumanReview,
			wantClose:  nil,
		},
		{
			name:       "current merged implies closed merged",
			current:    "merged",
			issueState: "opened",
			wantStatus: db.IssueLifecycleClosed,
			wantClose:  stringPtr(db.IssueCloseReasonMerged),
		},
		{
			name:       "preserve existing closed reason",
			current:    db.IssueLifecycleClosed,
			close:      stringPtr(db.IssueCloseReasonNeedHumanMerge),
			issueState: "opened",
			wantStatus: db.IssueLifecycleClosed,
			wantClose:  stringPtr(db.IssueCloseReasonNeedHumanMerge),
		},
		{
			name:       "default empty current becomes registered",
			current:    "",
			issueState: "opened",
			wantStatus: db.IssueLifecycleRegistered,
			wantClose:  nil,
		},
		{
			name:       "fallback keeps current",
			current:    db.IssueLifecycleFailed,
			issueState: "opened",
			wantStatus: db.IssueLifecycleFailed,
			wantClose:  nil,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotStatus, gotClose := svc.mapLifecycleByRemote(tc.current, tc.close, project, tc.labels, tc.issueState)
			if gotStatus != tc.wantStatus {
				t.Fatalf("status mismatch: got=%s want=%s", gotStatus, tc.wantStatus)
			}
			if (gotClose == nil) != (tc.wantClose == nil) {
				t.Fatalf("close reason presence mismatch: got=%v want=%v", gotClose, tc.wantClose)
			}
			if gotClose != nil && tc.wantClose != nil && *gotClose != *tc.wantClose {
				t.Fatalf("close reason mismatch: got=%s want=%s", *gotClose, *tc.wantClose)
			}
		})
	}
}

// TestBuildMRReadyNote_NoURLAndMissingBranch 用于单元测试。
func TestBuildMRReadyNote_NoURLAndMissingBranch(t *testing.T) {
	t.Parallel()

	note := buildMRReadyNote(7, "", "", &repocommon.MergeRequest{
		IID:          12,
		WebURL:       "",
		SourceBranch: "",
		TargetBranch: "",
	})
	if !strings.Contains(note, "- Merge Request: !12") {
		t.Fatalf("note should fallback to plain MR ref: %s", note)
	}
	if !strings.Contains(note, "- Source Branch: `-`") {
		t.Fatalf("source branch fallback missing: %s", note)
	}
	if !strings.Contains(note, "- Target Branch: `-`") {
		t.Fatalf("target branch fallback missing: %s", note)
	}
}

// TestRepoAuthToken 用于单元测试。
func TestRepoAuthToken(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	token, err := svc.repoAuthToken(context.Background(), db.Project{ProjectToken: stringPtr("  t1  ")})
	if err != nil || token != "t1" {
		t.Fatalf("project token should be used first, token=%q err=%v", token, err)
	}

	secretSvc := &Service{
		secret: &fakeSecretManager{
			values: map[string]string{"gitlab_ref": "token-from-secret"},
		},
	}
	token, err = secretSvc.repoAuthToken(context.Background(), db.Project{CredentialRef: "gitlab_ref"})
	if err != nil || token != "token-from-secret" {
		t.Fatalf("credential ref token mismatch, token=%q err=%v", token, err)
	}

	noSecretSvc := &Service{}
	_, err = noSecretSvc.repoAuthToken(context.Background(), db.Project{CredentialRef: "x"})
	if err == nil || !strings.Contains(err.Error(), "secret manager is not configured") {
		t.Fatalf("expected missing secret manager error, got: %v", err)
	}

	token, err = svc.repoAuthToken(context.Background(), db.Project{})
	if err != nil || token != "" {
		t.Fatalf("expected empty token without credential, token=%q err=%v", token, err)
	}
}

// TestServiceUtilities 用于单元测试。
func TestServiceUtilities(t *testing.T) {
	t.Parallel()

	svc := &Service{
		cfg: &appcfg.Config{
			Scheduler: appcfg.SchedulerConfig{PollIntervalSec: 30},
			Work:      appcfg.WorkConfig{WorkDir: "/tmp/agent-work"},
		},
		lastPolled: map[uint]time.Time{},
		log:        slog.Default(),
	}

	project := db.Project{ID: 10, PollIntervalSec: 0}
	if !svc.shouldPollProject(project) {
		t.Fatalf("first poll should run when project not tracked")
	}
	svc.lastPolled[project.ID] = time.Now()
	if svc.shouldPollProject(project) {
		t.Fatalf("shouldPollProject should be false within poll interval")
	}
	svc.lastPolled[project.ID] = time.Now().Add(-31 * time.Second)
	if !svc.shouldPollProject(project) {
		t.Fatalf("shouldPollProject should be true after poll interval")
	}

	if got := svc.issueRootDir(7, 9); !strings.Contains(got, "/tmp/agent-work/7/9") {
		t.Fatalf("unexpected issueRootDir: %s", got)
	}
}

// TestUpsertIssueNoteFallback 用于单元测试。
func TestUpsertIssueNoteFallback(t *testing.T) {
	t.Parallel()

	project := db.Project{ProjectKey: "p1"}

	repoOK := &fakeRepoClient{}
	svc := &Service{}
	svc.upsertIssueNote(context.Background(), repoOK, project, 11, "<!-- m -->", "body")
	if repoOK.upsertCalls != 1 {
		t.Fatalf("expected upsert call once, got %d", repoOK.upsertCalls)
	}
	if repoOK.createCalls != 0 {
		t.Fatalf("create should not be called when upsert succeeds")
	}
	if !strings.Contains(repoOK.lastUpsertBody, "<!-- m -->") {
		t.Fatalf("marker should be included in upsert body: %q", repoOK.lastUpsertBody)
	}

	repoFail := &fakeRepoClient{upsertErr: errors.New("upsert failed")}
	svc.upsertIssueNote(context.Background(), repoFail, project, 11, "<!-- m -->", "body")
	if repoFail.upsertCalls != 1 || repoFail.createCalls != 1 {
		t.Fatalf("expected upsert+create fallback, upsert=%d create=%d", repoFail.upsertCalls, repoFail.createCalls)
	}
}

// TestNewRepoClientUnsupported 用于单元测试。
func TestNewRepoClientUnsupported(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	_, err := svc.newRepoClient(db.Project{Provider: "github"})
	if err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("expected unsupported provider error, got: %v", err)
	}
}

type fakeSecretManager struct {
	values map[string]string
	err    error
}

func (m *fakeSecretManager) Get(_ context.Context, ref string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	v, ok := m.values[ref]
	if !ok {
		return "", errors.New("missing ref")
	}
	return v, nil
}

type fakeRepoClient struct {
	upsertErr      error
	upsertCalls    int
	createCalls    int
	lastUpsertBody string
}

func (c *fakeRepoClient) ListIssues(_ context.Context, _ db.Project, _ repocommon.ListIssuesOptions) ([]repocommon.Issue, error) {
	return nil, nil
}

func (c *fakeRepoClient) SetIssueLabels(_ context.Context, _ db.Project, _ int64, _ []string) error {
	return nil
}

func (c *fakeRepoClient) CreateIssueNote(_ context.Context, _ db.Project, _ int64, _ string) error {
	c.createCalls++
	return nil
}

func (c *fakeRepoClient) UpsertIssueNote(_ context.Context, _ db.Project, _ int64, _ string, body string) error {
	c.upsertCalls++
	c.lastUpsertBody = body
	return c.upsertErr
}

func (c *fakeRepoClient) CloseIssue(_ context.Context, _ db.Project, _ int64) error {
	return nil
}

func (c *fakeRepoClient) EnsureMergeRequest(_ context.Context, _ db.Project, _ repocommon.CreateOrUpdateMRRequest) (*repocommon.MergeRequest, error) {
	return nil, nil
}

func (c *fakeRepoClient) MergeMergeRequest(_ context.Context, _ db.Project, _ int64) error {
	return nil
}
