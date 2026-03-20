package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joomcode/errorx"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/base"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/codex"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	infraGit "github.com/haowen-xu/agent-coder/internal/infra/git"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
	"github.com/haowen-xu/agent-coder/internal/infra/repo/gitlab"
	"github.com/haowen-xu/agent-coder/internal/infra/secret"
)

// Service 表示数据结构定义。
type Service struct {
	cfg        *appcfg.Config       // cfg 字段说明。
	log        *slog.Logger         // log 字段说明。
	db         *db.Client           // db 字段说明。
	ps         *promptstore.Service // ps 字段说明。
	agent      base.Client          // agent 字段说明。
	git        *infraGit.Client     // git 字段说明。
	secret     secret.Manager       // secret 字段说明。
	lastPolled map[uint]time.Time   // lastPolled 字段说明。
}

// New 执行相关逻辑。
func New(
	cfg *appcfg.Config,
	log *slog.Logger,
	dbClient *db.Client,
	ps *promptstore.Service,
	secretManager secret.Manager,
) *Service {
	agentClient := codex.NewClient(cfg.Agent.Codex.Binary)
	return &Service{
		cfg:        cfg,
		log:        log,
		db:         dbClient,
		ps:         ps,
		agent:      agentClient,
		git:        infraGit.NewClient(),
		secret:     secretManager,
		lastPolled: make(map[uint]time.Time),
	}
}

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
		s.lastPolled[project.ID] = time.Now()
	}
	if err := s.scheduleRuns(ctx); err != nil {
		return err
	}
	for i := 0; i < 20; i++ {
		run, err := s.db.ClaimNextQueuedRun(ctx)
		if err != nil {
			return err
		}
		if run == nil {
			break
		}
		if err := s.executeRun(ctx, run); err != nil {
			s.log.Error("execute run failed", slog.Uint64("run_id", uint64(run.ID)), slog.Any("error", err))
		}
	}
	return nil
}

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
				RegisteredAt:    time.Now(),
				LifecycleStatus: lifecycleStatus,
				IssueDir:        "",
				LastSyncedAt:    time.Now(),
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
		localIssue.LastSyncedAt = time.Now()
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
		lastIssueSyncAt = time.Now().UTC()
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
		row := &db.IssueRun{
			IssueID:          issue.ID,
			RunNo:            runNo,
			RunKind:          runKind,
			TriggerType:      triggerType,
			Status:           db.RunStatusQueued,
			AgentRole:        initialRole(runKind),
			LoopStep:         1,
			MaxLoopStep:      s.cfg.Agent.Codex.MaxLoopStep,
			QueuedAt:         time.Now(),
			BranchName:       branch,
			GitTreePath:      filepath.Join(issue.IssueDir, "git-tree"),
			AgentRunDir:      filepath.Join(issue.IssueDir, "agent", "runs", strconv.Itoa(runNo)),
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
			now := time.Now()
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
	repoClient, err := s.newRepoClient(*project)
	if err != nil {
		return err
	}

	run.Status = db.RunStatusRunning
	if run.StartedAt == nil || run.StartedAt.IsZero() {
		now := time.Now()
		run.StartedAt = &now
	}
	repoPath, err := s.git.EnsureProjectRepo(ctx, s.cfg.Work.WorkDir, project.RepoURL, project.ProjectKey)
	if err != nil {
		return err
	}
	if err := s.git.EnsureIssueWorktree(ctx, repoPath, run.GitTreePath, run.BranchName, project.DefaultBranch); err != nil {
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
		conflict, mergeOut, mergeErr := s.git.TryMergeDefault(ctx, run.GitTreePath, project.DefaultBranch)
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
			if err := s.autoCommitAndPush(ctx, issue, run); err != nil {
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

	finishedAt := time.Now()
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

// newRepoClient 是 *Service 的方法实现。
func (s *Service) newRepoClient(project db.Project) (repocommon.Client, error) {
	switch strings.ToLower(strings.TrimSpace(project.Provider)) {
	case "", db.ProviderGitLab:
		timeout := time.Duration(s.cfg.RepoHTTPTimeoutSec()) * time.Second
		return gitlab.NewClient(s.log, timeout, s.secret), nil
	default:
		return nil, errorx.IllegalArgument.New("unsupported provider: %s", project.Provider)
	}
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
		_ = repoClient.CreateIssueNote(ctx, project, issue.IssueIID, "agent run failed: "+lastErr)
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
			_ = repoClient.CreateIssueNote(ctx, project, issue.IssueIID, "failed to create/update MR: "+err.Error())
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
		_ = repoClient.CreateIssueNote(ctx, project, issue.IssueIID, note)
	case db.RunKindMerge:
		if issue.MRIID != nil {
			if err := repoClient.MergeMergeRequest(ctx, project, *issue.MRIID); err != nil {
				if repocommon.IsNeedHumanMerge(err) {
					reason := db.IssueCloseReasonNeedHumanMerge
					issue.LifecycleStatus = db.IssueLifecycleClosed
					issue.CloseReason = &reason
					_ = repoClient.CreateIssueNote(ctx, project, issue.IssueIID, "need human merge: "+err.Error())
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

// buildMRReadyNote 执行相关逻辑。
func buildMRReadyNote(
	issueIID int64,
	sourceBranch string,
	targetBranch string,
	mr *repocommon.MergeRequest,
) string {
	if mr == nil {
		return "MR is ready for human review."
	}

	mrRef := fmt.Sprintf("!%d", mr.IID)
	if strings.TrimSpace(mr.WebURL) != "" {
		mrRef = fmt.Sprintf("[!%d](%s)", mr.IID, strings.TrimSpace(mr.WebURL))
	}
	from := strings.TrimSpace(sourceBranch)
	if from == "" {
		from = strings.TrimSpace(mr.SourceBranch)
	}
	if from == "" {
		from = "-"
	}
	to := strings.TrimSpace(targetBranch)
	if to == "" {
		to = strings.TrimSpace(mr.TargetBranch)
	}
	if to == "" {
		to = "-"
	}

	return fmt.Sprintf(
		`Agent run completed. MR is ready for human review.

### AgentCoder MR
- Issue: #%d
- Merge Request: %s
- Source Branch: %s
- Target Branch: %s`,
		issueIID,
		mrRef,
		fmt.Sprintf("`%s`", from),
		fmt.Sprintf("`%s`", to),
	)
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

	return s.agent.Run(ctx, base.InvokeRequest{
		RunKind: run.RunKind,
		Role:    role,
		Prompt:  composed,
		WorkDir: run.GitTreePath,
		RunDir:  run.AgentRunDir,
		Timeout: time.Duration(s.cfg.Agent.Codex.TimeoutSec) * time.Second,
	})
}

// loadPrompt 是 *Service 的方法实现。
func (s *Service) loadPrompt(ctx context.Context, projectKey, runKind, role string) (string, error) {
	templates, err := s.ps.ListEffectiveByProject(ctx, projectKey)
	if err != nil {
		return "", err
	}
	for _, t := range templates {
		if t.RunKind == runKind && t.AgentRole == role {
			return t.Content, nil
		}
	}
	return "", fmt.Errorf("no prompt found for %s/%s", runKind, role)
}

// composePrompt 是 *Service 的方法实现。
func (s *Service) composePrompt(
	rolePrompt string,
	project db.Project,
	issue db.Issue,
	run db.IssueRun,
	role string,
) string {
	return fmt.Sprintf(
		`你在 agent-coder 中执行任务。
当前上下文：
- role: %s
- run_kind: %s
- loop_step: %d / %d
- repo_dir: %s
- run_dir: %s
- issue: %s (#%d)
- base_branch: %s
- work_branch: %s

硬约束：
1) 只在 repo_dir 内工作。
2) 不改与当前 issue 无关的文件。
3) 命令失败必须明确报告，不得伪造成功。
4) 最后一段必须输出 RESULT_JSON 代码块（严格 JSON）。

角色模板：
%s
`,
		role,
		run.RunKind,
		run.LoopStep,
		run.MaxLoopStep,
		run.GitTreePath,
		run.AgentRunDir,
		issue.Title,
		issue.IssueIID,
		project.DefaultBranch,
		run.BranchName,
		rolePrompt,
	)
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
		At:          time.Now(),
		Level:       "INFO",
		Stage:       stage,
		EventType:   eventType,
		Message:     decision.Summary,
		PayloadJSON: &payloadStr,
	}
	return s.db.AppendRunLog(ctx, row)
}

// initialRole 执行相关逻辑。
func initialRole(runKind string) string {
	if runKind == db.RunKindMerge {
		return db.AgentRoleMerge
	}
	return db.AgentRoleDev
}

// containsLabel 执行相关逻辑。
func containsLabel(labels []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, label := range labels {
		if strings.EqualFold(strings.TrimSpace(label), target) {
			return true
		}
	}
	return false
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

// autoCommitAndPush 是 *Service 的方法实现。
func (s *Service) autoCommitAndPush(ctx context.Context, issue *db.Issue, run *db.IssueRun) error {
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
	return s.git.PushBranch(ctx, run.GitTreePath, run.BranchName)
}

// stringPtr 执行相关逻辑。
func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := v
	return &s
}

// isUniqueConstraintErr 执行相关逻辑。
func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "unique constraint") ||
		strings.Contains(lower, "duplicate key") ||
		strings.Contains(lower, "uniq")
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
	_ = repoClient.CreateIssueNote(ctx, project, issue.IssueIID, "merge failed: "+reason)
	return s.db.SaveIssue(ctx, issue)
}

// issueRootDir 是 *Service 的方法实现。
func (s *Service) issueRootDir(projectID uint, issueID uint) string {
	return filepath.Join(
		s.cfg.Work.WorkDir,
		strconv.Itoa(int(projectID)),
		strconv.Itoa(int(issueID)),
	)
}
