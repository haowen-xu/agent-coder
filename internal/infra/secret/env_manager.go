package secret

import (
	"context"
	"os"
	"regexp"
	"strings"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type EnvManager struct {
	prefix string
}

func NewEnvManager(prefix string) *EnvManager {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "AGENT_CODER_SECRET_"
	}
	return &EnvManager{prefix: p}
}

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

func sanitizeEnvKey(in string) string {
	up := strings.ToUpper(strings.TrimSpace(in))
	re := regexp.MustCompile(`[^A-Z0-9]+`)
	return re.ReplaceAllString(up, "_")
}
