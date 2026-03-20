//go:build e2e

package gitlab

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/issuetracker"
)

const (
	e2eGitLabAPIBase   = "https://git.ccf-quant.com/api/v4"
	e2eGitLabProjectID = "365"
	e2eGitLabProject   = "ai-agents/agent-coder-testbed"
	e2eGitLabToken     = "glpat-p2hVy2Z6AyoMGjAJwbyeXG86MQp1OjFrCA.01.0y0sz78gm"
)

func TestClient_ListIssues_E2E(t *testing.T) {
	t.Parallel()

	token := e2eGitLabToken
	projectID := e2eGitLabProjectID
	project := db.Project{
		Provider:       db.ProviderGitLab,
		ProviderURL:    e2eGitLabAPIBase,
		ProjectSlug:    e2eGitLabProject,
		IssueProjectID: &projectID,
		ProjectToken:   &token,
	}

	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Second, nil)
	issues, err := client.ListIssues(context.Background(), project, issuetracker.ListIssuesOptions{
		State:    "all",
		PerPage:  50,
		MaxPages: 1,
	})
	if err != nil {
		t.Fatalf("ListIssues() e2e error = %v", err)
	}
	for i, issue := range issues {
		if issue.IID <= 0 {
			t.Fatalf("issue[%d] invalid IID: %d", i, issue.IID)
		}
	}
}
