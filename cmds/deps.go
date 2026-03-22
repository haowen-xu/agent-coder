package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/haowen-xu/agent-coder/internal/app"
	"github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	"github.com/haowen-xu/agent-coder/internal/infra/secret"
	orchsvc "github.com/haowen-xu/agent-coder/internal/service/orch"
)

var (
	newApplication    func(context.Context, string) (*app.App, error)
	newWorkerSvc      func(*config.Config, *slog.Logger, *db.Client, *promptstore.Service, secret.Manager) *orchsvc.Service
	notifySignals     func(chan<- os.Signal, ...os.Signal)
	stopSignals       func(chan<- os.Signal)
	runServer         func(*app.App) error
	shutdownServer    func(*app.App, context.Context) error
	ensureWebUIAssets func() error
	runWorkerLoop     func(*orchsvc.Service, context.Context) error
	runWorkerOnce     func(*orchsvc.Service, context.Context) error
)

func init() {
	newApplication = app.New
	newWorkerSvc = orchsvc.New
	notifySignals = signal.Notify
	stopSignals = signal.Stop
	runServer = defaultRunServer
	shutdownServer = defaultShutdownServer
	ensureWebUIAssets = defaultEnsureWebUIAssets
	runWorkerLoop = defaultRunWorkerLoop
	runWorkerOnce = defaultRunWorkerOnce
}

func defaultRunServer(application *app.App) error {
	return application.Server.Run()
}

func defaultShutdownServer(application *app.App, ctx context.Context) error {
	return application.Server.Shutdown(ctx)
}

func defaultRunWorkerLoop(wk *orchsvc.Service, ctx context.Context) error {
	return wk.RunLoop(ctx)
}

func defaultRunWorkerOnce(wk *orchsvc.Service, ctx context.Context) error {
	return wk.RunOnce(ctx)
}
