package orch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/haowen-xu/agent-coder/internal/utils"
)

const (
	// StateFileName 是 run 级状态文件名。
	StateFileName = ".orch-state.json"
)

// State 描述 orch 运行状态快照。
type State struct {
	Kind       AgentKind `json:"kind"`
	Status     string    `json:"status"`
	ProjectKey string    `json:"project_key"`
	Message    string    `json:"message,omitempty"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// RunPaths 描述 issue/run 的目录结构。
type RunPaths struct {
	IssueRoot string
	GitTree   string
	RunDir    string
	StateFile string
}

// WorkDir 负责管理 git-tree 与状态目录。
type WorkDir struct {
	root string
}

// NewWorkDir 创建 WorkDir。
func NewWorkDir(root string) *WorkDir {
	return &WorkDir{root: strings.TrimSpace(root)}
}

// Root 是 *WorkDir 的方法实现。
func (w *WorkDir) Root() string { return w.root }

// IssueRoot 是 *WorkDir 的方法实现。
func (w *WorkDir) IssueRoot(projectID uint, issueID uint) string {
	return filepath.Join(w.root, fmt.Sprintf("%d", projectID), fmt.Sprintf("%d", issueID))
}

// GitTree 是 *WorkDir 的方法实现。
func (w *WorkDir) GitTree(projectID uint, issueID uint) string {
	return filepath.Join(w.IssueRoot(projectID, issueID), "git-tree")
}

// RunDir 是 *WorkDir 的方法实现。
func (w *WorkDir) RunDir(projectID uint, issueID uint, runNo int) string {
	return filepath.Join(w.IssueRoot(projectID, issueID), "agent", "runs", fmt.Sprintf("%d", runNo))
}

// BuildRunPaths 构造 run 路径。
func (w *WorkDir) BuildRunPaths(projectID uint, issueID uint, runNo int) RunPaths {
	runDir := w.RunDir(projectID, issueID, runNo)
	return RunPaths{
		IssueRoot: w.IssueRoot(projectID, issueID),
		GitTree:   w.GitTree(projectID, issueID),
		RunDir:    runDir,
		StateFile: filepath.Join(runDir, StateFileName),
	}
}

// EnsureRunPaths 保证 issue/run 目录存在。
func (w *WorkDir) EnsureRunPaths(projectID uint, issueID uint, runNo int) (RunPaths, error) {
	if strings.TrimSpace(w.root) == "" {
		return RunPaths{}, fmt.Errorf("workdir root is required")
	}
	paths := w.BuildRunPaths(projectID, issueID, runNo)
	if err := os.MkdirAll(paths.GitTree, 0o755); err != nil {
		return RunPaths{}, err
	}
	if err := os.MkdirAll(paths.RunDir, 0o755); err != nil {
		return RunPaths{}, err
	}
	return paths, nil
}

// EnsureFromInvoke 按 invoke 请求中的目录信息创建目录。
func (w *WorkDir) EnsureFromInvoke(workDir string, runDir string) error {
	if strings.TrimSpace(workDir) != "" {
		if err := os.MkdirAll(workDir, 0o755); err != nil {
			return err
		}
	}
	if strings.TrimSpace(runDir) != "" {
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// StateFilePath 返回 runDir 对应状态文件路径。
func (w *WorkDir) StateFilePath(runDir string) string {
	return filepath.Join(strings.TrimSpace(runDir), StateFileName)
}

// WriteState 写入 run 级状态。
func (w *WorkDir) WriteState(runDir string, state State) error {
	runDir = strings.TrimSpace(runDir)
	if runDir == "" {
		return fmt.Errorf("runDir is required")
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = utils.NowUTC()
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(w.StateFilePath(runDir), raw, 0o644)
}

// ReadState 读取 run 级状态。
func (w *WorkDir) ReadState(runDir string) (State, error) {
	raw, err := os.ReadFile(w.StateFilePath(runDir))
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(raw, &state); err != nil {
		return State{}, err
	}
	return state, nil
}
