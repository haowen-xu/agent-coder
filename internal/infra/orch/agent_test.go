package orch

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	agentbase "github.com/haowen-xu/agent-coder/internal/infra/agent/base"
)

// TestOrchAgentKindsAndProjectLimits 验证不同 OrchAgent 子类的 kind 与 project 限流策略。
func TestOrchAgentKindsAndProjectLimits(t *testing.T) {
	t.Parallel()

	opts := AgentOptions{ProjectKey: "p1"}
	plan := NewOrchPlanAgent(opts)
	dev := NewOrchDevAgent(opts)
	review := NewOrchReviewAgent(opts)
	merge := NewOrchMergeAgent(opts)

	if plan.Kind() != AgentKindPlan || plan.ProjectWorkerLimit() != 0 {
		t.Fatalf("plan agent mismatch: kind=%s limit=%d", plan.Kind(), plan.ProjectWorkerLimit())
	}
	if dev.Kind() != AgentKindDev || dev.ProjectWorkerLimit() != 0 {
		t.Fatalf("dev agent mismatch: kind=%s limit=%d", dev.Kind(), dev.ProjectWorkerLimit())
	}
	if review.Kind() != AgentKindReview || review.ProjectWorkerLimit() != 0 {
		t.Fatalf("review agent mismatch: kind=%s limit=%d", review.Kind(), review.ProjectWorkerLimit())
	}
	if merge.Kind() != AgentKindMerge || merge.ProjectWorkerLimit() >= 0 {
		t.Fatalf("merge agent mismatch: kind=%s limit=%d", merge.Kind(), merge.ProjectWorkerLimit())
	}
}

// TestOrchAgentRunWithAgentClient 验证 OrchAgent 默认调用 AgentClient.Run 并落状态文件。
func TestOrchAgentRunWithAgentClient(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runDir := filepath.Join(root, "run-1")
	fakeAgent := &fakeAgentClient{
		result: &agentbase.InvokeResult{
			Decision: agentbase.Decision{
				Role:     "dev",
				Decision: "pass",
				Summary:  "ok",
			},
		},
	}
	workdir := NewWorkDir(root)
	dev := NewOrchDevAgent(AgentOptions{
		ProjectKey:  "p1",
		AgentClient: fakeAgent,
		RepoClient:  &fakeRepoClient{},
		WorkDir:     workdir,
		InvokeRequest: agentbase.InvokeRequest{
			RunKind: "dev",
			Role:    "dev",
			WorkDir: filepath.Join(root, "git-tree"),
			RunDir:  runDir,
			Timeout: time.Second,
		},
	})
	if err := dev.Run(context.Background()); err != nil {
		t.Fatalf("run dev agent failed: %v", err)
	}
	if dev.LastResult() == nil || dev.LastResult().Decision.Decision != "pass" {
		t.Fatalf("missing result after run: %#v", dev.LastResult())
	}
	if len(fakeAgent.calls) != 1 {
		t.Fatalf("expected 1 invoke call, got %d", len(fakeAgent.calls))
	}
	state, err := workdir.ReadState(runDir)
	if err != nil {
		t.Fatalf("read state failed: %v", err)
	}
	if state.Status != "succeeded" || state.Kind != AgentKindDev {
		t.Fatalf("unexpected state: %#v", state)
	}
}

// TestOrchAgentRunWithRunner 验证 Runner 覆写与失败状态落盘。
func TestOrchAgentRunWithRunner(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runDir := filepath.Join(root, "run-2")
	workdir := NewWorkDir(root)
	wantErr := errors.New("runner failed")

	merge := NewOrchMergeAgent(AgentOptions{
		ProjectKey: "p2",
		WorkDir:    workdir,
		InvokeRequest: agentbase.InvokeRequest{
			RunDir: runDir,
		},
		Runner: func(_ context.Context, _ RuntimeAgent) error {
			return wantErr
		},
	})
	err := merge.Run(context.Background())
	if err == nil || err.Error() != wantErr.Error() {
		t.Fatalf("expected runner error, got: %v", err)
	}
	state, readErr := workdir.ReadState(runDir)
	if readErr != nil {
		t.Fatalf("read state failed: %v", readErr)
	}
	if state.Status != "failed" {
		t.Fatalf("state should be failed, got: %#v", state)
	}
}
