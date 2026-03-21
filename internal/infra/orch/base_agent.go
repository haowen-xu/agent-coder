package orch

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentbase "github.com/haowen-xu/agent-coder/internal/infra/agent/base"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

// AgentKind 表示 OrchAgent 类型。
type AgentKind string

const (
	// AgentKindPlan 是 plan 角色。
	AgentKindPlan AgentKind = "plan"
	// AgentKindDev 是 dev 角色。
	AgentKindDev AgentKind = "dev"
	// AgentKindReview 是 review 角色。
	AgentKindReview AgentKind = "review"
	// AgentKindMerge 是 merge 角色。
	AgentKindMerge AgentKind = "merge"
)

// OrchAgent 是 orchestrator 的统一执行接口。
// ProjectWorkerLimit 语义：
// 1. =0: 不启用 per-project 并发限制。
// 2. <0: 使用队列默认 maxProjectWorkers。
// 3. >0: 使用 agent 自定义 per-project 并发限制。
type OrchAgent interface {
	Kind() AgentKind
	ProjectKey() string
	ProjectWorkerLimit() int
	Run(ctx context.Context) error
}

// AgentOptions 是创建 OrchAgent 的输入参数。
type AgentOptions struct {
	ProjectKey    string
	AgentClient   agentbase.Client
	RepoClient    repocommon.Client
	WorkDir       *WorkDir
	InvokeRequest agentbase.InvokeRequest
	Runner        AgentRunner
}

// AgentRunner 允许调用方覆写 OrchAgent 的运行逻辑。
type AgentRunner func(ctx context.Context, a RuntimeAgent) error

// RuntimeAgent 暴露 OrchAgent 运行时可访问的数据。
type RuntimeAgent interface {
	Kind() AgentKind
	ProjectKey() string
	AgentClient() agentbase.Client
	RepoClient() repocommon.Client
	WorkDir() *WorkDir
	InvokeRequest() agentbase.InvokeRequest
	LastResult() *agentbase.InvokeResult
	SetLastResult(res *agentbase.InvokeResult)
}

type orchBaseAgent struct {
	kind               AgentKind
	projectKey         string
	projectWorkerLimit int
	agentClient        agentbase.Client
	repoClient         repocommon.Client
	workDir            *WorkDir
	invokeRequest      agentbase.InvokeRequest
	runner             AgentRunner
	lastResult         *agentbase.InvokeResult
}

func newBaseAgent(kind AgentKind, projectWorkerLimit int, opts AgentOptions) *orchBaseAgent {
	return &orchBaseAgent{
		kind:               kind,
		projectKey:         strings.TrimSpace(opts.ProjectKey),
		projectWorkerLimit: projectWorkerLimit,
		agentClient:        opts.AgentClient,
		repoClient:         opts.RepoClient,
		workDir:            opts.WorkDir,
		invokeRequest:      opts.InvokeRequest,
		runner:             opts.Runner,
	}
}

// Kind 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) Kind() AgentKind { return a.kind }

// ProjectKey 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) ProjectKey() string { return a.projectKey }

// ProjectWorkerLimit 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) ProjectWorkerLimit() int { return a.projectWorkerLimit }

// AgentClient 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) AgentClient() agentbase.Client { return a.agentClient }

// RepoClient 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) RepoClient() repocommon.Client { return a.repoClient }

// WorkDir 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) WorkDir() *WorkDir { return a.workDir }

// InvokeRequest 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) InvokeRequest() agentbase.InvokeRequest { return a.invokeRequest }

// LastResult 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) LastResult() *agentbase.InvokeResult { return a.lastResult }

// SetLastResult 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) SetLastResult(res *agentbase.InvokeResult) { a.lastResult = res }

// Run 是 *orchBaseAgent 的方法实现。
func (a *orchBaseAgent) Run(ctx context.Context) error {
	if a.workDir != nil {
		if err := a.workDir.EnsureFromInvoke(a.invokeRequest.WorkDir, a.invokeRequest.RunDir); err != nil {
			return err
		}
		if strings.TrimSpace(a.invokeRequest.RunDir) != "" {
			_ = a.workDir.WriteState(a.invokeRequest.RunDir, State{
				Kind:       a.kind,
				Status:     "running",
				ProjectKey: a.projectKey,
				UpdatedAt:  time.Now().UTC(),
			})
		}
	}

	if a.runner != nil {
		err := a.runner(ctx, a)
		a.writeFinalState(err)
		return err
	}
	if a.agentClient == nil {
		err := fmt.Errorf("orch agent client is required for kind=%s", a.kind)
		a.writeFinalState(err)
		return err
	}

	res, err := a.agentClient.Run(ctx, a.invokeRequest)
	if err != nil {
		a.writeFinalState(err)
		return err
	}
	a.lastResult = res
	a.writeFinalState(nil)
	return nil
}

func (a *orchBaseAgent) writeFinalState(err error) {
	if a.workDir == nil || strings.TrimSpace(a.invokeRequest.RunDir) == "" {
		return
	}
	state := State{
		Kind:       a.kind,
		Status:     "succeeded",
		ProjectKey: a.projectKey,
		UpdatedAt:  time.Now().UTC(),
	}
	if err != nil {
		state.Status = "failed"
		state.Message = strings.TrimSpace(err.Error())
	}
	_ = a.workDir.WriteState(a.invokeRequest.RunDir, state)
}
