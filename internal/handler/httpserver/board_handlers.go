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

// boardProjectItem 表示数据结构定义。
type boardProjectItem struct {
	ID            uint   `json:"id"`             // ID 字段说明。
	ProjectKey    string `json:"project_key"`    // ProjectKey 字段说明。
	ProjectSlug   string `json:"project_slug"`   // ProjectSlug 字段说明。
	Name          string `json:"name"`           // Name 字段说明。
	Provider      string `json:"provider"`       // Provider 字段说明。
	DefaultBranch string `json:"default_branch"` // DefaultBranch 字段说明。
	Enabled       bool   `json:"enabled"`        // Enabled 字段说明。
}

// boardIssueItem 表示数据结构定义。
type boardIssueItem struct {
	ID              uint       `json:"id"`                     // ID 字段说明。
	IssueIID        int64      `json:"issue_iid"`              // IssueIID 字段说明。
	Title           string     `json:"title"`                  // Title 字段说明。
	State           string     `json:"state"`                  // State 字段说明。
	LifecycleStatus string     `json:"lifecycle_status"`       // LifecycleStatus 字段说明。
	CloseReason     *string    `json:"close_reason,omitempty"` // CloseReason 字段说明。
	BranchName      *string    `json:"branch_name,omitempty"`  // BranchName 字段说明。
	MRIID           *int64     `json:"mr_iid,omitempty"`       // MRIID 字段说明。
	MRURL           *string    `json:"mr_url,omitempty"`       // MRURL 字段说明。
	LastSyncedAt    time.Time  `json:"last_synced_at"`         // LastSyncedAt 字段说明。
	UpdatedAt       time.Time  `json:"updated_at"`             // UpdatedAt 字段说明。
	ClosedAt        *time.Time `json:"closed_at,omitempty"`    // ClosedAt 字段说明。
}

// boardProjects 是 *Server 的方法实现。
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

// boardProjectIssues 是 *Server 的方法实现。
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

// toBoardIssueItem 执行相关逻辑。
func toBoardIssueItem(row db.Issue) boardIssueItem {
	return boardIssueItem{
		ID:              row.ID,
		IssueIID:        row.IssueIID,
		Title:           row.Title,
		State:           row.State,
		LifecycleStatus: row.LifecycleStatus,
		CloseReason:     row.CloseReason,
		BranchName:      row.BranchName,
		MRIID:           row.MRIID,
		MRURL:           row.MRURL,
		LastSyncedAt:    row.LastSyncedAt,
		UpdatedAt:       row.UpdatedAt,
		ClosedAt:        row.ClosedAt,
	}
}
