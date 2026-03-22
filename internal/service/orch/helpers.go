package orch

import (
	"fmt"
	"strings"

	infraorch "github.com/haowen-xu/agent-coder/internal/infra/orch"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

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

// issueRootDir 是 *Service 的方法实现。
func (s *Service) issueRootDir(projectID uint, issueID uint) string {
	return s.ensureOrchWorkDir().IssueRoot(projectID, issueID)
}

func (s *Service) ensureOrchWorkDir() *infraorch.WorkDir {
	if s.orchWork != nil {
		return s.orchWork
	}
	root := ""
	if s.cfg != nil {
		root = s.cfg.Work.WorkDir
	}
	s.orchWork = infraorch.NewWorkDir(root)
	return s.orchWork
}

func (s *Service) ensureOrchQueue() *infraorch.OrchWorkerQueue {
	if s.orchQueue != nil {
		return s.orchQueue
	}
	s.orchQueue = infraorch.NewOrchWorkerQueue(runClaimBatchSize, 1)
	return s.orchQueue
}
