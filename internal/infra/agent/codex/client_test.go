package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haowen-xu/agent-coder/internal/infra/agent/base"
)

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

// TestParseDecision 用于单元测试。
func TestParseDecision(t *testing.T) {
	t.Parallel()

	out := "prefix\n```RESULT_JSON\n{\"role\":\"review\",\"decision\":\"pass\",\"summary\":\"ok\"}\n```\n"
	row := parseDecision(out)
	if row.Role != "review" || row.Decision != "pass" || row.Summary != "ok" {
		t.Fatalf("parseDecision mismatch: %#v", row)
	}
}

// TestNewClientAndName 用于单元测试。
func TestNewClientAndName(t *testing.T) {
	t.Parallel()

	c1 := NewClient("")
	if c1.binary != "codex" {
		t.Fatalf("default binary mismatch: %s", c1.binary)
	}
	if c1.Name() != "codex" {
		t.Fatalf("name mismatch: %s", c1.Name())
	}
	c2 := NewClient("my-codex", false)
	if c2.binary != "my-codex" {
		t.Fatalf("custom binary mismatch: %s", c2.binary)
	}
}

// TestRunSuccessAndFailure 用于单元测试。
func TestRunSuccessAndFailure(t *testing.T) {
	ctx := context.Background()
	workdir := t.TempDir()
	bin := filepath.Join(workdir, "fake-codex.sh")

	script := "#!/bin/sh\n" +
		"if [ \"$FAKE_MODE\" = \"ok\" ]; then\n" +
		"  cat <<'EOF'\n" +
		"```RESULT_JSON\n" +
		"{\"role\":\"dev\",\"decision\":\"ready_for_review\",\"summary\":\"done\"}\n" +
		"```\n" +
		"EOF\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo \"boom\" 1>&2\n" +
		"exit 2\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex script failed: %v", err)
	}

	client := NewClient(bin)
	okRes, err := client.Run(ctx, base.InvokeRequest{
		Role:    "dev",
		Prompt:  "do work",
		WorkDir: workdir,
		Timeout: 5 * time.Second,
		Env: map[string]string{
			"FAKE_MODE": "ok",
		},
	})
	if err != nil {
		t.Fatalf("Run success path should not error: %v", err)
	}
	if okRes.Decision.Decision != "ready_for_review" {
		t.Fatalf("unexpected success decision: %#v", okRes.Decision)
	}

	failRes, err := client.Run(ctx, base.InvokeRequest{
		Role:    "review",
		Prompt:  "review",
		WorkDir: workdir,
		Timeout: 5 * time.Second,
		Env: map[string]string{
			"FAKE_MODE": "fail",
		},
	})
	if err == nil {
		t.Fatalf("Run failure path should return error")
	}
	if failRes == nil || failRes.Decision.Decision != "blocked" {
		t.Fatalf("failure decision should be blocked: %#v", failRes)
	}
}
