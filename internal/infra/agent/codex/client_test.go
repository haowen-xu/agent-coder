package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
		Timeout: 15 * time.Second,
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
		Timeout: 15 * time.Second,
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

// TestRunWithFakeCodex_ValidatesInvocationInput 用于单元测试。
func TestRunWithFakeCodex_ValidatesInvocationInput(t *testing.T) {
	ctx := context.Background()
	workdir := t.TempDir()
	bin := filepath.Join(workdir, "fake-codex-input.sh")

	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"printf '%s\\n' \"$@\" > \"$FAKE_ARGS_FILE\"\n" +
		"pwd > \"$FAKE_PWD_FILE\"\n" +
		"printf '%s' \"${FAKE_ENV_VALUE:-}\" > \"$FAKE_ENV_FILE\"\n" +
		"cat <<'EOF'\n" +
		"```RESULT_JSON\n" +
		"{\"role\":\"dev\",\"decision\":\"ready_for_review\",\"summary\":\"ok\"}\n" +
		"```\n" +
		"EOF\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex script failed: %v", err)
	}

	runWithSandbox := func(useSandbox bool) []string {
		argsFile := filepath.Join(workdir, "args-"+map[bool]string{true: "sandbox", false: "nosandbox"}[useSandbox]+".txt")
		pwdFile := filepath.Join(workdir, "pwd-"+map[bool]string{true: "sandbox", false: "nosandbox"}[useSandbox]+".txt")
		envFile := filepath.Join(workdir, "env-"+map[bool]string{true: "sandbox", false: "nosandbox"}[useSandbox]+".txt")
		client := NewClient(bin)
		res, err := client.Run(ctx, base.InvokeRequest{
			Role:       "dev",
			Prompt:     "PROMPT_FOR_FAKE_CODEX",
			WorkDir:    workdir,
			UseSandbox: useSandbox,
			Timeout:    10 * time.Second,
			Env: map[string]string{
				"FAKE_ARGS_FILE": argsFile,
				"FAKE_PWD_FILE":  pwdFile,
				"FAKE_ENV_FILE":  envFile,
				"FAKE_ENV_VALUE": "ENV_OK",
			},
		})
		if err != nil {
			t.Fatalf("fake codex run failed (sandbox=%v): %v", useSandbox, err)
		}
		if res == nil || res.Decision.Decision != "ready_for_review" {
			t.Fatalf("unexpected invoke result (sandbox=%v): %#v", useSandbox, res)
		}

		pwd, err := os.ReadFile(pwdFile)
		if err != nil {
			t.Fatalf("read pwd file failed: %v", err)
		}
		if strings.TrimSpace(string(pwd)) != workdir {
			t.Fatalf("workdir mismatch: got=%q want=%q", strings.TrimSpace(string(pwd)), workdir)
		}
		envVal, err := os.ReadFile(envFile)
		if err != nil {
			t.Fatalf("read env file failed: %v", err)
		}
		if string(envVal) != "ENV_OK" {
			t.Fatalf("env propagation mismatch: got=%q", string(envVal))
		}

		rawArgs, err := os.ReadFile(argsFile)
		if err != nil {
			t.Fatalf("read args file failed: %v", err)
		}
		args := strings.Split(strings.TrimSpace(string(rawArgs)), "\n")
		if len(args) == 0 {
			t.Fatalf("fake codex args should not be empty")
		}
		if args[0] != "exec" {
			t.Fatalf("first arg should be exec, got=%q", args[0])
		}
		if args[len(args)-1] != "PROMPT_FOR_FAKE_CODEX" {
			t.Fatalf("prompt should be passed as last arg, got=%q", args[len(args)-1])
		}
		return args
	}

	sandboxArgs := runWithSandbox(true)
	if !containsArg(sandboxArgs, "--full-auto") || !containsArg(sandboxArgs, "--sandbox") || !containsArg(sandboxArgs, "workspace-write") {
		t.Fatalf("sandbox args missing: %v", sandboxArgs)
	}
	if containsArg(sandboxArgs, "--dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("sandbox run should not include bypass approvals flag: %v", sandboxArgs)
	}

	noSandboxArgs := runWithSandbox(false)
	if !containsArg(noSandboxArgs, "--dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("no-sandbox run should include bypass flag: %v", noSandboxArgs)
	}
}

func containsArg(args []string, target string) bool {
	for _, it := range args {
		if it == target {
			return true
		}
	}
	return false
}
