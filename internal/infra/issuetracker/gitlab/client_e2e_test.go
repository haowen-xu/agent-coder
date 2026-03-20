//go:build e2e

package gitlab

import (
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestClient_IssueTrackerCommon_E2E(t *testing.T) {
	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 30*time.Second, nil)
	runIssueTrackerCommonTests(t, client)
}
