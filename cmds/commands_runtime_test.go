package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"syscall"
	"testing"
	"time"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	"github.com/haowen-xu/agent-coder/internal/infra/secret"

	"github.com/haowen-xu/agent-coder/internal/app"
	"github.com/haowen-xu/agent-coder/internal/service/worker"
)

type depsSnapshot struct {
	newApplication    func(context.Context, string) (*app.App, error)
	newWorkerSvc      func(*appcfg.Config, *slog.Logger, *db.Client, *promptstore.Service, secret.Manager) *worker.Service
	notifySignals     func(chan<- os.Signal, ...os.Signal)
	stopSignals       func(chan<- os.Signal)
	runServer         func(*app.App) error
	shutdownServer    func(*app.App, context.Context) error
	ensureWebUIAssets func() error
	runWorkerLoop     func(*worker.Service, context.Context) error
	runWorkerOnce     func(*worker.Service, context.Context) error
}

func patchDeps(t *testing.T) {
	t.Helper()
	snap := depsSnapshot{
		newApplication:    newApplication,
		newWorkerSvc:      newWorkerSvc,
		notifySignals:     notifySignals,
		stopSignals:       stopSignals,
		runServer:         runServer,
		shutdownServer:    shutdownServer,
		ensureWebUIAssets: ensureWebUIAssets,
		runWorkerLoop:     runWorkerLoop,
		runWorkerOnce:     runWorkerOnce,
	}
	t.Cleanup(func() {
		newApplication = snap.newApplication
		newWorkerSvc = snap.newWorkerSvc
		notifySignals = snap.notifySignals
		stopSignals = snap.stopSignals
		runServer = snap.runServer
		shutdownServer = snap.shutdownServer
		ensureWebUIAssets = snap.ensureWebUIAssets
		runWorkerLoop = snap.runWorkerLoop
		runWorkerOnce = snap.runWorkerOnce
	})
}

// TestServerCmdRunE_ErrorAndSignalPaths 用于单元测试。
func TestServerCmdRunE_ErrorAndSignalPaths(t *testing.T) {
	patchDeps(t)
	ensureWebUIAssets = func() error { return nil }
	newApplication = func(_ context.Context, _ string) (*app.App, error) {
		return &app.App{
			Config: &appcfg.Config{
				Server: appcfg.ServerConfig{ShutdownTimeout: "100ms"},
			},
		}, nil
	}
	stopSignals = func(chan<- os.Signal) {}

	t.Run("server_run_error", func(t *testing.T) {
		wantErr := errors.New("server run failed")
		ensureWebUIAssets = func() error { return nil }
		runServer = func(_ *app.App) error { return wantErr }
		notifySignals = func(chan<- os.Signal, ...os.Signal) {}
		shutdownServer = func(_ *app.App, _ context.Context) error { return nil }

		cmd := serverCmd()
		if err := cmd.RunE(cmd, nil); !errors.Is(err, wantErr) {
			t.Fatalf("expected run error, got: %v", err)
		}
	})

	t.Run("assets_check_error", func(t *testing.T) {
		wantErr := errors.New("webui assets missing")
		ensureWebUIAssets = func() error { return wantErr }
		cmd := serverCmd()
		if err := cmd.RunE(cmd, nil); !errors.Is(err, wantErr) {
			t.Fatalf("expected webui assets error, got: %v", err)
		}
	})

	t.Run("new_application_error", func(t *testing.T) {
		wantErr := errors.New("app init failed")
		ensureWebUIAssets = func() error { return nil }
		newApplication = func(_ context.Context, _ string) (*app.App, error) {
			return nil, wantErr
		}
		cmd := serverCmd()
		if err := cmd.RunE(cmd, nil); !errors.Is(err, wantErr) {
			t.Fatalf("expected app init error, got: %v", err)
		}
	})

	t.Run("signal_shutdown", func(t *testing.T) {
		ensureWebUIAssets = func() error { return nil }
		newApplication = func(_ context.Context, _ string) (*app.App, error) {
			return &app.App{
				Config: &appcfg.Config{
					Server: appcfg.ServerConfig{ShutdownTimeout: "100ms"},
				},
			}, nil
		}
		runServer = func(_ *app.App) error {
			time.Sleep(120 * time.Millisecond)
			return nil
		}
		notifySignals = func(ch chan<- os.Signal, _ ...os.Signal) {
			ch <- syscall.SIGTERM
		}
		shutdownCalled := false
		shutdownServer = func(_ *app.App, _ context.Context) error {
			shutdownCalled = true
			return nil
		}

		cmd := serverCmd()
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("expected nil on signal shutdown, got: %v", err)
		}
		if !shutdownCalled {
			t.Fatalf("shutdownServer should be called on signal path")
		}
	})
}

// TestWorkerCmdRunE_ErrorAndSignalPaths 用于单元测试。
func TestWorkerCmdRunE_ErrorAndSignalPaths(t *testing.T) {
	patchDeps(t)
	newApplication = func(_ context.Context, _ string) (*app.App, error) {
		return &app.App{Config: &appcfg.Config{}}, nil
	}
	newWorkerSvc = func(*appcfg.Config, *slog.Logger, *db.Client, *promptstore.Service, secret.Manager) *worker.Service {
		return nil
	}
	stopSignals = func(chan<- os.Signal) {}

	t.Run("worker_loop_error", func(t *testing.T) {
		wantErr := errors.New("worker loop failed")
		runWorkerLoop = func(_ *worker.Service, _ context.Context) error { return wantErr }
		notifySignals = func(chan<- os.Signal, ...os.Signal) {}

		cmd := workerCmd()
		if err := cmd.RunE(cmd, nil); !errors.Is(err, wantErr) {
			t.Fatalf("expected worker loop error, got: %v", err)
		}
	})

	t.Run("signal_cancel", func(t *testing.T) {
		runWorkerLoop = func(_ *worker.Service, _ context.Context) error {
			time.Sleep(120 * time.Millisecond)
			return nil
		}
		notifySignals = func(ch chan<- os.Signal, _ ...os.Signal) {
			ch <- syscall.SIGINT
		}

		cmd := workerCmd()
		if err := cmd.RunE(cmd, nil); err != nil {
			t.Fatalf("expected nil on signal path, got: %v", err)
		}
	})
}

// TestMigrateAndSyncIssuesSuccessPaths 用于单元测试。
func TestMigrateAndSyncIssuesSuccessPaths(t *testing.T) {
	patchDeps(t)
	newApplication = func(_ context.Context, _ string) (*app.App, error) {
		return &app.App{Config: &appcfg.Config{}}, nil
	}
	newWorkerSvc = func(*appcfg.Config, *slog.Logger, *db.Client, *promptstore.Service, secret.Manager) *worker.Service {
		return nil
	}
	runWorkerOnce = func(_ *worker.Service, _ context.Context) error { return nil }

	migrate := migrateCmd()
	if err := migrate.RunE(migrate, nil); err != nil {
		t.Fatalf("migrate success path should return nil, got: %v", err)
	}

	syncIssues := syncIssuesCmd()
	if err := syncIssues.RunE(syncIssues, nil); err != nil {
		t.Fatalf("sync-issues success path should return nil, got: %v", err)
	}
}
