package prompts

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/joomcode/errorx"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// Key 表示数据结构定义。
type Key struct {
	RunKind   string `json:"run_kind"`   // RunKind 字段说明。
	AgentRole string `json:"agent_role"` // AgentRole 字段说明。
}

// Template 表示数据结构定义。
type Template struct {
	ProjectKey string `json:"project_key,omitempty"` // ProjectKey 字段说明。
	RunKind    string `json:"run_kind"`              // RunKind 字段说明。
	AgentRole  string `json:"agent_role"`            // AgentRole 字段说明。
	Source     string `json:"source"`                // Source 字段说明。
	Content    string `json:"content"`               // Content 字段说明。
}

var (
	//go:embed defaults/*.md
	// TEST 注释。

	// TEST 注释。

	defaultPromptFS embed.FS

	defaultPromptFiles = map[string]string{
		"dev:dev":      "defaults/dev.dev.md",
		"dev:review":   "defaults/dev.review.md",
		"merge:merge":  "defaults/merge.merge.md",
		"merge:review": "defaults/merge.review.md",
	}

	orderedKeys = []Key{
		{RunKind: "dev", AgentRole: "dev"},
		{RunKind: "dev", AgentRole: "review"},
		{RunKind: "merge", AgentRole: "merge"},
		{RunKind: "merge", AgentRole: "review"},
	}
)

// OrderedKeys 执行相关逻辑。
func OrderedKeys() []Key {
	keys := make([]Key, 0, len(orderedKeys))
	keys = append(keys, orderedKeys...)
	return keys
}

// ValidateKey 执行相关逻辑。
func ValidateKey(runKind string, agentRole string) error {
	key := keyID(runKind, agentRole)
	if _, ok := defaultPromptFiles[key]; !ok {
		return errorx.IllegalArgument.New("unsupported run_kind/agent_role: %s/%s", runKind, agentRole)
	}
	return nil
}

// DefaultTemplate 执行相关逻辑。
func DefaultTemplate(runKind string, agentRole string) (string, error) {
	if err := ValidateKey(runKind, agentRole); err != nil {
		return "", err
	}

	path := defaultPromptFiles[keyID(runKind, agentRole)]
	data, err := defaultPromptFS.ReadFile(path)
	if err != nil {
		return "", xerr.Infra.Wrap(err, "read embedded default prompt: %s", filepath.Base(path))
	}
	return string(data), nil
}

// ListDefaultTemplates 执行相关逻辑。
func ListDefaultTemplates() ([]Template, error) {
	templates := make([]Template, 0, len(orderedKeys))
	for _, k := range orderedKeys {
		content, err := DefaultTemplate(k.RunKind, k.AgentRole)
		if err != nil {
			return nil, err
		}
		templates = append(templates, Template{
			RunKind:   k.RunKind,
			AgentRole: k.AgentRole,
			Source:    "embedded_default",
			Content:   content,
		})
	}
	return templates, nil
}

// keyID 执行相关逻辑。
func keyID(runKind string, agentRole string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(runKind), strings.TrimSpace(agentRole))
}
