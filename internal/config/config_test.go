package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

// TestValidate 用于单元测试。
func TestValidate(t *testing.T) {
	t.Parallel()

	valid := newValidConfig()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config should pass, got: %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(c *Config)
		contain string
	}{
		{
			name: "sqlite requires path",
			mutate: func(c *Config) {
				c.DB.Driver = "sqlite"
				c.DB.SQLitePath = ""
			},
			contain: "db.sqlite_path",
		},
		{
			name: "postgres requires dsn",
			mutate: func(c *Config) {
				c.DB.Driver = "postgres"
				c.DB.PostgresDSN = ""
			},
			contain: "db.postgres_dsn",
		},
		{
			name: "unsupported db driver",
			mutate: func(c *Config) {
				c.DB.Driver = "mysql"
			},
			contain: "unsupported db.driver",
		},
		{
			name: "invalid port",
			mutate: func(c *Config) {
				c.Server.Port = 0
			},
			contain: "server.port",
		},
		{
			name: "invalid server read timeout",
			mutate: func(c *Config) {
				c.Server.ReadTimeout = "bad"
			},
			contain: "server.read_timeout",
		},
		{
			name: "invalid secret provider",
			mutate: func(c *Config) {
				c.Secret.Provider = "vault"
			},
			contain: "unsupported secret.provider",
		},
		{
			name: "empty work dir",
			mutate: func(c *Config) {
				c.Work.WorkDir = " "
			},
			contain: "work.work_dir",
		},
		{
			name: "invalid codex timeout",
			mutate: func(c *Config) {
				c.Agent.Codex.TimeoutSec = 0
			},
			contain: "agent.codex.timeout_sec",
		},
		{
			name: "invalid codex max retry",
			mutate: func(c *Config) {
				c.Agent.Codex.MaxRetry = 0
			},
			contain: "agent.codex.max_retry",
		},
		{
			name: "invalid codex max loop",
			mutate: func(c *Config) {
				c.Agent.Codex.MaxLoopStep = 0
			},
			contain: "agent.codex.max_loop_step",
		},
		{
			name: "empty bootstrap username",
			mutate: func(c *Config) {
				c.Bootstrap.AdminUsername = " "
			},
			contain: "bootstrap.admin_username",
		},
		{
			name: "empty bootstrap password",
			mutate: func(c *Config) {
				c.Bootstrap.AdminPassword = " "
			},
			contain: "bootstrap.admin_password",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := newValidConfig()
			tc.mutate(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.contain) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestRepoHTTPTimeoutSec 用于单元测试。
func TestRepoHTTPTimeoutSec(t *testing.T) {
	t.Parallel()

	var nilCfg *Config
	if got := nilCfg.RepoHTTPTimeoutSec(); got != 30 {
		t.Fatalf("nil config timeout mismatch: %d", got)
	}

	cfg := newValidConfig()
	cfg.RepoProvider.HTTPTimeoutSec = 50
	cfg.IssueProvider.HTTPTimeoutSec = 20
	if got := cfg.RepoHTTPTimeoutSec(); got != 50 {
		t.Fatalf("repo_provider should take precedence, got: %d", got)
	}

	cfg.RepoProvider.HTTPTimeoutSec = 0
	cfg.IssueProvider.HTTPTimeoutSec = 25
	if got := cfg.RepoHTTPTimeoutSec(); got != 25 {
		t.Fatalf("issue_provider fallback mismatch: %d", got)
	}

	cfg.IssueProvider.HTTPTimeoutSec = 0
	if got := cfg.RepoHTTPTimeoutSec(); got != 30 {
		t.Fatalf("default timeout mismatch: %d", got)
	}
}

// TestDurationHelpers 用于单元测试。
func TestDurationHelpers(t *testing.T) {
	t.Parallel()

	server := ServerConfig{
		Host:            "0.0.0.0",
		Port:            8080,
		ReadTimeout:     "2s",
		WriteTimeout:    "3s",
		ShutdownTimeout: "4s",
	}
	if server.Address() != "0.0.0.0:8080" {
		t.Fatalf("unexpected address: %s", server.Address())
	}
	if server.ReadTimeoutDuration() != 2*time.Second {
		t.Fatalf("read timeout parse mismatch")
	}
	if server.WriteTimeoutDuration() != 3*time.Second {
		t.Fatalf("write timeout parse mismatch")
	}
	if server.ShutdownTimeoutDuration() != 4*time.Second {
		t.Fatalf("shutdown timeout parse mismatch")
	}

	server.ReadTimeout = "bad"
	server.WriteTimeout = "bad"
	server.ShutdownTimeout = "bad"
	if server.ReadTimeoutDuration() != 15*time.Second {
		t.Fatalf("read timeout fallback mismatch")
	}
	if server.WriteTimeoutDuration() != 15*time.Second {
		t.Fatalf("write timeout fallback mismatch")
	}
	if server.ShutdownTimeoutDuration() != 10*time.Second {
		t.Fatalf("shutdown timeout fallback mismatch")
	}

	db := DBConfig{ConnMaxLifetime: "2m"}
	if db.ConnMaxLifetimeDuration() != 2*time.Minute {
		t.Fatalf("conn max lifetime parse mismatch")
	}
	db.ConnMaxLifetime = "bad"
	if db.ConnMaxLifetimeDuration() != 30*time.Minute {
		t.Fatalf("conn max lifetime fallback mismatch")
	}

	auth := AuthConfig{SessionTTL: "5h"}
	if auth.SessionTTLDuration() != 5*time.Hour {
		t.Fatalf("session ttl parse mismatch")
	}
	auth.SessionTTL = "bad"
	if auth.SessionTTLDuration() != 72*time.Hour {
		t.Fatalf("session ttl fallback mismatch")
	}

	scheduler := SchedulerConfig{RunEvery: "7s"}
	if scheduler.RunEveryDuration() != 7*time.Second {
		t.Fatalf("run every parse mismatch")
	}
	scheduler.RunEvery = "bad"
	if scheduler.RunEveryDuration() != 30*time.Second {
		t.Fatalf("run every fallback mismatch")
	}
}

// TestSetDefaults 用于单元测试。
func TestSetDefaults(t *testing.T) {
	t.Parallel()

	v := viper.New()
	setDefaults(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal default config failed: %v", err)
	}

	if cfg.App.Name != "agent-coder" || cfg.Server.Port != 8080 {
		t.Fatalf("unexpected defaults: app=%s port=%d", cfg.App.Name, cfg.Server.Port)
	}
	if cfg.Agent.Codex.TimeoutSec != 7200 || cfg.Scheduler.PollIntervalSec != 30 {
		t.Fatalf("unexpected codex/scheduler defaults")
	}
}

// TestCurrentReplace 用于单元测试。
func TestCurrentReplace(t *testing.T) {
	old := Current()
	defer Replace(old)

	cfg := newValidConfig()
	Replace(&cfg)

	got := Current()
	if got == nil {
		t.Fatalf("Current should not be nil")
	}
	if got.Bootstrap.AdminUsername != cfg.Bootstrap.AdminUsername {
		t.Fatalf("replace/current mismatch")
	}

	Replace(nil)
	if Current() == nil {
		t.Fatalf("Replace(nil) should keep existing config")
	}
}

// TestLoad 用于单元测试。
func TestLoad(t *testing.T) {
	old := Current()
	defer Replace(old)

	t.Run("load success", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		raw := []byte(`app:
  name: agent-coder-test
server:
  port: 18080
`)
		if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
			t.Fatalf("write config file failed: %v", err)
		}
		cfg, err := Load(cfgPath)
		if err != nil {
			t.Fatalf("Load should succeed, got: %v", err)
		}
		if cfg.App.Name != "agent-coder-test" || cfg.Server.Port != 18080 {
			t.Fatalf("unexpected loaded config: %#v", cfg.Server)
		}
		if Current() == nil {
			t.Fatalf("Current should be set after Load")
		}
	})

	t.Run("load missing file", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
		if err == nil || !strings.Contains(err.Error(), "read config file") {
			t.Fatalf("expected read config file error, got: %v", err)
		}
	})

	t.Run("load validation error", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "bad.yaml")
		raw := []byte(`server:
  port: 70000
`)
		if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
			t.Fatalf("write bad config failed: %v", err)
		}
		_, err := Load(cfgPath)
		if err == nil || !strings.Contains(err.Error(), "server.port") {
			t.Fatalf("expected validation error for port, got: %v", err)
		}
	})
}

// newValidConfig 构造测试使用的最小可用配置。
func newValidConfig() Config {
	return Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            8080,
			ReadTimeout:     "15s",
			WriteTimeout:    "15s",
			ShutdownTimeout: "10s",
		},
		DB: DBConfig{
			Enabled:         true,
			Driver:          "sqlite",
			SQLitePath:      "agent-coder.db",
			ConnMaxLifetime: "30m",
		},
		Secret: SecretConfig{
			Provider: "env",
		},
		Auth: AuthConfig{
			SessionTTL: "72h",
		},
		Work: WorkConfig{
			WorkDir: ".agent-coder/workdirs",
		},
		Agent: AgentConfig{
			Codex: AgentCodexConfig{
				TimeoutSec:  7200,
				MaxRetry:    5,
				MaxLoopStep: 5,
			},
		},
		Bootstrap: BootstrapConfig{
			AdminUsername: "admin",
			AdminPassword: "admin123",
		},
	}
}
