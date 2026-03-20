package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// Client 表示数据结构定义。
type Client struct{}

// NewClient 执行相关逻辑。
func NewClient() *Client {
	return &Client{}
}

// EnsureProjectRepo 是 *Client 的方法实现。
func (c *Client) EnsureProjectRepo(ctx context.Context, repoRoot string, repoURL string, projectKey string) (string, error) {
	projectKey = sanitizePathPart(projectKey)
	repoPath := filepath.Join(repoRoot, "_repos", projectKey)
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		if _, runErr := c.run(ctx, repoPath, "fetch", "--all", "--prune"); runErr != nil {
			return "", runErr
		}
		return repoPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		return "", xerr.Infra.Wrap(err, "mkdir repo root")
	}
	if _, err := c.run(ctx, "", "clone", repoURL, repoPath); err != nil {
		return "", err
	}
	return repoPath, nil
}

// EnsureIssueWorktree 是 *Client 的方法实现。
func (c *Client) EnsureIssueWorktree(
	ctx context.Context,
	repoPath string,
	worktreePath string,
	branch string,
	defaultBranch string,
) error {
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return xerr.Infra.Wrap(err, "mkdir worktree parent")
	}
	_ = os.RemoveAll(worktreePath)

	if _, err := c.run(ctx, repoPath, "fetch", "origin", "--prune"); err != nil {
		return err
	}
	baseRef := "origin/" + defaultBranch
	if ok, _ := c.remoteBranchExists(ctx, repoPath, branch); ok {
		baseRef = "origin/" + branch
	}

	if _, err := c.run(ctx, repoPath, "worktree", "add", "--force", "--checkout", "-B", branch, worktreePath, baseRef); err != nil {
		return err
	}
	if _, err := c.run(ctx, worktreePath, "config", "user.email", "agent-coder@local"); err != nil {
		return err
	}
	if _, err := c.run(ctx, worktreePath, "config", "user.name", "agent-coder"); err != nil {
		return err
	}
	return nil
}

// TryMergeDefault 是 *Client 的方法实现。
func (c *Client) TryMergeDefault(ctx context.Context, worktreePath string, defaultBranch string) (bool, string, error) {
	if _, err := c.run(ctx, worktreePath, "fetch", "origin", "--prune"); err != nil {
		return false, "", err
	}
	out, err := c.run(ctx, worktreePath, "merge", "--no-ff", "--no-edit", "origin/"+defaultBranch)
	if err != nil {
		if strings.Contains(strings.ToLower(out), "conflict") {
			return true, out, nil
		}
		return false, out, err
	}
	return false, out, nil
}

// HasChanges 是 *Client 的方法实现。
func (c *Client) HasChanges(ctx context.Context, worktreePath string) (bool, error) {
	out, err := c.run(ctx, worktreePath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// CommitAll 是 *Client 的方法实现。
func (c *Client) CommitAll(ctx context.Context, worktreePath string, message string) error {
	if _, err := c.run(ctx, worktreePath, "add", "-A"); err != nil {
		return err
	}
	_, err := c.run(ctx, worktreePath, "commit", "-m", message)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "nothing to commit") {
			return nil
		}
		return err
	}
	return nil
}

// PushBranch 是 *Client 的方法实现。
func (c *Client) PushBranch(ctx context.Context, worktreePath string, branch string) error {
	_, err := c.run(ctx, worktreePath, "push", "-u", "origin", branch)
	return err
}

// run 是 *Client 的方法实现。
func (c *Client) run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		out := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
		if out == "" {
			out = err.Error()
		}
		return out, xerr.Infra.New("git %s failed: %s", strings.Join(args, " "), out)
	}
	return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), nil
}

// remoteBranchExists 是 *Client 的方法实现。
func (c *Client) remoteBranchExists(ctx context.Context, repoPath string, branch string) (bool, error) {
	out, err := c.run(ctx, repoPath, "ls-remote", "--heads", "origin", branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// sanitizePathPart 执行相关逻辑。
func sanitizePathPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "default"
	}
	replaced := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(v)
	return fmt.Sprintf("%s", replaced)
}
