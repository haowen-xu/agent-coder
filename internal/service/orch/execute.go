package orch

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/base"
	infraorch "github.com/haowen-xu/agent-coder/internal/infra/orch"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
	"github.com/haowen-xu/agent-coder/internal/utils"
)

// executeRun 是 *Service 的方法实现。
func (s *Service) executeRun(ctx context.Context, run *db.IssueRun) error {
	issue, err := s.db.GetIssueByID(ctx, run.IssueID)
	if err != nil {
		return err
	}
	if issue == nil {
		return nil
	}
	project, err := s.db.GetProjectByID(ctx, issue.ProjectID)
	if err != nil {
		return err
	}
	if project == nil {
		return nil
	}
	repoToken, err := s.repoAuthToken(ctx, *project)
	if err != nil {
		return err
	}
	repoClient, err := s.newRepoClient(*project)
	if err != nil {
		return err
	}

	run.Status = db.RunStatusRunning
	if run.StartedAt == nil || run.StartedAt.IsZero() {
		now := utils.NowUTC()
		run.StartedAt = &now
	}
	repoPath, err := s.git.EnsureProjectRepo(ctx, s.cfg.Work.WorkDir, strings.TrimSpace(project.RepoURL), project.ProjectKey, repoToken)
	if err != nil {
		return err
	}
	if err := s.git.EnsureIssueWorktree(ctx, repoPath, run.GitTreePath, run.BranchName, project.DefaultBranch, repoToken); err != nil {
		return err
	}
	if err := os.MkdirAll(run.GitTreePath, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(run.AgentRunDir, 0o755); err != nil {
		return err
	}
	if err := s.db.SaveRun(ctx, run); err != nil {
		return err
	}

	_ = repoClient.SetIssueLabels(ctx, *project, issue.IssueIID, []string{project.LabelAgentReady, project.LabelInProgress})

	if run.RunKind == db.RunKindMerge {
		conflict, mergeOut, mergeErr := s.git.TryMergeDefault(ctx, run.GitTreePath, project.DefaultBranch, repoToken)
		if mergeErr != nil {
			run.Status = db.RunStatusFailed
			run.ErrorSummary = stringPtr(mergeErr.Error())
			if saveErr := s.db.SaveRun(ctx, run); saveErr != nil {
				return saveErr
			}
			return s.markMergeFailure(ctx, repoClient, *project, issue, run, mergeErr.Error())
		}
		if conflict {
			run.ConflictRetryCount++
			_ = s.appendDecisionLog(ctx, run.ID, "conflict", "merge_detected", base.Decision{
				Role:       db.AgentRoleMerge,
				Decision:   "rework",
				Summary:    "merge conflict detected",
				NextAction: "resolve conflicts",
			})
			if run.ConflictRetryCount > run.MaxConflictRetry {
				run.Status = db.RunStatusFailed
				run.ErrorSummary = stringPtr("max conflict retry exceeded: " + mergeOut)
				if saveErr := s.db.SaveRun(ctx, run); saveErr != nil {
					return saveErr
				}
				return s.markMergeFailure(ctx, repoClient, *project, issue, run, "max conflict retry exceeded")
			}
			if err := s.db.SaveRun(ctx, run); err != nil {
				return err
			}
		}
	}

	failed := false
	var lastErr string
	for step := run.LoopStep; step <= run.MaxLoopStep; step++ {
		run.LoopStep = step
		run.AgentRole = initialRole(run.RunKind)
		if err := s.db.SaveRun(ctx, run); err != nil {
			return err
		}

		invokeRes, invokeErr := s.invokeRole(ctx, *project, *issue, *run, run.AgentRole)
		if invokeErr != nil {
			lastErr = invokeErr.Error()
			failed = true
			break
		}
		_ = s.appendDecisionLog(ctx, run.ID, "agent", run.AgentRole, invokeRes.Decision)
		if invokeRes.Decision.Decision == "blocked" {
			lastErr = invokeRes.Decision.BlockingReason
			failed = true
			break
		}

		run.AgentRole = db.AgentRoleReview
		if err := s.db.SaveRun(ctx, run); err != nil {
			return err
		}
		reviewRes, reviewErr := s.invokeRole(ctx, *project, *issue, *run, db.AgentRoleReview)
		if reviewErr != nil {
			lastErr = reviewErr.Error()
			failed = true
			break
		}
		_ = s.appendDecisionLog(ctx, run.ID, "agent", db.AgentRoleReview, reviewRes.Decision)
		if reviewRes.Decision.Decision == "pass" {
			if err := s.autoCommitAndPush(ctx, issue, run, repoToken); err != nil {
				lastErr = err.Error()
				failed = true
				break
			}
			failed = false
			lastErr = ""
			break
		}
		if step >= run.MaxLoopStep {
			failed = true
			lastErr = "max_loop_step exceeded without review pass"
			break
		}
	}

	finishedAt := utils.NowUTC()
	run.FinishedAt = &finishedAt
	if failed {
		run.Status = db.RunStatusFailed
		run.ErrorSummary = &lastErr
	} else {
		run.Status = db.RunStatusSucceeded
	}
	if err := s.db.SaveRun(ctx, run); err != nil {
		return err
	}

	return s.finalizeIssue(ctx, repoClient, *project, issue, run, failed, lastErr)
}

// finalizeIssue 是 *Service 的方法实现。
func (s *Service) finalizeIssue(
	ctx context.Context,
	repoClient repocommon.Client,
	project db.Project,
	issue *db.Issue,
	run *db.IssueRun,
	failed bool,
	lastErr string,
) error {
	issue.CurrentRunID = nil
	if failed {
		retryCount, err := s.db.CountIssueRunsByStatus(ctx, issue.ID, []string{db.RunStatusFailed})
		if err != nil {
			return err
		}
		if int(retryCount) >= s.cfg.Agent.Codex.MaxRetry {
			issue.LifecycleStatus = db.IssueLifecycleFailed
		} else {
			issue.LifecycleStatus = db.IssueLifecycleRegistered
		}
		issue.CloseReason = nil
		s.upsertIssueNote(ctx, repoClient, project, issue.IssueIID, issueNoteMarkerRunStatus, "agent run failed: "+lastErr)
		return s.db.SaveIssue(ctx, issue)
	}

	switch run.RunKind {
	case db.RunKindDev:
		mr, err := repoClient.EnsureMergeRequest(ctx, project, repocommon.CreateOrUpdateMRRequest{
			SourceBranch: run.BranchName,
			TargetBranch: project.DefaultBranch,
			Title:        fmt.Sprintf("AgentCoder: issue #%d %s", issue.IssueIID, issue.Title),
			Description:  "Automated MR generated by agent-coder.",
		})
		if err != nil {
			issue.LifecycleStatus = db.IssueLifecycleRegistered
			issue.CloseReason = nil
			run.Status = db.RunStatusFailed
			run.ErrorSummary = stringPtr("ensure MR failed: " + err.Error())
			if saveErr := s.db.SaveRun(ctx, run); saveErr != nil {
				return saveErr
			}
			s.upsertIssueNote(ctx, repoClient, project, issue.IssueIID, issueNoteMarkerRunStatus, "failed to create/update MR: "+err.Error())
			return s.db.SaveIssue(ctx, issue)
		}
		if mr != nil {
			run.MRIID = &mr.IID
			run.MRURL = &mr.WebURL
			issue.MRIID = &mr.IID
			issue.MRURL = &mr.WebURL
			if saveErr := s.db.SaveRun(ctx, run); saveErr != nil {
				return saveErr
			}
		}

		issue.LifecycleStatus = db.IssueLifecycleHumanReview
		issue.CloseReason = nil
		_ = repoClient.SetIssueLabels(ctx, project, issue.IssueIID, []string{project.LabelHumanReview})
		note := buildMRReadyNote(issue.IssueIID, run.BranchName, project.DefaultBranch, mr)
		s.upsertIssueNote(ctx, repoClient, project, issue.IssueIID, issueNoteMarkerMRReady, note)
	case db.RunKindMerge:
		if issue.MRIID != nil {
			if err := repoClient.MergeMergeRequest(ctx, project, *issue.MRIID); err != nil {
				if repocommon.IsNeedHumanMerge(err) {
					reason := db.IssueCloseReasonNeedHumanMerge
					issue.LifecycleStatus = db.IssueLifecycleClosed
					issue.CloseReason = &reason
					s.upsertIssueNote(ctx, repoClient, project, issue.IssueIID, issueNoteMarkerMergeStatus, "need human merge: "+err.Error())
					return s.db.SaveIssue(ctx, issue)
				}
				run.ConflictRetryCount++
				run.Status = db.RunStatusFailed
				run.ErrorSummary = stringPtr("merge MR failed: " + err.Error())
				if saveErr := s.db.SaveRun(ctx, run); saveErr != nil {
					return saveErr
				}
				return s.markMergeFailure(ctx, repoClient, project, issue, run, err.Error())
			}
		}
		_ = repoClient.SetIssueLabels(ctx, project, issue.IssueIID, []string{project.LabelMerged})
		_ = repoClient.CloseIssue(ctx, project, issue.IssueIID)
		reason := db.IssueCloseReasonMerged
		issue.LifecycleStatus = db.IssueLifecycleClosed
		issue.CloseReason = &reason
	}
	return s.db.SaveIssue(ctx, issue)
}

// invokeRole 是 *Service 的方法实现。
func (s *Service) invokeRole(
	ctx context.Context,
	project db.Project,
	issue db.Issue,
	run db.IssueRun,
	role string,
) (*base.InvokeResult, error) {
	prompt, err := s.loadPrompt(ctx, project.ProjectKey, run.RunKind, role)
	if err != nil {
		return nil, err
	}
	composed := s.composePrompt(prompt, project, issue, run, role)
	useSandbox := shouldUseSandboxForRole(project, role)
	repoClient, err := s.newRepoClient(project)
	if err != nil {
		return nil, err
	}
	req := base.InvokeRequest{
		RunKind:    run.RunKind,
		Role:       role,
		Prompt:     composed,
		WorkDir:    run.GitTreePath,
		RunDir:     run.AgentRunDir,
		Timeout:    time.Duration(s.cfg.Agent.Codex.TimeoutSec) * time.Second,
		UseSandbox: useSandbox,
	}
	opts := infraorch.AgentOptions{
		ProjectKey:    project.ProjectKey,
		AgentClient:   s.agent,
		RepoClient:    repoClient,
		WorkDir:       s.ensureOrchWorkDir(),
		InvokeRequest: req,
	}

	var roleAgent interface {
		infraorch.OrchAgent
		LastResult() *base.InvokeResult
	}
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "plan":
		roleAgent = infraorch.NewOrchPlanAgent(opts)
	case db.AgentRoleMerge:
		roleAgent = infraorch.NewOrchMergeAgent(opts)
	case db.AgentRoleReview:
		roleAgent = infraorch.NewOrchReviewAgent(opts)
	default:
		roleAgent = infraorch.NewOrchDevAgent(opts)
	}
	if err := roleAgent.Run(ctx); err != nil {
		return nil, err
	}
	if roleAgent.LastResult() == nil {
		return nil, fmt.Errorf("orch agent result is empty for role=%s", role)
	}
	return roleAgent.LastResult(), nil
}

// appendDecisionLog 是 *Service 的方法实现。
func (s *Service) appendDecisionLog(ctx context.Context, runID uint, stage string, eventType string, decision base.Decision) error {
	seq, err := s.db.GetNextRunSeq(ctx, runID)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(decision)
	payloadStr := string(payload)
	row := &db.RunLog{
		RunID:       runID,
		Seq:         seq,
		At:          utils.NowUTC(),
		Level:       "INFO",
		Stage:       stage,
		EventType:   eventType,
		Message:     decision.Summary,
		PayloadJSON: &payloadStr,
	}
	return s.db.AppendRunLog(ctx, row)
}

// autoCommitAndPush 是 *Service 的方法实现。
func (s *Service) autoCommitAndPush(ctx context.Context, issue *db.Issue, run *db.IssueRun, repoToken string) error {
	changed, err := s.git.HasChanges(ctx, run.GitTreePath)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	msg := fmt.Sprintf("agent-coder: issue #%d loop-%d", issue.IssueIID, run.LoopStep)
	if err := s.git.CommitAll(ctx, run.GitTreePath, msg); err != nil {
		return err
	}
	return s.git.PushBranch(ctx, run.GitTreePath, run.BranchName, repoToken)
}

// markMergeFailure 是 *Service 的方法实现。
func (s *Service) markMergeFailure(
	ctx context.Context,
	repoClient repocommon.Client,
	project db.Project,
	issue *db.Issue,
	run *db.IssueRun,
	reason string,
) error {
	issue.CurrentRunID = nil
	failedMergeRuns, err := s.db.CountIssueRunsByStatusAndKind(ctx, issue.ID, db.RunKindMerge, []string{db.RunStatusFailed})
	if err != nil {
		return err
	}
	if int(failedMergeRuns) >= s.cfg.Agent.Codex.MaxRetry {
		issue.LifecycleStatus = db.IssueLifecycleFailed
	} else {
		issue.LifecycleStatus = db.IssueLifecycleVerified
	}
	issue.CloseReason = nil
	s.upsertIssueNote(ctx, repoClient, project, issue.IssueIID, issueNoteMarkerMergeStatus, "merge failed: "+reason)
	return s.db.SaveIssue(ctx, issue)
}
