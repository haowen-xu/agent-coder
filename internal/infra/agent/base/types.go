package base

import "time"

type TestResult struct {
	Cmd    string `json:"cmd"`
	OK     bool   `json:"ok"`
	Output string `json:"output"`
}

type Decision struct {
	Role           string       `json:"role"`
	Decision       string       `json:"decision"`
	Summary        string       `json:"summary"`
	ChangedFiles   []string     `json:"changed_files"`
	Tests          []TestResult `json:"tests"`
	NextAction     string       `json:"next_action"`
	BlockingReason string       `json:"blocking_reason"`
}

type InvokeRequest struct {
	RunKind string
	Role    string
	Prompt  string
	WorkDir string
	RunDir  string
	Timeout time.Duration
	Env     map[string]string
}

type InvokeResult struct {
	ExitCode    int
	Stdout      string
	Stderr      string
	ThreadID    string
	LastMessage string
	Decision    Decision
}
