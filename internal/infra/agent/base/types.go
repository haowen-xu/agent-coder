package base

import "time"

// TestResult 表示数据结构定义。
type TestResult struct {
	Cmd    string `json:"cmd"`    // Cmd 字段说明。
	OK     bool   `json:"ok"`     // OK 字段说明。
	Output string `json:"output"` // Output 字段说明。
}

// Decision 表示数据结构定义。
type Decision struct {
	Role           string       `json:"role"`            // Role 字段说明。
	Decision       string       `json:"decision"`        // Decision 字段说明。
	Summary        string       `json:"summary"`         // Summary 字段说明。
	ChangedFiles   []string     `json:"changed_files"`   // ChangedFiles 字段说明。
	Tests          []TestResult `json:"tests"`           // Tests 字段说明。
	NextAction     string       `json:"next_action"`     // NextAction 字段说明。
	BlockingReason string       `json:"blocking_reason"` // BlockingReason 字段说明。
}

// InvokeRequest 表示数据结构定义。
type InvokeRequest struct {
	RunKind    string            // RunKind 字段说明。
	Role       string            // Role 字段说明。
	Prompt     string            // Prompt 字段说明。
	WorkDir    string            // WorkDir 字段说明。
	RunDir     string            // RunDir 字段说明。
	Timeout    time.Duration     // Timeout 字段说明。
	Env        map[string]string // Env 字段说明。
	UseSandbox bool              // UseSandbox 字段说明。
}

// InvokeResult 表示数据结构定义。
type InvokeResult struct {
	ExitCode    int      // ExitCode 字段说明。
	Stdout      string   // Stdout 字段说明。
	Stderr      string   // Stderr 字段说明。
	ThreadID    string   // ThreadID 字段说明。
	LastMessage string   // LastMessage 字段说明。
	Decision    Decision // Decision 字段说明。
}
