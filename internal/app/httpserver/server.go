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
	db "github.com/haowen-xu/agent-coder/internal/dal"
	authhandler "github.com/haowen-xu/agent-coder/internal/handler/auth"
	issuehandler "github.com/haowen-xu/agent-coder/internal/handler/issue"
	issuerunhandler "github.com/haowen-xu/agent-coder/internal/handler/issue_run"
	opshandler "github.com/haowen-xu/agent-coder/internal/handler/ops"
	projecthandler "github.com/haowen-xu/agent-coder/internal/handler/project"
	userhandler "github.com/haowen-xu/agent-coder/internal/handler/user"
	issuesvc "github.com/haowen-xu/agent-coder/internal/service/issue"
	issuerunsvc "github.com/haowen-xu/agent-coder/internal/service/issue_run"
	opssvc "github.com/haowen-xu/agent-coder/internal/service/ops"
	projectsvc "github.com/haowen-xu/agent-coder/internal/service/project"
	usersvc "github.com/haowen-xu/agent-coder/internal/service/user"
	"github.com/haowen-xu/agent-coder/internal/utils"
)

// Server 表示数据结构定义。
type Server struct {
	hz *server.Hertz // hz 字段说明。

	log *slog.Logger   // log 字段说明。
	db  *db.Client     // db 字段说明。
	cfg *appcfg.Config // cfg 字段说明。

	authHandler     *authhandler.Handler     // authHandler 字段说明。
	userHandler     *userhandler.Handler     // userHandler 字段说明。
	projectHandler  *projecthandler.Handler  // projectHandler 字段说明。
	issueHandler    *issuehandler.Handler    // issueHandler 字段说明。
	issueRunHandler *issuerunhandler.Handler // issueRunHandler 字段说明。
	opsHandler      *opshandler.Handler      // opsHandler 字段说明。
}

// New 执行相关逻辑。
func New(
	cfg *appcfg.Config,
	log *slog.Logger,
	dbClient *db.Client,
	userSvc *usersvc.Service,
	projectSvc *projectsvc.Service,
	issueSvc *issuesvc.Service,
	issueRunSvc *issuerunsvc.Service,
	opsSvc *opssvc.Service,
) *Server {
	hz := server.New(
		server.WithHostPorts(cfg.Server.Address()),
		server.WithDisablePrintRoute(true),
		server.WithReadTimeout(cfg.Server.ReadTimeoutDuration()),
		server.WithWriteTimeout(cfg.Server.WriteTimeoutDuration()),
	)

	s := &Server{
		hz:              hz,
		log:             log,
		db:              dbClient,
		cfg:             cfg,
		authHandler:     authhandler.New(userSvc),
		userHandler:     userhandler.New(userSvc),
		projectHandler:  projecthandler.New(projectSvc),
		issueHandler:    issuehandler.New(issueSvc),
		issueRunHandler: issuerunhandler.New(issueRunSvc),
		opsHandler:      opshandler.New(opsSvc),
	}
	s.registerRoutes()
	return s
}

// registerRoutes 是 *Server 的方法实现。
func (s *Server) registerRoutes() {
	s.hz.Use(s.servePrecompressedStaticGzip())

	s.hz.GET("/healthz", s.healthz)
	s.hz.GET("/api/v1/meta", s.meta)
	s.hz.POST("/api/v1/auth/login", s.authHandler.Login)

	api := s.hz.Group("/api/v1")
	api.Use(s.authHandler.RequireLogin())
	api.GET("/auth/me", s.authHandler.Me)

	board := api.Group("/board")
	board.GET("/projects", s.projectHandler.BoardProjects)
	board.GET("/projects/:projectKey/issues", s.projectHandler.BoardProjectIssues)

	admin := api.Group("/admin")
	admin.Use(s.authHandler.RequireAdmin())

	admin.GET("/users", s.userHandler.AdminListUsers)
	admin.POST("/users", s.userHandler.AdminCreateUser)
	admin.PUT("/users/:id", s.userHandler.AdminUpdateUser)

	admin.GET("/projects", s.projectHandler.AdminListProjects)
	admin.POST("/projects", s.projectHandler.AdminCreateProject)
	admin.PUT("/projects/:projectKey", s.projectHandler.AdminUpdateProject)
	admin.POST("/projects/:projectKey/reset-sync-cursor", s.projectHandler.AdminResetProjectSyncCursor)
	admin.GET("/projects/:projectKey/issues", s.projectHandler.AdminProjectIssues)

	admin.GET("/prompts/defaults", s.projectHandler.ListDefaultPrompts)
	admin.GET("/projects/:projectKey/prompts", s.projectHandler.ListProjectPrompts)
	admin.PUT("/projects/:projectKey/prompts/:runKind/:agentRole", s.projectHandler.PutProjectPrompt)
	admin.DELETE("/projects/:projectKey/prompts/:runKind/:agentRole", s.projectHandler.DeleteProjectPrompt)

	admin.GET("/issues/:issueID/runs", s.issueRunHandler.AdminIssueRuns)
	admin.POST("/issues/:issueID/retry", s.issueHandler.AdminRetryIssue)
	admin.GET("/runs/:runID/logs", s.issueRunHandler.AdminRunLogs)
	admin.POST("/runs/:runID/cancel", s.issueRunHandler.AdminCancelRun)
	admin.GET("/metrics", s.opsHandler.AdminMetrics)

	s.registerStaticRoutes()
}

// healthz 是 *Server 的方法实现。
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

// meta 是 *Server 的方法实现。
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
		"now": utils.NowUTC().Format(time.RFC3339),
	})
}

// Run 是 *Server 的方法实现。
func (s *Server) Run() error {
	s.log.Info("http server starting", slog.String("addr", s.cfg.Server.Address()))
	return s.hz.Run()
}

// Shutdown 是 *Server 的方法实现。
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")
	return s.hz.Shutdown(ctx)
}
