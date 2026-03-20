package app

import (
	"context"
	"log/slog"
	"strings"

	"github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/handler/httpserver"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	"github.com/haowen-xu/agent-coder/internal/logger"
	"github.com/haowen-xu/agent-coder/internal/service/core"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type App struct {
	Config      *config.Config
	Logger      *slog.Logger
	DB          *db.Client
	PromptStore *promptstore.Service
	CoreService *core.Service
	Server      *httpserver.Server
}

func New(ctx context.Context, configPath string) (*App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, xerr.Startup.Wrap(err, "load config")
	}

	log := logger.New(cfg.Log)
	dbClient, err := db.New(ctx, cfg.DB, log)
	if err != nil {
		return nil, xerr.Startup.Wrap(err, "init db")
	}
	if dbClient.Enabled() {
		if err := dbClient.EnsureBootstrapAdmin(
			ctx,
			strings.TrimSpace(cfg.Bootstrap.AdminUsername),
			cfg.Bootstrap.AdminPassword,
		); err != nil {
			return nil, xerr.Startup.Wrap(err, "ensure bootstrap admin")
		}
	}

	promptService := promptstore.NewService(dbClient)
	coreService := core.New(cfg, dbClient, promptService)
	srv := httpserver.New(cfg, log, dbClient, coreService)
	return &App{
		Config:      cfg,
		Logger:      log,
		DB:          dbClient,
		PromptStore: promptService,
		CoreService: coreService,
		Server:      srv,
	}, nil
}

func (a *App) Close() error {
	if a == nil || a.DB == nil {
		return nil
	}
	return a.DB.Close()
}
