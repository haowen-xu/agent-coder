package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultEnsureWebUIAssets 用于单元测试。
func TestDefaultEnsureWebUIAssets(t *testing.T) {
	oldDir := webuiStaticDirPath
	t.Cleanup(func() { webuiStaticDirPath = oldDir })

	t.Run("missing", func(t *testing.T) {
		webuiStaticDirPath = t.TempDir()
		err := defaultEnsureWebUIAssets()
		if err == nil {
			t.Fatal("expected error when static assets are missing")
		}
		if !strings.Contains(err.Error(), "pnpm build") {
			t.Fatalf("expected pnpm build hint, got: %v", err)
		}
	})

	t.Run("index_is_dir", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "index.html"), 0o755); err != nil {
			t.Fatalf("mkdir index path failed: %v", err)
		}
		webuiStaticDirPath = dir
		err := defaultEnsureWebUIAssets()
		if err == nil {
			t.Fatal("expected error when index.html is a directory")
		}
	})

	t.Run("index_without_assets", func(t *testing.T) {
		dir := t.TempDir()
		indexPath := filepath.Join(dir, "index.html")
		if err := os.WriteFile(indexPath, []byte("ok"), 0o644); err != nil {
			t.Fatalf("write index failed: %v", err)
		}
		webuiStaticDirPath = dir
		err := defaultEnsureWebUIAssets()
		if err == nil {
			t.Fatal("expected error when assets js bundle is missing")
		}
		if !strings.Contains(err.Error(), "pnpm build") {
			t.Fatalf("expected pnpm build hint, got: %v", err)
		}
	})

	t.Run("exists", func(t *testing.T) {
		dir := t.TempDir()
		indexPath := filepath.Join(dir, "index.html")
		if err := os.WriteFile(indexPath, []byte("ok"), 0o644); err != nil {
			t.Fatalf("write index failed: %v", err)
		}
		assetsDir := filepath.Join(dir, "assets")
		if err := os.MkdirAll(assetsDir, 0o755); err != nil {
			t.Fatalf("mkdir assets failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(assetsDir, "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
			t.Fatalf("write js asset failed: %v", err)
		}
		webuiStaticDirPath = dir
		if err := defaultEnsureWebUIAssets(); err != nil {
			t.Fatalf("expected nil when index exists, got: %v", err)
		}
	})
}
