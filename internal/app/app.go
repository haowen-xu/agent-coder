package app

import (
	"context"
	"log/slog"
	"strings"

	"github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/handler/httpserver"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	"github.com/haowen-xu/agent-coder/internal/infra/secret"
	"github.com/haowen-xu/agent-coder/internal/logger"
	"github.com/haowen-xu/agent-coder/internal/service/core"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// App 表示数据结构定义。
type App struct {
	Config      *config.Config       // Config 字段说明。
	Logger      *slog.Logger         // Logger 字段说明。
	DB          *db.Client           // DB 字段说明。
	Secret      secret.Manager       // Secret 字段说明。
	PromptStore *promptstore.Service // PromptStore 字段说明。
	CoreService *core.Service        // CoreService 字段说明。
	Server      *httpserver.Server   // Server 字段说明。
}

// New 执行相关逻辑。
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

	var secretMgr secret.Manager
	switch strings.ToLower(strings.TrimSpace(cfg.Secret.Provider)) {
	case "", "env":
		secretMgr = secret.NewEnvManager(cfg.Secret.EnvPrefix)
	default:
		return nil, xerr.Startup.New("unsupported secret provider: %s", cfg.Secret.Provider)
	}
	promptService := promptstore.NewService(dbClient)
	coreService := core.New(cfg, dbClient, promptService)
	srv := httpserver.New(cfg, log, dbClient, coreService)
	return &App{
		Config:      cfg,
		Logger:      log,
		DB:          dbClient,
		Secret:      secretMgr,
		PromptStore: promptService,
		CoreService: coreService,
		Server:      srv,
	}, nil
}

// Close 是 *App 的方法实现。
func (a *App) Close() error {
	if a == nil || a.DB == nil {
		return nil
	}
	return a.DB.Close()
}
