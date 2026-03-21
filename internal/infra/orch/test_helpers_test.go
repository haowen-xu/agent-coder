package orch

import (
	"context"
	"fmt"
	"sync"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	agentbase "github.com/haowen-xu/agent-coder/internal/infra/agent/base"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

type fakeAgentClient struct {
	mu      sync.Mutex
	calls   []agentbase.InvokeRequest
	result  *agentbase.InvokeResult
	runErr  error
	runHook func(agentbase.InvokeRequest)
}

func (f *fakeAgentClient) Name() string { return "fake-agent" }

func (f *fakeAgentClient) Run(_ context.Context, req agentbase.InvokeRequest) (*agentbase.InvokeResult, error) {
	f.mu.Lock()
	f.calls = append(f.calls, req)
	hook := f.runHook
	res := f.result
	err := f.runErr
	f.mu.Unlock()

	if hook != nil {
		hook(req)
	}
	if err != nil {
		return nil, err
	}
	if res != nil {
		cp := *res
		return &cp, nil
	}
	return &agentbase.InvokeResult{
		Decision: agentbase.Decision{
			Role:     req.Role,
			Decision: "pass",
			Summary:  "ok",
		},
	}, nil
}

type fakeRepoClient struct{}

func (f *fakeRepoClient) ListIssues(_ context.Context, _ db.Project, _ repocommon.ListIssuesOptions) ([]repocommon.Issue, error) {
	return nil, nil
}

func (f *fakeRepoClient) SetIssueLabels(_ context.Context, _ db.Project, _ int64, _ []string) error {
	return nil
}

func (f *fakeRepoClient) CreateIssueNote(_ context.Context, _ db.Project, _ int64, _ string) error {
	return nil
}

func (f *fakeRepoClient) UpsertIssueNote(_ context.Context, _ db.Project, _ int64, _ string, _ string) error {
	return nil
}

func (f *fakeRepoClient) CloseIssue(_ context.Context, _ db.Project, _ int64) error {
	return nil
}

func (f *fakeRepoClient) EnsureMergeRequest(_ context.Context, _ db.Project, _ repocommon.CreateOrUpdateMRRequest) (*repocommon.MergeRequest, error) {
	return nil, nil
}

func (f *fakeRepoClient) MergeMergeRequest(_ context.Context, _ db.Project, _ int64) error {
	return nil
}

type fakeQueueAgent struct {
	kind      AgentKind
	project   string
	limit     int
	runFn     func(context.Context) error
	execCount int
	mu        sync.Mutex
}

func (f *fakeQueueAgent) Kind() AgentKind { return f.kind }

func (f *fakeQueueAgent) ProjectKey() string { return f.project }

func (f *fakeQueueAgent) ProjectWorkerLimit() int { return f.limit }

func (f *fakeQueueAgent) Run(ctx context.Context) error {
	f.mu.Lock()
	f.execCount++
	fn := f.runFn
	f.mu.Unlock()
	if fn != nil {
		return fn(ctx)
	}
	return nil
}

func mustErr[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Sprintf("unexpected error: %v", err))
	}
	return v
}
