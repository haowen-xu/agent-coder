package httpserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	db "github.com/haowen-xu/agent-coder/internal/dal"
)

type boardProjectItem struct {
	ID            uint   `json:"id"`
	ProjectKey    string `json:"project_key"`
	ProjectSlug   string `json:"project_slug"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	DefaultBranch string `json:"default_branch"`
	Enabled       bool   `json:"enabled"`
}

type boardIssueItem struct {
	ID              uint       `json:"id"`
	IssueIID        int64      `json:"issue_iid"`
	Title           string     `json:"title"`
	State           string     `json:"state"`
	LifecycleStatus string     `json:"lifecycle_status"`
	BranchName      *string    `json:"branch_name,omitempty"`
	MRIID           *int64     `json:"mr_iid,omitempty"`
	MRURL           *string    `json:"mr_url,omitempty"`
	LastSyncedAt    time.Time  `json:"last_synced_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
}

func (s *Server) boardProjects(ctx context.Context, c *app.RequestContext) {
	rows, err := s.svc.ListProjects(ctx)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]boardProjectItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, boardProjectItem{
			ID:            row.ID,
			ProjectKey:    row.ProjectKey,
			ProjectSlug:   row.ProjectSlug,
			Name:          row.Name,
			Provider:      row.Provider,
			DefaultBranch: row.DefaultBranch,
			Enabled:       row.Enabled,
		})
	}
	writeOK(c, map[string]any{"items": out})
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
	out := make([]boardIssueItem, 0, len(issues))
	for _, row := range issues {
		out = append(out, toBoardIssueItem(row))
	}
	writeOK(c, map[string]any{
		"project_key": projectKey,
		"items":       out,
	})
}

func toBoardIssueItem(row db.Issue) boardIssueItem {
	return boardIssueItem{
		ID:              row.ID,
		IssueIID:        row.IssueIID,
		Title:           row.Title,
		State:           row.State,
		LifecycleStatus: row.LifecycleStatus,
		BranchName:      row.BranchName,
		MRIID:           row.MRIID,
		MRURL:           row.MRURL,
		LastSyncedAt:    row.LastSyncedAt,
		UpdatedAt:       row.UpdatedAt,
		ClosedAt:        row.ClosedAt,
	}
}
