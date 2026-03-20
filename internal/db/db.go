package db

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

	"github.com/joomcode/errorx"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type Client struct {
	db      *gorm.DB
	enabled bool
	dialect string
}

func New(ctx context.Context, cfg config.DBConfig, log *slog.Logger) (*Client, error) {
	if !cfg.Enabled {
		log.Warn("database disabled")
		return &Client{enabled: false, dialect: strings.ToLower(cfg.Driver)}, nil
	}

	dialector, dialect, err := buildDialector(cfg)
	if err != nil {
		return nil, err
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "open gorm db")
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, xerr.Infra.Wrap(err, "get sql db")
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetimeDuration())
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, xerr.Infra.Wrap(err, "ping db")
	}

	client := &Client{db: gdb, enabled: true, dialect: dialect}
	if cfg.AutoMigrate {
		if err := client.db.AutoMigrate(&SystemInfo{}, &PromptTemplate{}); err != nil {
			return nil, xerr.Infra.Wrap(err, "auto migrate")
		}
		if err := client.db.WithContext(ctx).
			Where("key = ?", "app_name").
			FirstOrCreate(&SystemInfo{Key: "app_name", Value: "agent-coder"}).Error; err != nil {
			return nil, xerr.Infra.Wrap(err, "seed system_info")
		}
	}

	log.Info("database connected", slog.String("dialect", dialect))
	return client, nil
}

func buildDialector(cfg config.DBConfig) (gorm.Dialector, string, error) {
	switch strings.ToLower(cfg.Driver) {
	case "sqlite":
		return sqlite.Open(cfg.SQLitePath), "sqlite", nil
	case "postgres":
		if strings.TrimSpace(cfg.PostgresDSN) == "" {
			return nil, "", xerr.Config.New("postgres_dsn is required when db.driver=postgres")
		}
		return postgres.Open(cfg.PostgresDSN), "postgres", nil
	default:
		return nil, "", errorx.IllegalArgument.New("unsupported db driver: %s", cfg.Driver)
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}

func (c *Client) Dialect() string {
	if c == nil {
		return ""
	}
	return c.dialect
}

func (c *Client) DB() *gorm.DB {
	if c == nil {
		return nil
	}
	return c.db
}

func (c *Client) SQLDB() *sql.DB {
	if c == nil || c.db == nil {
		return nil
	}
	sqlDB, err := c.db.DB()
	if err != nil {
		return nil
	}
	return sqlDB
}

func (c *Client) Close() error {
	sqlDB := c.SQLDB()
	if sqlDB == nil {
		return nil
	}
	return sqlDB.Close()
}
