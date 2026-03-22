package orch

import (
	"context"
	"log/slog"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	infraorch "github.com/haowen-xu/agent-coder/internal/infra/orch"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
	"github.com/haowen-xu/agent-coder/internal/utils"
)

// RunLoop 是 *Service 的方法实现。
func (s *Service) RunLoop(ctx context.Context) error {
	if !s.cfg.Scheduler.Enabled {
		s.log.Info("scheduler disabled, run worker once")
		return s.RunOnce(ctx)
	}

	ticker := time.NewTicker(s.cfg.Scheduler.RunEveryDuration())
	defer ticker.Stop()

	for {
		if err := s.RunOnce(ctx); err != nil {
			s.log.Error("worker tick failed", slog.Any("error", err))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// RunOnce 是 *Service 的方法实现。
func (s *Service) RunOnce(ctx context.Context) error {
	projects, err := s.db.ListEnabledProjects(ctx)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if !s.shouldPollProject(project) {
			continue
		}
		if err := s.syncProjectIssues(ctx, project); err != nil {
			s.log.Error("sync project issues failed", slog.String("project_key", project.ProjectKey), slog.Any("error", err))
		}
		s.lastPolled[project.ID] = utils.NowUTC()
	}
	if err := s.scheduleRuns(ctx); err != nil {
		return err
	}

	runs := make([]*db.IssueRun, 0, runClaimBatchSize)
	for i := 0; i < runClaimBatchSize; i++ {
		run, err := s.db.ClaimNextQueuedRun(ctx)
		if err != nil {
			return err
		}
		if run == nil {
			break
		}
		runs = append(runs, run)
	}
	if len(runs) == 0 {
		return nil
	}

	agents := make([]infraorch.OrchAgent, 0, len(runs))
	for _, run := range runs {
		agent, err := s.newRunOrchAgent(ctx, run)
		if err != nil {
			return err
		}
		agents = append(agents, agent)
	}
	for i, err := range s.ensureOrchQueue().RunAndWait(ctx, agents) {
		if err != nil {
			s.log.Error("execute run failed",
				slog.Uint64("run_id", uint64(runs[i].ID)),
				slog.Any("error", err),
			)
		}
	}
	return nil
}

// newRunOrchAgent 构造 run 级 OrchAgent。
func (s *Service) newRunOrchAgent(ctx context.Context, run *db.IssueRun) (infraorch.OrchAgent, error) {
	issue, err := s.db.GetIssueByID(ctx, run.IssueID)
	if err != nil {
		return nil, err
	}
	projectKey := ""
	var repoClient repocommon.Client
	if issue != nil {
		project, err := s.db.GetProjectByID(ctx, issue.ProjectID)
		if err != nil {
			return nil, err
		}
		if project != nil {
			projectKey = project.ProjectKey
			repoClient, err = s.newRepoClient(*project)
			if err != nil {
				return nil, err
			}
		}
	}

	opts := infraorch.AgentOptions{
		ProjectKey:  projectKey,
		AgentClient: s.agent,
		RepoClient:  repoClient,
		WorkDir:     s.ensureOrchWorkDir(),
		Runner: func(ctx context.Context, _ infraorch.RuntimeAgent) error {
			return s.executeRun(ctx, run)
		},
	}
	if run.RunKind == db.RunKindMerge {
		return infraorch.NewOrchMergeAgent(opts), nil
	}
	return infraorch.NewOrchDevAgent(opts), nil
}
