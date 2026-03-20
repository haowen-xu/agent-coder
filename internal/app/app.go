package app

import (
	"context"
	"log/slog"

	"github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/db"
	"github.com/haowen-xu/agent-coder/internal/httpserver"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	"github.com/haowen-xu/agent-coder/internal/logger"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type App struct {
	Config *config.Config
	Logger *slog.Logger
	DB     *db.Client
	Server *httpserver.Server
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

	promptService := promptstore.NewService(dbClient)
	srv := httpserver.New(cfg, log, dbClient, promptService)
	return &App{
		Config: cfg,
		Logger: log,
		DB:     dbClient,
		Server: srv,
	}, nil
}

func (a *App) Close() error {
	if a == nil || a.DB == nil {
		return nil
	}
	return a.DB.Close()
}
