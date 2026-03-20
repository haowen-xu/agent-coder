//go:build e2e

package gitlab

import (
	"io"
	"log/slog"
	"testing"
	"time"

	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

func TestClient_IssueTrackerCommon_E2E(t *testing.T) {
	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Second, nil)
	repocommon.RunStandardIssueTrackerTests(t, client)
}
