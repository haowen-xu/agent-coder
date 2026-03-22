package orch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
	"github.com/haowen-xu/agent-coder/internal/utils"
)

// syncProjectIssues 是 *Service 的方法实现。
func (s *Service) syncProjectIssues(ctx context.Context, project db.Project) error {
	client, err := s.newRepoClient(project)
	if err != nil {
		return err
	}
	var updatedAfter *time.Time
	if project.LastIssueSyncAt != nil && !project.LastIssueSyncAt.IsZero() {
		t := project.LastIssueSyncAt.UTC()
		updatedAfter = &t
	}
	issues, err := client.ListIssues(ctx, project, repocommon.ListIssuesOptions{
		State:        "all",
		UpdatedAfter: updatedAfter,
		PerPage:      100,
		MaxPages:     20,
	})
	if err != nil {
		return err
	}
	var lastIssueSyncAt time.Time
	if project.LastIssueSyncAt != nil && !project.LastIssueSyncAt.IsZero() {
		lastIssueSyncAt = project.LastIssueSyncAt.UTC()
	}

	for _, it := range issues {
		if !it.UpdatedAt.IsZero() && it.UpdatedAt.After(lastIssueSyncAt) {
			lastIssueSyncAt = it.UpdatedAt.UTC()
		}
		localIssue, err := s.db.GetIssueByProjectIID(ctx, project.ID, it.IID)
		if err != nil {
			return err
		}
		if localIssue == nil && !containsLabel(it.Labels, project.LabelAgentReady) {
			continue
		}
		labelsJSON, _ := json.Marshal(it.Labels)
		if localIssue == nil {
			lifecycleStatus, closeReason := s.mapLifecycleByRemote(db.IssueLifecycleRegistered, nil, project, it.Labels, it.State)
			row := &db.Issue{
				ProjectID:       project.ID,
				IssueIID:        it.IID,
				Title:           it.Title,
				State:           it.State,
				LabelsJSON:      string(labelsJSON),
				RegisteredAt:    utils.NowUTC(),
				LifecycleStatus: lifecycleStatus,
				IssueDir:        "",
				LastSyncedAt:    utils.NowUTC(),
				ClosedAt:        it.ClosedAt,
				CloseReason:     closeReason,
			}
			if it.UID != "" {
				row.IssueUID = &it.UID
			}
			if err := s.db.CreateIssue(ctx, row); err != nil {
				return err
			}
			row.IssueDir = s.issueRootDir(project.ID, row.ID)
			if err := s.db.SaveIssue(ctx, row); err != nil {
				return err
			}
			continue
		}

		localIssue.Title = it.Title
		localIssue.State = it.State
		localIssue.LabelsJSON = string(labelsJSON)
		localIssue.LastSyncedAt = utils.NowUTC()
		localIssue.ClosedAt = it.ClosedAt
		localIssue.LifecycleStatus, localIssue.CloseReason = s.mapLifecycleByRemote(localIssue.LifecycleStatus, localIssue.CloseReason, project, it.Labels, it.State)
		if localIssue.IssueDir == "" {
			localIssue.IssueDir = s.issueRootDir(project.ID, localIssue.ID)
		}
		if err := s.db.SaveIssue(ctx, localIssue); err != nil {
			return err
		}
	}
	if lastIssueSyncAt.IsZero() {
		lastIssueSyncAt = utils.NowUTC()
	}
	project.LastIssueSyncAt = &lastIssueSyncAt
	if err := s.db.SaveProject(ctx, &project); err != nil {
		return err
	}
	return nil
}

// scheduleRuns 是 *Service 的方法实现。
func (s *Service) scheduleRuns(ctx context.Context) error {
	issues, err := s.db.ListIssuesForScheduling(ctx, 200)
	if err != nil {
		return err
	}
	for _, issue := range issues {
		active, err := s.db.GetActiveRunByIssue(ctx, issue.ID)
		if err != nil {
			return err
		}
		if active != nil {
			continue
		}

		maxRunNo, err := s.db.GetMaxRunNo(ctx, issue.ID)
		if err != nil {
			return err
		}
		runNo := maxRunNo + 1
		runKind := db.RunKindDev
		triggerType := db.TriggerScheduler
		if issue.LifecycleStatus == db.IssueLifecycleVerified {
			runKind = db.RunKindMerge
			triggerType = db.TriggerManual
		} else if issue.LifecycleStatus == db.IssueLifecycleRework {
			triggerType = db.TriggerRework
		}

		branch := fmt.Sprintf("agent-coder/issue-%d", issue.IssueIID)
		paths := s.ensureOrchWorkDir().BuildRunPaths(issue.ProjectID, issue.ID, runNo)
		row := &db.IssueRun{
			IssueID:          issue.ID,
			RunNo:            runNo,
			RunKind:          runKind,
			TriggerType:      triggerType,
			Status:           db.RunStatusQueued,
			AgentRole:        initialRole(runKind),
			LoopStep:         1,
			MaxLoopStep:      s.cfg.Agent.Codex.MaxLoopStep,
			QueuedAt:         utils.NowUTC(),
			BranchName:       branch,
			GitTreePath:      paths.GitTree,
			AgentRunDir:      paths.RunDir,
			MaxConflictRetry: s.cfg.Agent.Codex.MaxRetry,
		}
		if err := s.db.CreateRun(ctx, row); err != nil {
			if isUniqueConstraintErr(err) {
				s.log.Warn("skip duplicated run enqueue due to concurrent scheduler",
					slog.Uint64("issue_id", uint64(issue.ID)),
					slog.Any("error", err),
				)
				continue
			}
			return err
		}

		bound, err := s.db.BindIssueRunIfIdle(ctx, issue.ID, row.ID, row.BranchName)
		if err != nil {
			return err
		}
		if !bound {
			now := utils.NowUTC()
			row.Status = db.RunStatusCanceled
			row.FinishedAt = &now
			row.ErrorSummary = stringPtr("enqueue skipped: issue already bound by another worker")
			if saveErr := s.db.SaveRun(ctx, row); saveErr != nil {
				return saveErr
			}
			continue
		}
	}
	return nil
}

// shouldPollProject 是 *Service 的方法实现。
func (s *Service) shouldPollProject(project db.Project) bool {
	if project.PollIntervalSec <= 0 {
		project.PollIntervalSec = s.cfg.Scheduler.PollIntervalSec
	}
	last, ok := s.lastPolled[project.ID]
	if !ok {
		return true
	}
	return time.Since(last) >= time.Duration(project.PollIntervalSec)*time.Second
}

// mapLifecycleByRemote 是 *Service 的方法实现。
func (s *Service) mapLifecycleByRemote(
	current string,
	currentCloseReason *string,
	project db.Project,
	labels []string,
	issueState string,
) (string, *string) {
	if containsLabel(labels, project.LabelMerged) {
		reason := db.IssueCloseReasonMerged
		return db.IssueLifecycleClosed, &reason
	}
	if strings.EqualFold(issueState, "closed") {
		reason := db.IssueCloseReasonManual
		return db.IssueLifecycleClosed, &reason
	}
	if containsLabel(labels, project.LabelVerified) {
		return db.IssueLifecycleVerified, nil
	}
	if containsLabel(labels, project.LabelRework) {
		return db.IssueLifecycleRework, nil
	}
	if containsLabel(labels, project.LabelHumanReview) {
		return db.IssueLifecycleHumanReview, nil
	}
	if strings.EqualFold(strings.TrimSpace(current), "merged") {
		reason := db.IssueCloseReasonMerged
		return db.IssueLifecycleClosed, &reason
	}
	if strings.EqualFold(strings.TrimSpace(current), db.IssueLifecycleClosed) && currentCloseReason != nil {
		return db.IssueLifecycleClosed, currentCloseReason
	}
	if current == "" {
		return db.IssueLifecycleRegistered, nil
	}
	return current, nil
}
