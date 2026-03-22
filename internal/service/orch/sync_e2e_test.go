//go:build e2e

package orch

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
)

const (
	e2eGitLabAPIBase   = "https://git.ccf-quant.com/api/v4"
	e2eGitLabProjectID = "365"
	e2eGitLabProject   = "ai-agents/agent-coder-testbed"
	e2eGitLabToken     = "glpat-p2hVy2Z6AyoMGjAJwbyeXG86MQp1OjFrCA.01.0y0sz78gm"
)

// TestWorker_SyncProjectIssues_E2E 用于单元测试。
func TestWorker_SyncProjectIssues_E2E(t *testing.T) {
	ctx := context.Background()
	workDir := t.TempDir()
	dbPath := filepath.Join(workDir, "e2e.db")

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbClient, err := db.New(ctx, appcfg.DBConfig{
		Enabled:         true,
		Driver:          "sqlite",
		SQLitePath:      dbPath,
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: "1m",
		AutoMigrate:     true,
	}, log)
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() { _ = dbClient.Close() })

	token := e2eGitLabToken
	projectID := e2eGitLabProjectID
	project := db.Project{
		ProjectKey:       "e2e-testbed",
		ProjectSlug:      e2eGitLabProject,
		Name:             "e2e-testbed",
		Provider:         db.ProviderGitLab,
		ProviderURL:      e2eGitLabAPIBase,
		RepoURL:          "https://git.ccf-quant.com/ai-agents/agent-coder-testbed",
		DefaultBranch:    "main",
		IssueProjectID:   &projectID,
		CredentialRef:    "",
		ProjectToken:     &token,
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
	if err := dbClient.CreateProject(ctx, &project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	wk := New(&appcfg.Config{
		Work: appcfg.WorkConfig{
			WorkDir: workDir,
		},
		Scheduler: appcfg.SchedulerConfig{
			PollIntervalSec: 30,
		},
		RepoProvider: appcfg.RepoProviderConfig{
			HTTPTimeoutSec: 30,
		},
		Agent: appcfg.AgentConfig{
			Codex: appcfg.AgentCodexConfig{
				Binary:      "codex",
				TimeoutSec:  60,
				MaxRetry:    5,
				MaxLoopStep: 5,
			},
		},
	}, log, dbClient, promptstore.NewService(dbClient), nil)

	if err := wk.syncProjectIssues(ctx, project); err != nil {
		t.Fatalf("syncProjectIssues() error = %v", err)
	}

	projectAfter, err := dbClient.GetProjectByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("get project by id: %v", err)
	}
	if projectAfter == nil || projectAfter.LastIssueSyncAt == nil || projectAfter.LastIssueSyncAt.IsZero() {
		t.Fatalf("expected project last_issue_sync_at to be updated")
	}

	rows, err := dbClient.ListIssuesByProject(ctx, project.ID, 200)
	if err != nil {
		t.Fatalf("list issues by project: %v", err)
	}
	for i, issue := range rows {
		if issue.ProjectID != project.ID {
			t.Fatalf("issue[%d] project_id mismatch: %d", i, issue.ProjectID)
		}
		if issue.LastSyncedAt.After(time.Now().Add(5 * time.Second)) {
			t.Fatalf("issue[%d] last_synced_at looks invalid: %s", i, issue.LastSyncedAt)
		}
	}
}
