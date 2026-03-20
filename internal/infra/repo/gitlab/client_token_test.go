package gitlab

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

func TestClient_ListIssues_UsesProjectTokenFirst(t *testing.T) {
	t.Parallel()

	const expectedToken = "project-token-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("PRIVATE-TOKEN"); got != expectedToken {
			t.Fatalf("expected PRIVATE-TOKEN %q, got %q", expectedToken, got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	projectID := "365"
	projectToken := expectedToken
	project := db.Project{
		ProviderURL:    server.URL,
		IssueProjectID: &projectID,
		ProjectSlug:    "ai-agents/agent-coder-testbed",
		CredentialRef:  "ignored_ref",
		ProjectToken:   &projectToken,
	}

	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 10*time.Second, nil)
	issues, err := client.ListIssues(context.Background(), project, repocommon.ListIssuesOptions{
		State:    "all",
		PerPage:  20,
		MaxPages: 1,
	})
	if err != nil {
		t.Fatalf("ListIssues() error = %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected empty issues, got %d", len(issues))
	}
}
