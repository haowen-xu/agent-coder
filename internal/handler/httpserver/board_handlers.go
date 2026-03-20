package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

func (s *Server) boardProjects(ctx context.Context, c *app.RequestContext) {
	rows, err := s.svc.ListProjects(ctx)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(c, map[string]any{"items": rows})
}

func (s *Server) boardProjectIssues(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		writeError(c, http.StatusBadRequest, "projectKey is required")
		return
	}
	limit := 100
	if q := strings.TrimSpace(string(c.Query("limit"))); q != "" {
		if n, parseErr := strconv.Atoi(q); parseErr == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	issues, err := s.svc.ListProjectIssues(ctx, projectKey, limit)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, map[string]any{
		"project_key": projectKey,
		"items":       issues,
	})
}
