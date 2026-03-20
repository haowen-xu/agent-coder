package secret

import (
	"context"
	"os"
	"regexp"
	"strings"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// EnvManager 表示数据结构定义。
type EnvManager struct {
	prefix string // prefix 字段说明。
}

// NewEnvManager 执行相关逻辑。
func NewEnvManager(prefix string) *EnvManager {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "AGENT_CODER_SECRET_"
	}
	return &EnvManager{prefix: p}
}

// Get 是 *EnvManager 的方法实现。
func (m *EnvManager) Get(_ context.Context, ref string) (string, error) {
	keyRef := strings.TrimSpace(ref)
	if keyRef == "" {
		return "", xerr.Config.New("secret ref is required")
	}
	key := m.prefix + sanitizeEnvKey(keyRef)
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return "", xerr.Config.New("secret not found for ref=%s env=%s", ref, key)
	}
	return val, nil
}

// sanitizeEnvKey 执行相关逻辑。
func sanitizeEnvKey(in string) string {
	up := strings.ToUpper(strings.TrimSpace(in))
	re := regexp.MustCompile(`[^A-Z0-9]+`)
	return re.ReplaceAllString(up, "_")
}
