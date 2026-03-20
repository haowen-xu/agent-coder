package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/service/core"
)

type Server struct {
	hz  *server.Hertz
	log *slog.Logger
	db  *db.Client
	cfg *appcfg.Config
	svc *core.Service
}

func New(cfg *appcfg.Config, log *slog.Logger, dbClient *db.Client, svc *core.Service) *Server {
	hz := server.New(
		server.WithHostPorts(cfg.Server.Address()),
		server.WithDisablePrintRoute(true),
		server.WithReadTimeout(cfg.Server.ReadTimeoutDuration()),
		server.WithWriteTimeout(cfg.Server.WriteTimeoutDuration()),
	)

	s := &Server{
		hz:  hz,
		log: log,
		db:  dbClient,
		cfg: cfg,
		svc: svc,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.hz.GET("/healthz", s.healthz)
	s.hz.GET("/api/v1/meta", s.meta)
	s.hz.POST("/api/v1/auth/login", s.login)

	api := s.hz.Group("/api/v1")
	api.Use(s.requireLogin())
	api.GET("/auth/me", s.me)

	board := api.Group("/board")
	board.GET("/projects", s.boardProjects)
	board.GET("/projects/:projectKey/issues", s.boardProjectIssues)

	admin := api.Group("/admin")
	admin.Use(s.requireAdmin())

	admin.GET("/users", s.adminListUsers)
	admin.POST("/users", s.adminCreateUser)
	admin.PUT("/users/:id", s.adminUpdateUser)

	admin.GET("/projects", s.adminListProjects)
	admin.POST("/projects", s.adminCreateProject)
	admin.PUT("/projects/:projectKey", s.adminUpdateProject)
	admin.POST("/projects/:projectKey/reset-sync-cursor", s.adminResetProjectSyncCursor)
	admin.GET("/projects/:projectKey/issues", s.adminProjectIssues)

	admin.GET("/prompts/defaults", s.listDefaultPrompts)
	admin.GET("/projects/:projectKey/prompts", s.listProjectPrompts)
	admin.PUT("/projects/:projectKey/prompts/:runKind/:agentRole", s.putProjectPrompt)
	admin.DELETE("/projects/:projectKey/prompts/:runKind/:agentRole", s.deleteProjectPrompt)

	admin.GET("/issues/:issueID/runs", s.adminIssueRuns)
	admin.POST("/issues/:issueID/retry", s.adminRetryIssue)
	admin.GET("/runs/:runID/logs", s.adminRunLogs)
	admin.POST("/runs/:runID/cancel", s.adminCancelRun)
	admin.GET("/metrics", s.adminMetrics)

	s.registerStaticRoutes()
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

	c.JSON(http.StatusOK, map[string]any{
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
