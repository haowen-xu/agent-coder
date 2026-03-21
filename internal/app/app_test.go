package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeAppTestConfig(t *testing.T, dbPath string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	raw := []byte("app:\n  name: agent-coder-test\n  env: test\n" +
		"server:\n  host: 127.0.0.1\n  port: 18081\n" +
		"db:\n  enabled: true\n  driver: sqlite\n  sqlite_path: " + dbPath + "\n  auto_migrate: true\n" +
		"secret:\n  provider: env\n  env_prefix: AGENT_CODER_SECRET_\n" +
		"auth:\n  session_ttl: 1h\n" +
		"work:\n  work_dir: .agent-coder/workdirs\n" +
		"agent:\n  codex:\n    binary: codex\n    timeout_sec: 60\n    max_retry: 3\n    max_loop_step: 3\n" +
		"bootstrap:\n  admin_username: admin\n  admin_password: admin123\n")
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config failed: %v", err)
	}
	return cfgPath
}

// TestAppNewAndClose 用于单元测试。
func TestAppNewAndClose(t *testing.T) {
	ctx := context.Background()

	if _, err := New(ctx, filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatalf("expected app new error for missing config")
	}

	dbPath := filepath.Join(t.TempDir(), "app.db")
	cfgPath := writeAppTestConfig(t, dbPath)
	application, err := New(ctx, cfgPath)
	if err != nil {
		t.Fatalf("app New failed: %v", err)
	}
	if application.Config == nil || application.DB == nil || application.CoreService == nil || application.Server == nil {
		t.Fatalf("app components should be initialized")
	}

	admin, err := application.DB.GetUserByUsername(ctx, "admin")
	if err != nil || admin == nil {
		t.Fatalf("bootstrap admin should exist: user=%#v err=%v", admin, err)
	}

	if err := application.Close(); err != nil {
		t.Fatalf("app close failed: %v", err)
	}
	var nilApp *App
	if err := nilApp.Close(); err != nil {
		t.Fatalf("nil app close should not fail: %v", err)
	}
}
