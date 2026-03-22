package project

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	authhandler "github.com/haowen-xu/agent-coder/internal/handler/auth"
	"github.com/haowen-xu/agent-coder/internal/handler/httputil"
	projectsvc "github.com/haowen-xu/agent-coder/internal/service/project"
)

// Handler 表示数据结构定义。
type Handler struct {
	projectSvc *projectsvc.Service // projectSvc 字段说明。
}

// New 执行相关逻辑。
func New(projectSvc *projectsvc.Service) *Handler {
	return &Handler{projectSvc: projectSvc}
}

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

// adminProjectRequest 表示数据结构定义。
type adminProjectRequest struct {
	ProjectKey        string  `json:"project_key"`         // ProjectKey 字段说明。
	ProjectSlug       string  `json:"project_slug"`        // ProjectSlug 字段说明。
	Name              string  `json:"name"`                // Name 字段说明。
	Provider          string  `json:"provider"`            // Provider 字段说明。
	ProviderURL       string  `json:"provider_url"`        // ProviderURL 字段说明。
	RepoURL           string  `json:"repo_url"`            // RepoURL 字段说明。
	DefaultBranch     string  `json:"default_branch"`      // DefaultBranch 字段说明。
	IssueProjectID    *string `json:"issue_project_id"`    // IssueProjectID 字段说明。
	CredentialRef     string  `json:"credential_ref"`      // CredentialRef 字段说明。
	ProjectToken      *string `json:"project_token"`       // ProjectToken 字段说明。
	SandboxPlanReview bool    `json:"sandbox_plan_review"` // SandboxPlanReview 字段说明。
	PollIntervalSec   int     `json:"poll_interval_sec"`   // PollIntervalSec 字段说明。
	Enabled           *bool   `json:"enabled"`             // Enabled 字段说明。
	LabelAgentReady   string  `json:"label_agent_ready"`   // LabelAgentReady 字段说明。
	LabelInProgress   string  `json:"label_in_progress"`   // LabelInProgress 字段说明。
	LabelHumanReview  string  `json:"label_human_review"`  // LabelHumanReview 字段说明。
	LabelRework       string  `json:"label_rework"`        // LabelRework 字段说明。
	LabelVerified     string  `json:"label_verified"`      // LabelVerified 字段说明。
	LabelMerged       string  `json:"label_merged"`        // LabelMerged 字段说明。
}

// adminProjectItem 表示数据结构定义。
type adminProjectItem struct {
	ID                uint       `json:"id"`                           // ID 字段说明。
	ProjectKey        string     `json:"project_key"`                  // ProjectKey 字段说明。
	ProjectSlug       string     `json:"project_slug"`                 // ProjectSlug 字段说明。
	Name              string     `json:"name"`                         // Name 字段说明。
	Provider          string     `json:"provider"`                     // Provider 字段说明。
	ProviderURL       string     `json:"provider_url"`                 // ProviderURL 字段说明。
	RepoURL           string     `json:"repo_url"`                     // RepoURL 字段说明。
	DefaultBranch     string     `json:"default_branch"`               // DefaultBranch 字段说明。
	IssueProjectID    *string    `json:"issue_project_id,omitempty"`   // IssueProjectID 字段说明。
	CredentialRef     string     `json:"credential_ref"`               // CredentialRef 字段说明。
	ProjectToken      *string    `json:"project_token,omitempty"`      // ProjectToken 字段说明。
	SandboxPlanReview bool       `json:"sandbox_plan_review"`          // SandboxPlanReview 字段说明。
	PollIntervalSec   int        `json:"poll_interval_sec"`            // PollIntervalSec 字段说明。
	Enabled           bool       `json:"enabled"`                      // Enabled 字段说明。
	LastIssueSyncAt   *time.Time `json:"last_issue_sync_at,omitempty"` // LastIssueSyncAt 字段说明。
	LabelAgentReady   string     `json:"label_agent_ready"`            // LabelAgentReady 字段说明。
	LabelInProgress   string     `json:"label_in_progress"`            // LabelInProgress 字段说明。
	LabelHumanReview  string     `json:"label_human_review"`           // LabelHumanReview 字段说明。
	LabelRework       string     `json:"label_rework"`                 // LabelRework 字段说明。
	LabelVerified     string     `json:"label_verified"`               // LabelVerified 字段说明。
	LabelMerged       string     `json:"label_merged"`                 // LabelMerged 字段说明。
	CreatedBy         uint       `json:"created_by"`                   // CreatedBy 字段说明。
	CreatedAt         time.Time  `json:"created_at"`                   // CreatedAt 字段说明。
	UpdatedAt         time.Time  `json:"updated_at"`                   // UpdatedAt 字段说明。
}

// putPromptRequest 表示数据结构定义。
type putPromptRequest struct {
	Content string `json:"content"` // Content 字段说明。
}

// BoardProjects 是 *Handler 的方法实现。
func (h *Handler) BoardProjects(ctx context.Context, c *app.RequestContext) {
	rows, err := h.projectSvc.ListProjects(ctx)
	if err != nil {
		httputil.WriteError(c, http.StatusInternalServerError, err.Error())
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
	httputil.WriteOK(c, map[string]any{"items": out})
}

// BoardProjectIssues 是 *Handler 的方法实现。
func (h *Handler) BoardProjectIssues(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		httputil.WriteError(c, http.StatusBadRequest, "projectKey is required")
		return
	}
	limit := 100
	if q := strings.TrimSpace(string(c.Query("limit"))); q != "" {
		if n, parseErr := strconv.Atoi(q); parseErr == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	issues, err := h.projectSvc.ListProjectIssues(ctx, projectKey, limit)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]boardIssueItem, 0, len(issues))
	for _, row := range issues {
		out = append(out, toBoardIssueItem(row))
	}
	httputil.WriteOK(c, map[string]any{
		"project_key": projectKey,
		"items":       out,
	})
}

// AdminListProjects 是 *Handler 的方法实现。
func (h *Handler) AdminListProjects(ctx context.Context, c *app.RequestContext) {
	rows, err := h.projectSvc.ListProjects(ctx)
	if err != nil {
		httputil.WriteError(c, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]adminProjectItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAdminProjectItem(row))
	}
	httputil.WriteOK(c, map[string]any{"items": out})
}

// AdminCreateProject 是 *Handler 的方法实现。
func (h *Handler) AdminCreateProject(ctx context.Context, c *app.RequestContext) {
	var req adminProjectRequest
	if err := httputil.BindJSON(c, &req); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid json body")
		return
	}
	user := authhandler.CurrentUser(c)
	if user == nil {
		httputil.WriteError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	in := projectsvc.UpsertInput{
		ProjectKey:        req.ProjectKey,
		ProjectSlug:       req.ProjectSlug,
		Name:              req.Name,
		Provider:          req.Provider,
		ProviderURL:       req.ProviderURL,
		RepoURL:           req.RepoURL,
		DefaultBranch:     req.DefaultBranch,
		IssueProjectID:    req.IssueProjectID,
		CredentialRef:     req.CredentialRef,
		ProjectToken:      req.ProjectToken,
		SandboxPlanReview: req.SandboxPlanReview,
		PollIntervalSec:   req.PollIntervalSec,
		LabelAgentReady:   req.LabelAgentReady,
		LabelInProgress:   req.LabelInProgress,
		LabelHumanReview:  req.LabelHumanReview,
		LabelRework:       req.LabelRework,
		LabelVerified:     req.LabelVerified,
		LabelMerged:       req.LabelMerged,
	}
	if req.Enabled != nil {
		in.Enabled = *req.Enabled
	} else {
		in.Enabled = true
	}

	row, err := h.projectSvc.CreateProject(ctx, user.ID, in)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	httputil.WriteOK(c, toAdminProjectItem(*row))
}

// AdminUpdateProject 是 *Handler 的方法实现。
func (h *Handler) AdminUpdateProject(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		httputil.WriteError(c, http.StatusBadRequest, "projectKey is required")
		return
	}

	var req adminProjectRequest
	if err := httputil.BindJSON(c, &req); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid json body")
		return
	}
	in := projectsvc.UpsertInput{
		ProjectKey:        req.ProjectKey,
		ProjectSlug:       req.ProjectSlug,
		Name:              req.Name,
		Provider:          req.Provider,
		ProviderURL:       req.ProviderURL,
		RepoURL:           req.RepoURL,
		DefaultBranch:     req.DefaultBranch,
		IssueProjectID:    req.IssueProjectID,
		CredentialRef:     req.CredentialRef,
		ProjectToken:      req.ProjectToken,
		SandboxPlanReview: req.SandboxPlanReview,
		PollIntervalSec:   req.PollIntervalSec,
		LabelAgentReady:   req.LabelAgentReady,
		LabelInProgress:   req.LabelInProgress,
		LabelHumanReview:  req.LabelHumanReview,
		LabelRework:       req.LabelRework,
		LabelVerified:     req.LabelVerified,
		LabelMerged:       req.LabelMerged,
	}
	if req.Enabled != nil {
		in.Enabled = *req.Enabled
	}

	row, err := h.projectSvc.UpdateProject(ctx, projectKey, in)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	httputil.WriteOK(c, toAdminProjectItem(*row))
}

// AdminResetProjectSyncCursor 是 *Handler 的方法实现。
func (h *Handler) AdminResetProjectSyncCursor(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		httputil.WriteError(c, http.StatusBadRequest, "projectKey is required")
		return
	}
	row, err := h.projectSvc.ResetProjectSyncCursor(ctx, projectKey)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	httputil.WriteOK(c, map[string]any{
		"project_key":        row.ProjectKey,
		"last_issue_sync_at": row.LastIssueSyncAt,
		"reset":              true,
	})
}

// AdminProjectIssues 是 *Handler 的方法实现。
func (h *Handler) AdminProjectIssues(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		httputil.WriteError(c, http.StatusBadRequest, "projectKey is required")
		return
	}
	limit := 100
	if q := strings.TrimSpace(string(c.Query("limit"))); q != "" {
		if n, parseErr := strconv.Atoi(q); parseErr == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	rows, err := h.projectSvc.ListProjectIssues(ctx, projectKey, limit)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]boardIssueItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBoardIssueItem(row))
	}
	httputil.WriteOK(c, map[string]any{
		"project_key": projectKey,
		"items":       out,
	})
}

// ListDefaultPrompts 是 *Handler 的方法实现。
func (h *Handler) ListDefaultPrompts(_ context.Context, c *app.RequestContext) {
	rows, err := h.projectSvc.ListDefaultPrompts()
	if err != nil {
		httputil.WriteError(c, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.WriteOK(c, map[string]any{"items": rows})
}

// ListProjectPrompts 是 *Handler 的方法实现。
func (h *Handler) ListProjectPrompts(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		httputil.WriteError(c, http.StatusBadRequest, "projectKey is required")
		return
	}

	rows, err := h.projectSvc.ListProjectPrompts(ctx, projectKey)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	httputil.WriteOK(c, map[string]any{
		"project_key": projectKey,
		"items":       rows,
	})
}

// PutProjectPrompt 是 *Handler 的方法实现。
func (h *Handler) PutProjectPrompt(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	runKind := strings.TrimSpace(c.Param("runKind"))
	agentRole := strings.TrimSpace(c.Param("agentRole"))
	if projectKey == "" || runKind == "" || agentRole == "" {
		httputil.WriteError(c, http.StatusBadRequest, "projectKey/runKind/agentRole are required")
		return
	}

	var req putPromptRequest
	if err := json.Unmarshal(c.Request.Body(), &req); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid json body")
		return
	}

	row, err := h.projectSvc.UpsertProjectPrompt(ctx, projectKey, runKind, agentRole, req.Content)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	httputil.WriteOK(c, row)
}

// DeleteProjectPrompt 是 *Handler 的方法实现。
func (h *Handler) DeleteProjectPrompt(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	runKind := strings.TrimSpace(c.Param("runKind"))
	agentRole := strings.TrimSpace(c.Param("agentRole"))
	if projectKey == "" || runKind == "" || agentRole == "" {
		httputil.WriteError(c, http.StatusBadRequest, "projectKey/runKind/agentRole are required")
		return
	}

	if err := h.projectSvc.DeleteProjectPrompt(ctx, projectKey, runKind, agentRole); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	httputil.WriteOK(c, map[string]any{
		"project_key": projectKey,
		"run_kind":    runKind,
		"agent_role":  agentRole,
		"deleted":     true,
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

// toAdminProjectItem 执行相关逻辑。
func toAdminProjectItem(row db.Project) adminProjectItem {
	return adminProjectItem{
		ID:                row.ID,
		ProjectKey:        row.ProjectKey,
		ProjectSlug:       row.ProjectSlug,
		Name:              row.Name,
		Provider:          row.Provider,
		ProviderURL:       row.ProviderURL,
		RepoURL:           row.RepoURL,
		DefaultBranch:     row.DefaultBranch,
		IssueProjectID:    row.IssueProjectID,
		CredentialRef:     row.CredentialRef,
		ProjectToken:      row.ProjectToken,
		SandboxPlanReview: row.SandboxPlanReview,
		PollIntervalSec:   row.PollIntervalSec,
		Enabled:           row.Enabled,
		LastIssueSyncAt:   row.LastIssueSyncAt,
		LabelAgentReady:   row.LabelAgentReady,
		LabelInProgress:   row.LabelInProgress,
		LabelHumanReview:  row.LabelHumanReview,
		LabelRework:       row.LabelRework,
		LabelVerified:     row.LabelVerified,
		LabelMerged:       row.LabelMerged,
		CreatedBy:         row.CreatedBy,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
