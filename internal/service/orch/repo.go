package orch

import (
	"context"
	"strings"
	"time"

	"github.com/joomcode/errorx"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
	"github.com/haowen-xu/agent-coder/internal/infra/repo/gitlab"
)

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

// repoAuthToken 获取仓库拉取/推送使用的认证 token。
func (s *Service) repoAuthToken(ctx context.Context, project db.Project) (string, error) {
	if project.ProjectToken != nil {
		if token := strings.TrimSpace(*project.ProjectToken); token != "" {
			return token, nil
		}
	}

	ref := strings.TrimSpace(project.CredentialRef)
	if ref == "" {
		return "", nil
	}
	if s.secret == nil {
		return "", errorx.IllegalState.New("secret manager is not configured")
	}
	token, err := s.secret.Get(ctx, ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(token), nil
}

// upsertIssueNote 使用 marker 进行幂等更新，失败时降级追加评论。
func (s *Service) upsertIssueNote(
	ctx context.Context,
	repoClient repocommon.Client,
	project db.Project,
	issueIID int64,
	marker string,
	body string,
) {
	finalBody := withIssueNoteMarker(marker, body)
	if err := repoClient.UpsertIssueNote(ctx, project, issueIID, marker, finalBody); err != nil {
		_ = repoClient.CreateIssueNote(ctx, project, issueIID, finalBody)
	}
}

// withIssueNoteMarker 执行相关逻辑。
func withIssueNoteMarker(marker string, body string) string {
	marker = strings.TrimSpace(marker)
	body = strings.TrimSpace(body)
	if marker == "" || strings.Contains(body, marker) {
		return body
	}
	if body == "" {
		return marker
	}
	return marker + "\n" + body
}
