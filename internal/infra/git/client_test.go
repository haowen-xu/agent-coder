package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSanitizePathPart 用于单元测试。
func TestSanitizePathPart(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "default"},
		{in: "a/b c", want: "a_b_c"},
		{in: "a\\b", want: "a_b"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			if got := sanitizePathPart(tc.in); got != tc.want {
				t.Fatalf("sanitizePathPart(%q)=%q want=%q", tc.in, got, tc.want)
			}
		})
	}
}

// TestMaskSensitiveURL 用于单元测试。
func TestMaskSensitiveURL(t *testing.T) {
	t.Parallel()

	in := "clone https://oauth2:mytoken@example.com/group/repo.git"
	out := maskSensitiveURL(in)

	if strings.Contains(out, "mytoken") {
		t.Fatalf("token should be masked: %q", out)
	}
	if !strings.Contains(out, "oauth2:%2A%2A%2A%2A@") {
		t.Fatalf("masked output should contain masked password: %q", out)
	}
}

// TestMaskSensitiveAuthHeader 用于单元测试。
func TestMaskSensitiveAuthHeader(t *testing.T) {
	t.Parallel()

	in := "git -c http.extraHeader=Authorization: Basic abc123 push"
	out := maskSensitive(in)
	if strings.Contains(out, "abc123") {
		t.Fatalf("header token should be masked: %q", out)
	}
	if !strings.Contains(strings.ToLower(out), "authorization: basic ****") {
		t.Fatalf("expected masked auth header, got: %q", out)
	}
}

// TestRunWithTokenErrorMessageMasked 用于单元测试。
func TestRunWithTokenErrorMessageMasked(t *testing.T) {
	t.Parallel()

	c := NewClient()
	_, err := c.runWithToken(context.Background(), "", "top-secret-token", "this-subcommand-does-not-exist")
	if err == nil {
		t.Fatalf("expected git command to fail")
	}
	msg := err.Error()
	if strings.Contains(msg, "top-secret-token") {
		t.Fatalf("error message leaks raw token: %q", msg)
	}
}

// TestGitClientWorkflow 用于单元测试。
func TestGitClientWorkflow(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	seed := filepath.Join(root, "seed")
	updater := filepath.Join(root, "updater")

	gitCmd(t, "", "init", "--bare", origin)
	gitCmd(t, "", "clone", origin, seed)
	gitCmd(t, seed, "config", "user.email", "tester@example.com")
	gitCmd(t, seed, "config", "user.name", "tester")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write README failed: %v", err)
	}
	gitCmd(t, seed, "add", "README.md")
	gitCmd(t, seed, "commit", "-m", "init")
	gitCmd(t, seed, "push", "-u", "origin", "HEAD:main")

	client := NewClient()
	repoRoot := filepath.Join(root, "repos")
	repoPath, err := client.EnsureProjectRepo(ctx, repoRoot, origin, "project/a", "")
	if err != nil {
		t.Fatalf("EnsureProjectRepo clone failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		t.Fatalf("repo .git should exist: %v", err)
	}
	if _, err := client.EnsureProjectRepo(ctx, repoRoot, origin, "project/a", ""); err != nil {
		t.Fatalf("EnsureProjectRepo fetch failed: %v", err)
	}

	worktree := filepath.Join(root, "wt")
	if err := client.EnsureIssueWorktree(ctx, repoPath, worktree, "feature-1", "main", ""); err != nil {
		t.Fatalf("EnsureIssueWorktree failed: %v", err)
	}
	hasChanges, err := client.HasChanges(ctx, worktree)
	if err != nil || hasChanges {
		t.Fatalf("new worktree should be clean: changed=%v err=%v", hasChanges, err)
	}

	if err := os.WriteFile(filepath.Join(worktree, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("write feature file failed: %v", err)
	}
	hasChanges, _ = client.HasChanges(ctx, worktree)
	if !hasChanges {
		t.Fatalf("worktree should detect local changes")
	}
	if err := client.CommitAll(ctx, worktree, "feature commit"); err != nil {
		t.Fatalf("CommitAll failed: %v", err)
	}
	if err := client.CommitAll(ctx, worktree, "noop commit"); err != nil {
		t.Fatalf("CommitAll should ignore nothing-to-commit: %v", err)
	}
	if err := client.PushBranch(ctx, worktree, "feature-1", ""); err != nil {
		t.Fatalf("PushBranch failed: %v", err)
	}

	if ok, err := client.remoteBranchExists(ctx, repoPath, "feature-1", ""); err != nil || !ok {
		t.Fatalf("remote branch should exist: ok=%v err=%v", ok, err)
	}
	if ok, _ := client.localRefExists(ctx, repoPath, "refs/remotes/origin/main"); !ok {
		t.Fatalf("local origin/main ref should exist")
	}

	gitCmd(t, "", "clone", origin, updater)
	gitCmd(t, updater, "checkout", "main")
	gitCmd(t, updater, "config", "user.email", "tester@example.com")
	gitCmd(t, updater, "config", "user.name", "tester")
	if err := os.WriteFile(filepath.Join(updater, "main.txt"), []byte("main update\n"), 0o644); err != nil {
		t.Fatalf("write main file failed: %v", err)
	}
	gitCmd(t, updater, "add", "main.txt")
	gitCmd(t, updater, "commit", "-m", "main update")
	gitCmd(t, updater, "push", "origin", "main")

	conflict, out, err := client.TryMergeDefault(ctx, worktree, "main", "")
	if err != nil {
		t.Fatalf("TryMergeDefault failed: %v out=%s", err, out)
	}
	if conflict {
		t.Fatalf("expected clean merge without conflict")
	}

}

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}
