package orch

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// OrchWorkerQueue 管理 OrchAgent 并发执行。
// 限制规则：
// 1. 全局并发不超过 maxWorkers。
// 2. per-project 并发按 agent.ProjectWorkerLimit 决定（merge 默认走 maxProjectWorkers）。
type OrchWorkerQueue struct {
	maxWorkers        int
	maxProjectWorkers int
	globalSem         chan struct{}

	mu          sync.Mutex
	projectSems map[string]chan struct{}
}

// NewOrchWorkerQueue 创建 OrchWorkerQueue。
func NewOrchWorkerQueue(maxWorkers int, maxProjectWorkers int) *OrchWorkerQueue {
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	if maxProjectWorkers < 0 {
		maxProjectWorkers = 0
	}
	return &OrchWorkerQueue{
		maxWorkers:        maxWorkers,
		maxProjectWorkers: maxProjectWorkers,
		globalSem:         make(chan struct{}, maxWorkers),
		projectSems:       make(map[string]chan struct{}),
	}
}

// MaxWorkers 返回全局并发上限。
func (q *OrchWorkerQueue) MaxWorkers() int { return q.maxWorkers }

// MaxProjectWorkers 返回默认 per-project 并发上限。
func (q *OrchWorkerQueue) MaxProjectWorkers() int { return q.maxProjectWorkers }

// Submit 异步提交一个 OrchAgent，返回结果 channel。
func (q *OrchWorkerQueue) Submit(ctx context.Context, agent OrchAgent) <-chan error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		if agent == nil {
			ch <- fmt.Errorf("orch agent is nil")
			return
		}
		if err := q.acquireGlobal(ctx); err != nil {
			ch <- err
			return
		}
		projectSem, gotProjectSlot, err := q.acquireProject(ctx, agent)
		if err != nil {
			q.releaseGlobal()
			ch <- err
			return
		}
		defer func() {
			if gotProjectSlot {
				q.releaseProject(projectSem)
			}
			q.releaseGlobal()
		}()

		ch <- agent.Run(ctx)
	}()
	return ch
}

// RunAndWait 提交并等待一批 OrchAgent 执行完成。
func (q *OrchWorkerQueue) RunAndWait(ctx context.Context, agents []OrchAgent) []error {
	if len(agents) == 0 {
		return nil
	}
	chs := make([]<-chan error, 0, len(agents))
	for _, agent := range agents {
		chs = append(chs, q.Submit(ctx, agent))
	}
	errs := make([]error, 0, len(chs))
	for _, ch := range chs {
		err, ok := <-ch
		if !ok {
			continue
		}
		errs = append(errs, err)
	}
	return errs
}

func (q *OrchWorkerQueue) acquireGlobal(ctx context.Context) error {
	select {
	case q.globalSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *OrchWorkerQueue) releaseGlobal() {
	select {
	case <-q.globalSem:
	default:
	}
}

func (q *OrchWorkerQueue) acquireProject(ctx context.Context, agent OrchAgent) (chan struct{}, bool, error) {
	limit := agent.ProjectWorkerLimit()
	if limit < 0 {
		limit = q.maxProjectWorkers
	}
	if limit <= 0 {
		return nil, false, nil
	}

	key := normalizeProjectKey(agent.ProjectKey())
	sem := q.getOrCreateProjectSem(key, limit)
	select {
	case sem <- struct{}{}:
		return sem, true, nil
	case <-ctx.Done():
		return nil, false, ctx.Err()
	}
}

func (q *OrchWorkerQueue) releaseProject(sem chan struct{}) {
	if sem == nil {
		return
	}
	select {
	case <-sem:
	default:
	}
}

func (q *OrchWorkerQueue) getOrCreateProjectSem(projectKey string, limit int) chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()
	sem, ok := q.projectSems[projectKey]
	if !ok || cap(sem) != limit {
		sem = make(chan struct{}, limit)
		q.projectSems[projectKey] = sem
	}
	return sem
}

func normalizeProjectKey(projectKey string) string {
	key := strings.TrimSpace(projectKey)
	if key == "" {
		return "__default_project__"
	}
	return key
}
