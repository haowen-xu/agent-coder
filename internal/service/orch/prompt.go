package orch

import (
	"context"
	"fmt"
	"strings"

	db "github.com/haowen-xu/agent-coder/internal/dal"
)

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

// initialRole 执行相关逻辑。
func initialRole(runKind string) string {
	if runKind == db.RunKindMerge {
		return db.AgentRoleMerge
	}
	return db.AgentRoleDev
}

// shouldUseSandboxForRole 执行相关逻辑。
func shouldUseSandboxForRole(project db.Project, role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case db.AgentRoleDev, db.AgentRoleMerge:
		return false
	case db.AgentRoleReview, "plan":
		return project.SandboxPlanReview
	default:
		return false
	}
}
