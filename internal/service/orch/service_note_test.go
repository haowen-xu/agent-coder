package orch

import (
	"strings"
	"testing"

	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

// TestBuildMRReadyNote_WithMRURL 用于单元测试。
func TestBuildMRReadyNote_WithMRURL(t *testing.T) {
	t.Parallel()

	note := buildMRReadyNote(42, "agent-coder/issue-42", "main", &repocommon.MergeRequest{
		IID:          128,
		WebURL:       "https://gitlab.example.com/group/repo/-/merge_requests/128",
		SourceBranch: "agent-coder/issue-42",
		TargetBranch: "main",
	})

	mustContain(t, note, "Agent run completed. MR is ready for human review.")
	mustContain(t, note, "- Issue: #42")
	mustContain(t, note, "[!128](https://gitlab.example.com/group/repo/-/merge_requests/128)")
	mustContain(t, note, "- Source Branch: `agent-coder/issue-42`")
	mustContain(t, note, "- Target Branch: `main`")
}

// TestBuildMRReadyNote_NoMR 用于单元测试。
func TestBuildMRReadyNote_NoMR(t *testing.T) {
	t.Parallel()

	note := buildMRReadyNote(7, "branch-a", "main", nil)
	if note != "MR is ready for human review." {
		t.Fatalf("unexpected fallback note: %q", note)
	}
}

// mustContain 执行相关逻辑。
func mustContain(t *testing.T, text string, expected string) {
	t.Helper()
	if !strings.Contains(text, expected) {
		t.Fatalf("expected %q in note, got:\n%s", expected, text)
	}
}
