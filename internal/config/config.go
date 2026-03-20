package config

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type Config struct {
	App           AppConfig           `mapstructure:"app"`
	Server        ServerConfig        `mapstructure:"server"`
	Log           LogConfig           `mapstructure:"log"`
	DB            DBConfig            `mapstructure:"db"`
	Secret        SecretConfig        `mapstructure:"secret"`
	Auth          AuthConfig          `mapstructure:"auth"`
	Work          WorkConfig          `mapstructure:"work"`
	Agent         AgentConfig         `mapstructure:"agent"`
	Scheduler     SchedulerConfig     `mapstructure:"scheduler"`
	RepoProvider  RepoProviderConfig  `mapstructure:"repo_provider"`
	IssueProvider IssueProviderConfig `mapstructure:"issue_provider"`
	Bootstrap     BootstrapConfig     `mapstructure:"bootstrap"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

type ServerConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	ReadTimeout     string `mapstructure:"read_timeout"`
	WriteTimeout    string `mapstructure:"write_timeout"`
	ShutdownTimeout string `mapstructure:"shutdown_timeout"`
}

type LogConfig struct {
	Level     string `mapstructure:"level"`
	Format    string `mapstructure:"format"`
	AddSource bool   `mapstructure:"add_source"`
}

type DBConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Driver          string `mapstructure:"driver"`
	SQLitePath      string `mapstructure:"sqlite_path"`
	PostgresDSN     string `mapstructure:"postgres_dsn"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime"`
	AutoMigrate     bool   `mapstructure:"auto_migrate"`
}

type SecretConfig struct {
	Provider  string `mapstructure:"provider"`
	EnvPrefix string `mapstructure:"env_prefix"`
}

type AuthConfig struct {
	SessionTTL string `mapstructure:"session_ttl"`
}

type WorkConfig struct {
	WorkDir string `mapstructure:"work_dir"`
}

type AgentConfig struct {
	Provider string           `mapstructure:"provider"`
	Codex    AgentCodexConfig `mapstructure:"codex"`
}

type AgentCodexConfig struct {
	Binary      string `mapstructure:"binary"`
	Sandbox     bool   `mapstructure:"sandbox"`
	TimeoutSec  int    `mapstructure:"timeout_sec"`
	MaxRetry    int    `mapstructure:"max_retry"`
	MaxLoopStep int    `mapstructure:"max_loop_step"`
}

type SchedulerConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	PollIntervalSec int    `mapstructure:"poll_interval_sec"`
	RunEvery        string `mapstructure:"run_every"`
}

type RepoProviderConfig struct {
	HTTPTimeoutSec int `mapstructure:"http_timeout_sec"`
}

// IssueProviderConfig 保留兼容旧配置项 issue_provider。
// 新配置建议使用 repo_provider。
type IssueProviderConfig = RepoProviderConfig

type BootstrapConfig struct {
	AdminUsername string `mapstructure:"admin_username"`
	AdminPassword string `mapstructure:"admin_password"`
}

var (
	cfgPtr atomic.Pointer[Config]
)

func Load(path string) (*Config, error) {
	v := viper.New()
	setDefaults(v)
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("AGENT_CODER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, xerr.Config.Wrap(err, "read config file: %s", path)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, xerr.Config.Wrap(err, "unmarshal config")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cfgPtr.Store(&cfg)
	return &cfg, nil
}

func Current() *Config {
	return cfgPtr.Load()
}

func Replace(cfg *Config) {
	if cfg == nil {
		return
	}
	cfgPtr.Store(cfg)
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "agent-coder")
	v.SetDefault("app.env", "dev")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")
	v.SetDefault("server.shutdown_timeout", "10s")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.add_source", false)

	v.SetDefault("db.enabled", true)
	v.SetDefault("db.driver", "sqlite")
	v.SetDefault("db.sqlite_path", "agent-coder.db")
	v.SetDefault("db.postgres_dsn", "")
	v.SetDefault("db.max_open_conns", 20)
	v.SetDefault("db.max_idle_conns", 10)
	v.SetDefault("db.conn_max_lifetime", "30m")
	v.SetDefault("db.auto_migrate", true)

	v.SetDefault("secret.provider", "env")
	v.SetDefault("secret.env_prefix", "AGENT_CODER_SECRET_")

	v.SetDefault("auth.session_ttl", "72h")

	v.SetDefault("work.work_dir", ".agent-coder/workdirs")

	v.SetDefault("agent.provider", "codex")
	v.SetDefault("agent.codex.binary", "codex")
	v.SetDefault("agent.codex.sandbox", true)
	v.SetDefault("agent.codex.timeout_sec", 7200)
	v.SetDefault("agent.codex.max_retry", 5)
	v.SetDefault("agent.codex.max_loop_step", 5)

	v.SetDefault("scheduler.enabled", true)
	v.SetDefault("scheduler.poll_interval_sec", 30)
	v.SetDefault("scheduler.run_every", "30s")

	v.SetDefault("repo_provider.http_timeout_sec", 30)
	v.SetDefault("issue_provider.http_timeout_sec", 30)

	v.SetDefault("bootstrap.admin_username", "admin")
	v.SetDefault("bootstrap.admin_password", "admin123")
}

func (c *Config) Validate() error {
	driver := strings.ToLower(c.DB.Driver)
	if c.DB.Enabled {
		switch driver {
		case "sqlite":
			if strings.TrimSpace(c.DB.SQLitePath) == "" {
				return xerr.Config.New("db.sqlite_path is required when db.driver=sqlite")
			}
		case "postgres":
			if strings.TrimSpace(c.DB.PostgresDSN) == "" {
				return xerr.Config.New("db.postgres_dsn is required when db.driver=postgres")
			}
		default:
			return xerr.Config.New("unsupported db.driver: %s", c.DB.Driver)
		}
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return xerr.Config.New("server.port must be in 1~65535")
	}
	if _, err := time.ParseDuration(c.Server.ReadTimeout); err != nil {
		return xerr.Config.Wrap(err, "invalid server.read_timeout")
	}
	if _, err := time.ParseDuration(c.Server.WriteTimeout); err != nil {
		return xerr.Config.Wrap(err, "invalid server.write_timeout")
	}
	if _, err := time.ParseDuration(c.Server.ShutdownTimeout); err != nil {
		return xerr.Config.Wrap(err, "invalid server.shutdown_timeout")
	}
	if _, err := time.ParseDuration(c.DB.ConnMaxLifetime); err != nil {
		return xerr.Config.Wrap(err, "invalid db.conn_max_lifetime")
	}
	if p := strings.ToLower(strings.TrimSpace(c.Secret.Provider)); p != "" && p != "env" {
		return xerr.Config.New("unsupported secret.provider: %s", c.Secret.Provider)
	}
	if _, err := time.ParseDuration(c.Auth.SessionTTL); err != nil {
		return xerr.Config.Wrap(err, "invalid auth.session_ttl")
	}
	if strings.TrimSpace(c.Work.WorkDir) == "" {
		return xerr.Config.New("work.work_dir is required")
	}
	if c.Agent.Codex.TimeoutSec <= 0 {
		return xerr.Config.New("agent.codex.timeout_sec must be > 0")
	}
	if c.Agent.Codex.MaxRetry <= 0 {
		return xerr.Config.New("agent.codex.max_retry must be > 0")
	}
	if c.Agent.Codex.MaxLoopStep <= 0 {
		return xerr.Config.New("agent.codex.max_loop_step must be > 0")
	}
	if strings.TrimSpace(c.Bootstrap.AdminUsername) == "" {
		return xerr.Config.New("bootstrap.admin_username is required")
	}
	if strings.TrimSpace(c.Bootstrap.AdminPassword) == "" {
		return xerr.Config.New("bootstrap.admin_password is required")
	}
	return nil
}

func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// RepoHTTPTimeoutSec 返回仓库平台 API 超时配置。
// 优先读取 repo_provider，若未配置则回退到 issue_provider（兼容旧字段）。
func (c *Config) RepoHTTPTimeoutSec() int {
	if c == nil {
		return 30
	}
	if c.RepoProvider.HTTPTimeoutSec > 0 {
		return c.RepoProvider.HTTPTimeoutSec
	}
	if c.IssueProvider.HTTPTimeoutSec > 0 {
		return c.IssueProvider.HTTPTimeoutSec
	}
	return 30
}

func (c *ServerConfig) ReadTimeoutDuration() time.Duration {
	d, err := time.ParseDuration(c.ReadTimeout)
	if err != nil {
		return 15 * time.Second
	}
	return d
}

func (c *ServerConfig) WriteTimeoutDuration() time.Duration {
	d, err := time.ParseDuration(c.WriteTimeout)
	if err != nil {
		return 15 * time.Second
	}
	return d
}

func (c *ServerConfig) ShutdownTimeoutDuration() time.Duration {
	d, err := time.ParseDuration(c.ShutdownTimeout)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

func (c *DBConfig) ConnMaxLifetimeDuration() time.Duration {
	d, err := time.ParseDuration(c.ConnMaxLifetime)
	if err != nil {
		return 30 * time.Minute
	}
	return d
}

func (c *AuthConfig) SessionTTLDuration() time.Duration {
	d, err := time.ParseDuration(c.SessionTTL)
	if err != nil {
		return 72 * time.Hour
	}
	return d
}

func (c *SchedulerConfig) RunEveryDuration() time.Duration {
	d, err := time.ParseDuration(c.RunEvery)
	if err != nil {
		return 30 * time.Second
	}
	return d
}
