package prompts

import (
	"strings"
	"testing"
)

// TestPromptDefaults 用于单元测试。
func TestPromptDefaults(t *testing.T) {
	t.Parallel()

	keys := OrderedKeys()
	if len(keys) == 0 {
		t.Fatalf("OrderedKeys should not be empty")
	}

	keys[0].RunKind = "mutated"
	keysAgain := OrderedKeys()
	if keysAgain[0].RunKind == "mutated" {
		t.Fatalf("OrderedKeys should return a copy")
	}

	if err := ValidateKey("dev", "dev"); err != nil {
		t.Fatalf("ValidateKey(dev,dev) should pass: %v", err)
	}
	if err := ValidateKey("unknown", "dev"); err == nil {
		t.Fatalf("ValidateKey should fail for unsupported key")
	}

	content, err := DefaultTemplate("dev", "dev")
	if err != nil || strings.TrimSpace(content) == "" {
		t.Fatalf("DefaultTemplate should return non-empty content: err=%v", err)
	}

	templates, err := ListDefaultTemplates()
	if err != nil {
		t.Fatalf("ListDefaultTemplates failed: %v", err)
	}
	if len(templates) != len(keysAgain) {
		t.Fatalf("templates count mismatch: got=%d want=%d", len(templates), len(keysAgain))
	}
	if templates[0].Source != "embedded_default" {
		t.Fatalf("unexpected default template source: %s", templates[0].Source)
	}

	if got := keyID(" dev ", " review "); got != "dev:review" {
		t.Fatalf("keyID trim mismatch: %q", got)
	}
}
