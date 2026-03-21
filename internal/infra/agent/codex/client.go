package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/haowen-xu/agent-coder/internal/infra/agent/base"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

var resultJSONRE = regexp.MustCompile("(?s)```RESULT_JSON\\s*(\\{.*?\\})\\s*```")

// Client 表示数据结构定义。
type Client struct {
	binary  string // binary 字段说明。
	sandbox bool   // sandbox 字段说明。
}

// NewClient 执行相关逻辑。
func NewClient(binary string, sandbox ...bool) *Client {
	bin := strings.TrimSpace(binary)
	if bin == "" {
		bin = "codex"
	}
	useSandbox := true
	if len(sandbox) > 0 {
		useSandbox = sandbox[0]
	}
	return &Client{
		binary:  bin,
		sandbox: useSandbox,
	}
}

// Name 是方法实现。
func (c *Client) Name() string {
	return "codex"
}

// Run 是方法实现。
func (c *Client) Run(ctx context.Context, req base.InvokeRequest) (*base.InvokeResult, error) {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Hour
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Best-effort codex invocation. If unavailable, return a deterministic fallback decision
	// so the worker loop can continue in degraded mode.
	useSandbox := req.UseSandbox

	args := []string{"exec", "--skip-git-repo-check"}
	if useSandbox {
		args = append(args, "--full-auto", "--sandbox", "workspace-write")
	} else {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}
	args = append(args, req.Prompt)
	cmd := exec.CommandContext(runCtx, c.binary, args...)
	cmd.Dir = req.WorkDir
	if len(req.Env) > 0 {
		env := make([]string, 0, len(req.Env))
		for k, v := range req.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = append(cmd.Environ(), env...)
	}

	stdout, err := cmd.Output()
	stderr := ""
	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr = string(ee.Stderr)
			exitCode = exitStatus(ee)
		} else {
			stderr = err.Error()
			exitCode = 1
		}
	}

	out := string(stdout)
	decision := parseDecision(out)
	if err != nil && decision.Decision != "blocked" {
		decision.Decision = "blocked"
		decision.Role = strings.TrimSpace(req.Role)
		if strings.TrimSpace(decision.Summary) == "" {
			decision.Summary = "codex execution failed"
		}
		if strings.TrimSpace(decision.BlockingReason) == "" {
			decision.BlockingReason = fallbackReason(stderr, out)
		}
	}
	if decision.Decision == "" {
		decision = fallbackDecision(req.Role, fallbackReason(stderr, out))
	}

	result := &base.InvokeResult{
		ExitCode:    exitCode,
		Stdout:      out,
		Stderr:      stderr,
		LastMessage: decision.Summary,
		Decision:    decision,
	}
	if err != nil && decision.Decision == "blocked" {
		return result, xerr.Infra.Wrap(err, "run codex command")
	}
	return result, nil
}

// parseDecision 执行相关逻辑。
func parseDecision(out string) base.Decision {
	match := resultJSONRE.FindStringSubmatch(out)
	if len(match) < 2 {
		return base.Decision{}
	}

	var row base.Decision
	if err := json.Unmarshal([]byte(match[1]), &row); err != nil {
		return base.Decision{}
	}
	return row
}

// fallbackDecision 执行相关逻辑。
func fallbackDecision(role string, reason string) base.Decision {
	role = strings.TrimSpace(role)
	return base.Decision{
		Role:           role,
		Decision:       "blocked",
		Summary:        "codex result parse failed",
		BlockingReason: strings.TrimSpace(reason),
	}
}

// fallbackReason 执行相关逻辑。
func fallbackReason(stderr string, stdout string) string {
	reason := strings.TrimSpace(stderr)
	if reason != "" {
		return reason
	}
	if strings.Contains(stdout, "RESULT_JSON") {
		return "failed to parse RESULT_JSON"
	}
	return "missing RESULT_JSON in codex output"
}

// exitStatus 执行相关逻辑。
func exitStatus(err *exec.ExitError) int {
	if err == nil {
		return 0
	}
	if status, ok := err.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus()
	}
	return 1
}
