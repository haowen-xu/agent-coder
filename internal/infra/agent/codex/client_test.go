package codex

import "testing"

// TestFallbackDecision_AlwaysBlocked 用于单元测试。
func TestFallbackDecision_AlwaysBlocked(t *testing.T) {
	t.Parallel()

	got := fallbackDecision("review", "missing RESULT_JSON in codex output")
	if got.Decision != "blocked" {
		t.Fatalf("expected blocked decision, got %q", got.Decision)
	}
	if got.Role != "review" {
		t.Fatalf("expected role review, got %q", got.Role)
	}
}

// TestFallbackReason_PreferStderr 用于单元测试。
func TestFallbackReason_PreferStderr(t *testing.T) {
	t.Parallel()

	got := fallbackReason("stderr text", "stdout text")
	if got != "stderr text" {
		t.Fatalf("expected stderr priority, got %q", got)
	}
}

// TestFallbackReason_ParseFailure 用于单元测试。
func TestFallbackReason_ParseFailure(t *testing.T) {
	t.Parallel()

	got := fallbackReason("", "```RESULT_JSON\ninvalid\n```")
	if got != "failed to parse RESULT_JSON" {
		t.Fatalf("unexpected reason: %q", got)
	}
}
