package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	"github.com/haowen-xu/agent-coder/internal/db"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
)

type Server struct {
	hz  *server.Hertz
	log *slog.Logger
	db  *db.Client
	cfg *appcfg.Config
	ps  *promptstore.Service
}

func New(cfg *appcfg.Config, log *slog.Logger, dbClient *db.Client, promptService *promptstore.Service) *Server {
	hz := server.New(
		server.WithHostPorts(cfg.Server.Address()),
		server.WithDisablePrintRoute(true),
		server.WithReadTimeout(cfg.Server.ReadTimeoutDuration()),
		server.WithWriteTimeout(cfg.Server.WriteTimeoutDuration()),
	)

	s := &Server{hz: hz, log: log, db: dbClient, cfg: cfg, ps: promptService}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.hz.GET("/healthz", s.healthz)
	s.hz.GET("/api/v1/meta", s.meta)
	s.hz.GET("/api/v1/admin/prompts/defaults", s.listDefaultPrompts)
	s.hz.GET("/api/v1/admin/projects/:projectKey/prompts", s.listProjectPrompts)
	s.hz.PUT("/api/v1/admin/projects/:projectKey/prompts/:runKind/:agentRole", s.putProjectPrompt)
	s.hz.DELETE("/api/v1/admin/projects/:projectKey/prompts/:runKind/:agentRole", s.deleteProjectPrompt)
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

func (s *Server) listDefaultPrompts(_ context.Context, c *app.RequestContext) {
	if s.ps == nil {
		c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "prompt service is not initialized"})
		return
	}

	rows, err := s.ps.ListDefaults()
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]any{"items": rows})
}

func (s *Server) listProjectPrompts(ctx context.Context, c *app.RequestContext) {
	if s.ps == nil {
		c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "prompt service is not initialized"})
		return
	}

	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "projectKey is required"})
		return
	}

	rows, err := s.ps.ListEffectiveByProject(ctx, projectKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]any{
		"project_key": projectKey,
		"items":       rows,
	})
}

type putPromptRequest struct {
	Content string `json:"content"`
}

func (s *Server) putProjectPrompt(ctx context.Context, c *app.RequestContext) {
	if s.ps == nil {
		c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "prompt service is not initialized"})
		return
	}

	projectKey := strings.TrimSpace(c.Param("projectKey"))
	runKind := strings.TrimSpace(c.Param("runKind"))
	agentRole := strings.TrimSpace(c.Param("agentRole"))
	if projectKey == "" || runKind == "" || agentRole == "" {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "projectKey/runKind/agentRole are required"})
		return
	}

	var req putPromptRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid json body"})
		return
	}

	row, err := s.ps.UpsertProjectOverride(ctx, projectKey, runKind, agentRole, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (s *Server) deleteProjectPrompt(ctx context.Context, c *app.RequestContext) {
	if s.ps == nil {
		c.JSON(http.StatusServiceUnavailable, map[string]any{"error": "prompt service is not initialized"})
		return
	}

	projectKey := strings.TrimSpace(c.Param("projectKey"))
	runKind := strings.TrimSpace(c.Param("runKind"))
	agentRole := strings.TrimSpace(c.Param("agentRole"))
	if projectKey == "" || runKind == "" || agentRole == "" {
		c.JSON(http.StatusBadRequest, map[string]any{"error": "projectKey/runKind/agentRole are required"})
		return
	}

	if err := s.ps.DeleteProjectOverride(ctx, projectKey, runKind, agentRole); err != nil {
		c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"project_key": projectKey,
		"run_kind":    runKind,
		"agent_role":  agentRole,
		"deleted":     true,
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
