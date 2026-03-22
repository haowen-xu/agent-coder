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
	"github.com/haowen-xu/agent-coder/internal/utils"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

// Client 表示数据结构定义。
type Client struct {
	db      *gorm.DB // db 字段说明。
	enabled bool     // enabled 字段说明。
	dialect string   // dialect 字段说明。
}

// New 执行相关逻辑。
func New(ctx context.Context, cfg config.DBConfig, log *slog.Logger) (*Client, error) {
	if !cfg.Enabled {
		log.Warn("database disabled")
		return &Client{enabled: false, dialect: strings.ToLower(cfg.Driver)}, nil
	}

	dialector, dialect, err := buildDialector(cfg)
	if err != nil {
		return nil, err
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		NowFunc: utils.NowUTC,
	})
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
		if err := client.db.AutoMigrate(
			&SystemInfo{},
			&User{},
			&UserSession{},
			&Project{},
			&Issue{},
			&IssueRun{},
			&RunLog{},
			&PromptTemplate{},
		); err != nil {
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

// buildDialector 执行相关逻辑。
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

// Enabled 是 *Client 的方法实现。
func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}

// Dialect 是 *Client 的方法实现。
func (c *Client) Dialect() string {
	if c == nil {
		return ""
	}
	return c.dialect
}

// DB 是 *Client 的方法实现。
func (c *Client) DB() *gorm.DB {
	if c == nil {
		return nil
	}
	return c.db
}

// SQLDB 是 *Client 的方法实现。
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

// Close 是 *Client 的方法实现。
func (c *Client) Close() error {
	sqlDB := c.SQLDB()
	if sqlDB == nil {
		return nil
	}
	return sqlDB.Close()
}
