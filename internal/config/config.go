package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type Config struct {
	App    AppConfig    `mapstructure:"app"`
	Server ServerConfig `mapstructure:"server"`
	Log    LogConfig    `mapstructure:"log"`
	DB     DBConfig     `mapstructure:"db"`
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
	return &cfg, nil
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
	return nil
}

func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
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
