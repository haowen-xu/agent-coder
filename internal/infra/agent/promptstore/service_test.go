package promptstore

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
)

func newPromptStoreDB(t *testing.T) *db.Client {
	t.Helper()
	client, err := db.New(context.Background(), appcfg.DBConfig{
		Enabled:         true,
		Driver:          "sqlite",
		SQLitePath:      filepath.Join(t.TempDir(), "promptstore.db"),
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: "1m",
		AutoMigrate:     true,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("init promptstore db failed: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// TestPromptStoreService 用于单元测试。
func TestPromptStoreService(t *testing.T) {
	ctx := context.Background()

	svc := NewService(nil)
	defaults, err := svc.ListDefaults()
	if err != nil || len(defaults) == 0 {
		t.Fatalf("ListDefaults mismatch: len=%d err=%v", len(defaults), err)
	}

	if _, err := svc.ListEffectiveByProject(ctx, " "); err == nil {
		t.Fatalf("expected project_key validation error")
	}
	effectiveWithoutDB, err := svc.ListEffectiveByProject(ctx, "p1")
	if err != nil || len(effectiveWithoutDB) == 0 {
		t.Fatalf("ListEffectiveByProject without db mismatch: len=%d err=%v", len(effectiveWithoutDB), err)
	}

	if _, err := svc.UpsertProjectOverride(ctx, "p1", "dev", "dev", "override"); err == nil {
		t.Fatalf("expected upsert override to fail when db disabled")
	}
	if err := svc.DeleteProjectOverride(ctx, "p1", "dev", "dev"); err == nil {
		t.Fatalf("expected delete override to fail when db disabled")
	}

	dbClient := newPromptStoreDB(t)
	svcWithDB := NewService(dbClient)
	if _, err := svcWithDB.UpsertProjectOverride(ctx, " ", "dev", "dev", "x"); err == nil {
		t.Fatalf("expected project_key validation error")
	}
	if _, err := svcWithDB.UpsertProjectOverride(ctx, "p1", "dev", "dev", " "); err == nil {
		t.Fatalf("expected content validation error")
	}
	if _, err := svcWithDB.UpsertProjectOverride(ctx, "p1", "unknown", "dev", "x"); err == nil {
		t.Fatalf("expected invalid key error")
	}

	override, err := svcWithDB.UpsertProjectOverride(ctx, "p1", "dev", "dev", "override content")
	if err != nil || override == nil || override.Source != "project_override" {
		t.Fatalf("UpsertProjectOverride mismatch: row=%#v err=%v", override, err)
	}

	effectiveWithOverride, err := svcWithDB.ListEffectiveByProject(ctx, "p1")
	if err != nil {
		t.Fatalf("ListEffectiveByProject with override failed: %v", err)
	}
	foundOverride := false
	for _, row := range effectiveWithOverride {
		if row.RunKind == "dev" && row.AgentRole == "dev" {
			if row.Source != "project_override" || row.Content != "override content" {
				t.Fatalf("override row mismatch: %#v", row)
			}
			foundOverride = true
		}
	}
	if !foundOverride {
		t.Fatalf("expected overridden dev/dev prompt")
	}

	if err := svcWithDB.DeleteProjectOverride(ctx, "p1", "dev", "dev"); err != nil {
		t.Fatalf("DeleteProjectOverride failed: %v", err)
	}
}
