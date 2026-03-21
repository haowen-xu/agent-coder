package orch

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestOrchWorkerQueueMergePerProjectLimit 验证 merge 的 per-project 并发限制生效。
func TestOrchWorkerQueueMergePerProjectLimit(t *testing.T) {
	t.Parallel()

	q := NewOrchWorkerQueue(4, 1)

	var totalRunning int64
	var totalMax int64
	var mu sync.Mutex
	activeMerge := map[string]int{}
	maxMerge := map[string]int{}

	newMerge := func(projectKey string) OrchAgent {
		return &fakeQueueAgent{
			kind:    AgentKindMerge,
			project: projectKey,
			limit:   -1, // 使用 queue 的 maxProjectWorkers
			runFn: func(_ context.Context) error {
				nowRunning := atomic.AddInt64(&totalRunning, 1)
				for {
					old := atomic.LoadInt64(&totalMax)
					if nowRunning <= old || atomic.CompareAndSwapInt64(&totalMax, old, nowRunning) {
						break
					}
				}
				mu.Lock()
				activeMerge[projectKey]++
				if activeMerge[projectKey] > maxMerge[projectKey] {
					maxMerge[projectKey] = activeMerge[projectKey]
				}
				mu.Unlock()

				time.Sleep(40 * time.Millisecond)

				mu.Lock()
				activeMerge[projectKey]--
				mu.Unlock()
				atomic.AddInt64(&totalRunning, -1)
				return nil
			},
		}
	}

	agents := []OrchAgent{
		newMerge("p1"),
		newMerge("p1"),
		newMerge("p1"),
		newMerge("p2"),
	}
	errs := q.RunAndWait(context.Background(), agents)
	for _, err := range errs {
		if err != nil {
			t.Fatalf("queue returned error: %v", err)
		}
	}
	if maxMerge["p1"] != 1 {
		t.Fatalf("merge per-project limit for p1 should be 1, got %d", maxMerge["p1"])
	}
	if maxMerge["p2"] != 1 {
		t.Fatalf("merge per-project limit for p2 should be 1, got %d", maxMerge["p2"])
	}
	if totalMax > 4 {
		t.Fatalf("global max workers exceeded: %d", totalMax)
	}
}

// TestOrchWorkerQueueDevNoProjectLimit 验证 dev 不受 per-project 限流影响。
func TestOrchWorkerQueueDevNoProjectLimit(t *testing.T) {
	t.Parallel()

	q := NewOrchWorkerQueue(3, 1)
	var running int64
	var maxRunning int64

	agents := make([]OrchAgent, 0, 3)
	for i := 0; i < 3; i++ {
		agents = append(agents, &fakeQueueAgent{
			kind:    AgentKindDev,
			project: "same-project",
			limit:   0, // dev/review/plan 不启用 per-project 限流
			runFn: func(_ context.Context) error {
				v := atomic.AddInt64(&running, 1)
				for {
					old := atomic.LoadInt64(&maxRunning)
					if v <= old || atomic.CompareAndSwapInt64(&maxRunning, old, v) {
						break
					}
				}
				time.Sleep(30 * time.Millisecond)
				atomic.AddInt64(&running, -1)
				return nil
			},
		})
	}

	errs := q.RunAndWait(context.Background(), agents)
	for _, err := range errs {
		if err != nil {
			t.Fatalf("queue returned error: %v", err)
		}
	}
	if maxRunning < 2 {
		t.Fatalf("dev agents should run concurrently without per-project limit, got max=%d", maxRunning)
	}
}
