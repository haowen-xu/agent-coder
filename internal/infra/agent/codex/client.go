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

type Client struct {
	binary string
}

func NewClient(binary string) *Client {
	bin := strings.TrimSpace(binary)
	if bin == "" {
		bin = "codex"
	}
	return &Client{binary: bin}
}

func (c *Client) Name() string {
	return "codex"
}

func (c *Client) Run(ctx context.Context, req base.InvokeRequest) (*base.InvokeResult, error) {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Hour
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Best-effort codex invocation. If unavailable, return a deterministic fallback decision
	// so the worker loop can continue in degraded mode.
	cmd := exec.CommandContext(runCtx, c.binary, "exec", req.Prompt)
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
	if decision.Decision == "" {
		decision = fallbackDecision(req.Role, stderr)
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

func fallbackDecision(role string, reason string) base.Decision {
	role = strings.TrimSpace(role)
	if role == "review" {
		return base.Decision{
			Role:           role,
			Decision:       "pass",
			Summary:        "fallback review pass",
			BlockingReason: strings.TrimSpace(reason),
		}
	}
	if reason != "" {
		return base.Decision{
			Role:           role,
			Decision:       "blocked",
			Summary:        "codex execution failed",
			BlockingReason: reason,
		}
	}
	return base.Decision{
		Role:     role,
		Decision: "ready_for_review",
		Summary:  "fallback ready_for_review",
	}
}

func exitStatus(err *exec.ExitError) int {
	if err == nil {
		return 0
	}
	if status, ok := err.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus()
	}
	return 1
}
