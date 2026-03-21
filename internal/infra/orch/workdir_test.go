package orch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestWorkDirPathsAndEnsure 验证 git-tree 与 run 目录生成和创建。
func TestWorkDirPathsAndEnsure(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	wd := NewWorkDir(root)
	paths, err := wd.EnsureRunPaths(7, 9, 3)
	if err != nil {
		t.Fatalf("EnsureRunPaths failed: %v", err)
	}
	if !strings.Contains(paths.GitTree, filepath.Join("7", "9", "git-tree")) {
		t.Fatalf("unexpected git-tree path: %s", paths.GitTree)
	}
	if !strings.Contains(paths.RunDir, filepath.Join("7", "9", "agent", "runs", "3")) {
		t.Fatalf("unexpected run dir path: %s", paths.RunDir)
	}
	if st, err := os.Stat(paths.GitTree); err != nil || !st.IsDir() {
		t.Fatalf("git-tree dir missing: %v", err)
	}
	if st, err := os.Stat(paths.RunDir); err != nil || !st.IsDir() {
		t.Fatalf("run dir missing: %v", err)
	}
}

// TestWorkDirStateIO 验证 run 状态读写。
func TestWorkDirStateIO(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	wd := NewWorkDir(root)
	runDir := filepath.Join(root, "x", "run")
	state := State{
		Kind:       AgentKindReview,
		Status:     "running",
		ProjectKey: "p1",
		UpdatedAt:  time.Now().UTC(),
	}
	if err := wd.WriteState(runDir, state); err != nil {
		t.Fatalf("WriteState failed: %v", err)
	}
	got, err := wd.ReadState(runDir)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}
	if got.Kind != state.Kind || got.Status != state.Status || got.ProjectKey != state.ProjectKey {
		t.Fatalf("state mismatch: got=%#v want=%#v", got, state)
	}
}
