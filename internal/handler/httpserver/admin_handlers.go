package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	coresvc "github.com/haowen-xu/agent-coder/internal/service/core"
)

type adminCreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
	Enabled  *bool  `json:"enabled"`
}

type adminUpdateUserRequest struct {
	Password *string `json:"password"`
	IsAdmin  *bool   `json:"is_admin"`
	Enabled  *bool   `json:"enabled"`
}

func (s *Server) adminListUsers(ctx context.Context, c *app.RequestContext) {
	rows, err := s.svc.ListUsers(ctx)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":            row.ID,
			"username":      row.Username,
			"is_admin":      row.IsAdmin,
			"enabled":       row.Enabled,
			"last_login_at": row.LastLoginAt,
			"created_at":    row.CreatedAt,
			"updated_at":    row.UpdatedAt,
		})
	}
	writeOK(c, map[string]any{"items": out})
}

func (s *Server) adminCreateUser(ctx context.Context, c *app.RequestContext) {
	var req adminCreateUserRequest
	if err := bindJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json body")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row, err := s.svc.CreateUser(ctx, req.Username, req.Password, req.IsAdmin, enabled)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeOK(c, map[string]any{
		"id":       row.ID,
		"username": row.Username,
		"is_admin": row.IsAdmin,
		"enabled":  row.Enabled,
	})
}

func (s *Server) adminUpdateUser(ctx context.Context, c *app.RequestContext) {
	id64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 32)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req adminUpdateUserRequest
	if err := bindJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json body")
		return
	}

	row, err := s.svc.UpdateUser(ctx, uint(id64), req.Password, req.IsAdmin, req.Enabled)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	writeOK(c, map[string]any{
		"id":       row.ID,
		"username": row.Username,
		"is_admin": row.IsAdmin,
		"enabled":  row.Enabled,
	})
}

type adminProjectRequest struct {
	ProjectKey       string  `json:"project_key"`
	ProjectSlug      string  `json:"project_slug"`
	Name             string  `json:"name"`
	Provider         string  `json:"provider"`
	ProviderURL      string  `json:"provider_url"`
	RepoURL          string  `json:"repo_url"`
	DefaultBranch    string  `json:"default_branch"`
	IssueProjectID   *string `json:"issue_project_id"`
	CredentialRef    string  `json:"credential_ref"`
	PollIntervalSec  int     `json:"poll_interval_sec"`
	Enabled          *bool   `json:"enabled"`
	LabelAgentReady  string  `json:"label_agent_ready"`
	LabelInProgress  string  `json:"label_in_progress"`
	LabelHumanReview string  `json:"label_human_review"`
	LabelRework      string  `json:"label_rework"`
	LabelVerified    string  `json:"label_verified"`
	LabelMerged      string  `json:"label_merged"`
}

type adminProjectItem struct {
	ID               uint       `json:"id"`
	ProjectKey       string     `json:"project_key"`
	ProjectSlug      string     `json:"project_slug"`
	Name             string     `json:"name"`
	Provider         string     `json:"provider"`
	ProviderURL      string     `json:"provider_url"`
	RepoURL          string     `json:"repo_url"`
	DefaultBranch    string     `json:"default_branch"`
	IssueProjectID   *string    `json:"issue_project_id,omitempty"`
	CredentialRef    string     `json:"credential_ref"`
	PollIntervalSec  int        `json:"poll_interval_sec"`
	Enabled          bool       `json:"enabled"`
	LastIssueSyncAt  *time.Time `json:"last_issue_sync_at,omitempty"`
	LabelAgentReady  string     `json:"label_agent_ready"`
	LabelInProgress  string     `json:"label_in_progress"`
	LabelHumanReview string     `json:"label_human_review"`
	LabelRework      string     `json:"label_rework"`
	LabelVerified    string     `json:"label_verified"`
	LabelMerged      string     `json:"label_merged"`
	CreatedBy        uint       `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (s *Server) adminListProjects(ctx context.Context, c *app.RequestContext) {
	rows, err := s.svc.ListProjects(ctx)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]adminProjectItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAdminProjectItem(row))
	}
	writeOK(c, map[string]any{"items": out})
}

func (s *Server) adminCreateProject(ctx context.Context, c *app.RequestContext) {
	var req adminProjectRequest
	if err := bindJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json body")
		return
	}
	user := currentUser(c)
	in := coresvc.ProjectUpsertInput{
		ProjectKey:       req.ProjectKey,
		ProjectSlug:      req.ProjectSlug,
		Name:             req.Name,
		Provider:         req.Provider,
		ProviderURL:      req.ProviderURL,
		RepoURL:          req.RepoURL,
		DefaultBranch:    req.DefaultBranch,
		IssueProjectID:   req.IssueProjectID,
		CredentialRef:    req.CredentialRef,
		PollIntervalSec:  req.PollIntervalSec,
		LabelAgentReady:  req.LabelAgentReady,
		LabelInProgress:  req.LabelInProgress,
		LabelHumanReview: req.LabelHumanReview,
		LabelRework:      req.LabelRework,
		LabelVerified:    req.LabelVerified,
		LabelMerged:      req.LabelMerged,
	}
	if req.Enabled != nil {
		in.Enabled = *req.Enabled
	} else {
		in.Enabled = true
	}

	row, err := s.svc.CreateProject(ctx, user.ID, in)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, toAdminProjectItem(*row))
}

func (s *Server) adminUpdateProject(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		writeError(c, http.StatusBadRequest, "projectKey is required")
		return
	}

	var req adminProjectRequest
	if err := bindJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json body")
		return
	}
	in := coresvc.ProjectUpsertInput{
		ProjectKey:       req.ProjectKey,
		ProjectSlug:      req.ProjectSlug,
		Name:             req.Name,
		Provider:         req.Provider,
		ProviderURL:      req.ProviderURL,
		RepoURL:          req.RepoURL,
		DefaultBranch:    req.DefaultBranch,
		IssueProjectID:   req.IssueProjectID,
		CredentialRef:    req.CredentialRef,
		PollIntervalSec:  req.PollIntervalSec,
		LabelAgentReady:  req.LabelAgentReady,
		LabelInProgress:  req.LabelInProgress,
		LabelHumanReview: req.LabelHumanReview,
		LabelRework:      req.LabelRework,
		LabelVerified:    req.LabelVerified,
		LabelMerged:      req.LabelMerged,
	}
	if req.Enabled != nil {
		in.Enabled = *req.Enabled
	}

	row, err := s.svc.UpdateProject(ctx, projectKey, in)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, toAdminProjectItem(*row))
}

type putPromptRequest struct {
	Content string `json:"content"`
}

func (s *Server) listDefaultPrompts(_ context.Context, c *app.RequestContext) {
	rows, err := s.svc.ListDefaultPrompts()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(c, map[string]any{"items": rows})
}

func (s *Server) listProjectPrompts(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		writeError(c, http.StatusBadRequest, "projectKey is required")
		return
	}

	rows, err := s.svc.ListProjectPrompts(ctx, projectKey)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, map[string]any{
		"project_key": projectKey,
		"items":       rows,
	})
}

func (s *Server) putProjectPrompt(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	runKind := strings.TrimSpace(c.Param("runKind"))
	agentRole := strings.TrimSpace(c.Param("agentRole"))
	if projectKey == "" || runKind == "" || agentRole == "" {
		writeError(c, http.StatusBadRequest, "projectKey/runKind/agentRole are required")
		return
	}

	var req putPromptRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json body")
		return
	}

	row, err := s.svc.UpsertProjectPrompt(ctx, projectKey, runKind, agentRole, req.Content)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, row)
}

func (s *Server) deleteProjectPrompt(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	runKind := strings.TrimSpace(c.Param("runKind"))
	agentRole := strings.TrimSpace(c.Param("agentRole"))
	if projectKey == "" || runKind == "" || agentRole == "" {
		writeError(c, http.StatusBadRequest, "projectKey/runKind/agentRole are required")
		return
	}

	if err := s.svc.DeleteProjectPrompt(ctx, projectKey, runKind, agentRole); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, map[string]any{
		"project_key": projectKey,
		"run_kind":    runKind,
		"agent_role":  agentRole,
		"deleted":     true,
	})
}

func toAdminProjectItem(row db.Project) adminProjectItem {
	return adminProjectItem{
		ID:               row.ID,
		ProjectKey:       row.ProjectKey,
		ProjectSlug:      row.ProjectSlug,
		Name:             row.Name,
		Provider:         row.Provider,
		ProviderURL:      row.ProviderURL,
		RepoURL:          row.RepoURL,
		DefaultBranch:    row.DefaultBranch,
		IssueProjectID:   row.IssueProjectID,
		CredentialRef:    row.CredentialRef,
		PollIntervalSec:  row.PollIntervalSec,
		Enabled:          row.Enabled,
		LastIssueSyncAt:  row.LastIssueSyncAt,
		LabelAgentReady:  row.LabelAgentReady,
		LabelInProgress:  row.LabelInProgress,
		LabelHumanReview: row.LabelHumanReview,
		LabelRework:      row.LabelRework,
		LabelVerified:    row.LabelVerified,
		LabelMerged:      row.LabelMerged,
		CreatedBy:        row.CreatedBy,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}
