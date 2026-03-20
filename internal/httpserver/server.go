package httpserver

import (
	"context"
	"log/slog"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/db"
)

type Server struct {
	hz  *server.Hertz
	log *slog.Logger
	db  *db.Client
	cfg *appcfg.Config
}

func New(cfg *appcfg.Config, log *slog.Logger, dbClient *db.Client) *Server {
	hz := server.New(
		server.WithHostPorts(cfg.Server.Address()),
		server.WithDisablePrintRoute(true),
		server.WithReadTimeout(cfg.Server.ReadTimeoutDuration()),
		server.WithWriteTimeout(cfg.Server.WriteTimeoutDuration()),
	)

	s := &Server{hz: hz, log: log, db: dbClient, cfg: cfg}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.hz.GET("/healthz", s.healthz)
	s.hz.GET("/api/v1/meta", s.meta)
	s.hz.GET("/", func(_ context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "agent-coder backend is running")
	})
}

func (s *Server) healthz(ctx context.Context, c *app.RequestContext) {
	dbStatus := "disabled"
	if s.db != nil && s.db.Enabled() {
		dbStatus = "up"
		sqlDB := s.db.SQLDB()
		if sqlDB == nil {
			dbStatus = "down"
		} else if err := sqlDB.PingContext(ctx); err != nil {
			dbStatus = "down"
		}
	}

	statusCode := consts.StatusOK
	status := "ok"
	if dbStatus == "down" {
		statusCode = consts.StatusServiceUnavailable
		status = "degraded"
	}

	c.JSON(statusCode, map[string]any{
		"status": status,
		"db":     dbStatus,
	})
}

func (s *Server) meta(_ context.Context, c *app.RequestContext) {
	dialect := "disabled"
	if s.db != nil && s.db.Enabled() {
		dialect = s.db.Dialect()
	}
	c.JSON(consts.StatusOK, map[string]any{
		"app": map[string]any{
			"name": s.cfg.App.Name,
			"env":  s.cfg.App.Env,
		},
		"server": map[string]any{
			"addr": s.cfg.Server.Address(),
		},
		"db": map[string]any{
			"enabled": s.db != nil && s.db.Enabled(),
			"dialect": dialect,
		},
		"now": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) Run() error {
	s.log.Info("http server starting", slog.String("addr", s.cfg.Server.Address()))
	return s.hz.Run()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")
	return s.hz.Shutdown(ctx)
}
