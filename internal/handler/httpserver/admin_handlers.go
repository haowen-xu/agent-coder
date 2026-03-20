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

// adminCreateUserRequest 表示数据结构定义。
type adminCreateUserRequest struct {
	Username string `json:"username"` // Username 字段说明。
	Password string `json:"password"` // Password 字段说明。
	IsAdmin  bool   `json:"is_admin"` // IsAdmin 字段说明。
	Enabled  *bool  `json:"enabled"`  // Enabled 字段说明。
}

// adminUpdateUserRequest 表示数据结构定义。
type adminUpdateUserRequest struct {
	Password *string `json:"password"` // Password 字段说明。
	IsAdmin  *bool   `json:"is_admin"` // IsAdmin 字段说明。
	Enabled  *bool   `json:"enabled"`  // Enabled 字段说明。
}

// adminListUsers 是 *Server 的方法实现。
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

// adminCreateUser 是 *Server 的方法实现。
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

// adminUpdateUser 是 *Server 的方法实现。
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

// adminProjectRequest 表示数据结构定义。
type adminProjectRequest struct {
	ProjectKey       string  `json:"project_key"`        // ProjectKey 字段说明。
	ProjectSlug      string  `json:"project_slug"`       // ProjectSlug 字段说明。
	Name             string  `json:"name"`               // Name 字段说明。
	Provider         string  `json:"provider"`           // Provider 字段说明。
	ProviderURL      string  `json:"provider_url"`       // ProviderURL 字段说明。
	RepoURL          string  `json:"repo_url"`           // RepoURL 字段说明。
	DefaultBranch    string  `json:"default_branch"`     // DefaultBranch 字段说明。
	IssueProjectID   *string `json:"issue_project_id"`   // IssueProjectID 字段说明。
	CredentialRef    string  `json:"credential_ref"`     // CredentialRef 字段说明。
	ProjectToken     *string `json:"project_token"`      // ProjectToken 字段说明。
	PollIntervalSec  int     `json:"poll_interval_sec"`  // PollIntervalSec 字段说明。
	Enabled          *bool   `json:"enabled"`            // Enabled 字段说明。
	LabelAgentReady  string  `json:"label_agent_ready"`  // LabelAgentReady 字段说明。
	LabelInProgress  string  `json:"label_in_progress"`  // LabelInProgress 字段说明。
	LabelHumanReview string  `json:"label_human_review"` // LabelHumanReview 字段说明。
	LabelRework      string  `json:"label_rework"`       // LabelRework 字段说明。
	LabelVerified    string  `json:"label_verified"`     // LabelVerified 字段说明。
	LabelMerged      string  `json:"label_merged"`       // LabelMerged 字段说明。
}

// adminProjectItem 表示数据结构定义。
type adminProjectItem struct {
	ID               uint       `json:"id"`                           // ID 字段说明。
	ProjectKey       string     `json:"project_key"`                  // ProjectKey 字段说明。
	ProjectSlug      string     `json:"project_slug"`                 // ProjectSlug 字段说明。
	Name             string     `json:"name"`                         // Name 字段说明。
	Provider         string     `json:"provider"`                     // Provider 字段说明。
	ProviderURL      string     `json:"provider_url"`                 // ProviderURL 字段说明。
	RepoURL          string     `json:"repo_url"`                     // RepoURL 字段说明。
	DefaultBranch    string     `json:"default_branch"`               // DefaultBranch 字段说明。
	IssueProjectID   *string    `json:"issue_project_id,omitempty"`   // IssueProjectID 字段说明。
	CredentialRef    string     `json:"credential_ref"`               // CredentialRef 字段说明。
	ProjectToken     *string    `json:"project_token,omitempty"`      // ProjectToken 字段说明。
	PollIntervalSec  int        `json:"poll_interval_sec"`            // PollIntervalSec 字段说明。
	Enabled          bool       `json:"enabled"`                      // Enabled 字段说明。
	LastIssueSyncAt  *time.Time `json:"last_issue_sync_at,omitempty"` // LastIssueSyncAt 字段说明。
	LabelAgentReady  string     `json:"label_agent_ready"`            // LabelAgentReady 字段说明。
	LabelInProgress  string     `json:"label_in_progress"`            // LabelInProgress 字段说明。
	LabelHumanReview string     `json:"label_human_review"`           // LabelHumanReview 字段说明。
	LabelRework      string     `json:"label_rework"`                 // LabelRework 字段说明。
	LabelVerified    string     `json:"label_verified"`               // LabelVerified 字段说明。
	LabelMerged      string     `json:"label_merged"`                 // LabelMerged 字段说明。
	CreatedBy        uint       `json:"created_by"`                   // CreatedBy 字段说明。
	CreatedAt        time.Time  `json:"created_at"`                   // CreatedAt 字段说明。
	UpdatedAt        time.Time  `json:"updated_at"`                   // UpdatedAt 字段说明。
}

// adminListProjects 是 *Server 的方法实现。
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

// adminCreateProject 是 *Server 的方法实现。
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
		ProjectToken:     req.ProjectToken,
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

// adminUpdateProject 是 *Server 的方法实现。
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
		ProjectToken:     req.ProjectToken,
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

// putPromptRequest 表示数据结构定义。
type putPromptRequest struct {
	Content string `json:"content"` // Content 字段说明。
}

// adminRunItem 表示数据结构定义。
type adminRunItem struct {
	ID                 uint       `json:"id"`                      // ID 字段说明。
	IssueID            uint       `json:"issue_id"`                // IssueID 字段说明。
	RunNo              int        `json:"run_no"`                  // RunNo 字段说明。
	RunKind            string     `json:"run_kind"`                // RunKind 字段说明。
	TriggerType        string     `json:"trigger_type"`            // TriggerType 字段说明。
	Status             string     `json:"status"`                  // Status 字段说明。
	AgentRole          string     `json:"agent_role"`              // AgentRole 字段说明。
	LoopStep           int        `json:"loop_step"`               // LoopStep 字段说明。
	MaxLoopStep        int        `json:"max_loop_step"`           // MaxLoopStep 字段说明。
	QueuedAt           time.Time  `json:"queued_at"`               // QueuedAt 字段说明。
	StartedAt          *time.Time `json:"started_at,omitempty"`    // StartedAt 字段说明。
	FinishedAt         *time.Time `json:"finished_at,omitempty"`   // FinishedAt 字段说明。
	BranchName         string     `json:"branch_name"`             // BranchName 字段说明。
	MRIID              *int64     `json:"mr_iid,omitempty"`        // MRIID 字段说明。
	MRURL              *string    `json:"mr_url,omitempty"`        // MRURL 字段说明。
	ConflictRetryCount int        `json:"conflict_retry_count"`    // ConflictRetryCount 字段说明。
	MaxConflictRetry   int        `json:"max_conflict_retry"`      // MaxConflictRetry 字段说明。
	ErrorSummary       *string    `json:"error_summary,omitempty"` // ErrorSummary 字段说明。
	CreatedAt          time.Time  `json:"created_at"`              // CreatedAt 字段说明。
	UpdatedAt          time.Time  `json:"updated_at"`              // UpdatedAt 字段说明。
}

// adminRunLogItem 表示数据结构定义。
type adminRunLogItem struct {
	ID          uint      `json:"id"`                     // ID 字段说明。
	RunID       uint      `json:"run_id"`                 // RunID 字段说明。
	Seq         int       `json:"seq"`                    // Seq 字段说明。
	At          time.Time `json:"at"`                     // At 字段说明。
	Level       string    `json:"level"`                  // Level 字段说明。
	Stage       string    `json:"stage"`                  // Stage 字段说明。
	EventType   string    `json:"event_type"`             // EventType 字段说明。
	Message     string    `json:"message"`                // Message 字段说明。
	PayloadJSON *string   `json:"payload_json,omitempty"` // PayloadJSON 字段说明。
}

// adminRunActionRequest 表示数据结构定义。
type adminRunActionRequest struct {
	Reason string `json:"reason"` // Reason 字段说明。
}

// listDefaultPrompts 是 *Server 的方法实现。
func (s *Server) listDefaultPrompts(_ context.Context, c *app.RequestContext) {
	rows, err := s.svc.ListDefaultPrompts()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(c, map[string]any{"items": rows})
}

// listProjectPrompts 是 *Server 的方法实现。
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

// putProjectPrompt 是 *Server 的方法实现。
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

// deleteProjectPrompt 是 *Server 的方法实现。
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

// adminProjectIssues 是 *Server 的方法实现。
func (s *Server) adminProjectIssues(ctx context.Context, c *app.RequestContext) {
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
	rows, err := s.svc.ListProjectIssues(ctx, projectKey, limit)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]boardIssueItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toBoardIssueItem(row))
	}
	writeOK(c, map[string]any{
		"project_key": projectKey,
		"items":       out,
	})
}

// adminIssueRuns 是 *Server 的方法实现。
func (s *Server) adminIssueRuns(ctx context.Context, c *app.RequestContext) {
	issueID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("issueID")), 10, 32)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid issue id")
		return
	}
	limit := 100
	if q := strings.TrimSpace(string(c.Query("limit"))); q != "" {
		if n, parseErr := strconv.Atoi(q); parseErr == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	rows, err := s.svc.ListIssueRuns(ctx, uint(issueID64), limit)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]adminRunItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAdminRunItem(row))
	}
	writeOK(c, map[string]any{
		"issue_id": uint(issueID64),
		"items":    out,
	})
}

// adminRunLogs 是 *Server 的方法实现。
func (s *Server) adminRunLogs(ctx context.Context, c *app.RequestContext) {
	runID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("runID")), 10, 32)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid run id")
		return
	}
	limit := 500
	if q := strings.TrimSpace(string(c.Query("limit"))); q != "" {
		if n, parseErr := strconv.Atoi(q); parseErr == nil && n > 0 && n <= 5000 {
			limit = n
		}
	}
	rows, err := s.svc.ListRunLogs(ctx, uint(runID64), limit)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]adminRunLogItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, adminRunLogItem{
			ID:          row.ID,
			RunID:       row.RunID,
			Seq:         row.Seq,
			At:          row.At,
			Level:       row.Level,
			Stage:       row.Stage,
			EventType:   row.EventType,
			Message:     row.Message,
			PayloadJSON: row.PayloadJSON,
		})
	}
	writeOK(c, map[string]any{
		"run_id": uint(runID64),
		"items":  out,
	})
}

// adminRetryIssue 是 *Server 的方法实现。
func (s *Server) adminRetryIssue(ctx context.Context, c *app.RequestContext) {
	issueID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("issueID")), 10, 32)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid issue id")
		return
	}
	row, err := s.svc.RetryIssue(ctx, uint(issueID64))
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, map[string]any{
		"issue_id":         row.ID,
		"lifecycle_status": row.LifecycleStatus,
		"close_reason":     row.CloseReason,
		"retried":          true,
	})
}

// adminCancelRun 是 *Server 的方法实现。
func (s *Server) adminCancelRun(ctx context.Context, c *app.RequestContext) {
	runID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("runID")), 10, 32)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid run id")
		return
	}
	var req adminRunActionRequest
	if len(c.Request.Body()) > 0 {
		if err := bindJSON(c, &req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	row, err := s.svc.CancelRun(ctx, uint(runID64), req.Reason)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, toAdminRunItem(*row))
}

// adminResetProjectSyncCursor 是 *Server 的方法实现。
func (s *Server) adminResetProjectSyncCursor(ctx context.Context, c *app.RequestContext) {
	projectKey := strings.TrimSpace(c.Param("projectKey"))
	if projectKey == "" {
		writeError(c, http.StatusBadRequest, "projectKey is required")
		return
	}
	row, err := s.svc.ResetProjectSyncCursor(ctx, projectKey)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeOK(c, map[string]any{
		"project_key":        row.ProjectKey,
		"last_issue_sync_at": row.LastIssueSyncAt,
		"reset":              true,
	})
}

// adminMetrics 是 *Server 的方法实现。
func (s *Server) adminMetrics(ctx context.Context, c *app.RequestContext) {
	row, err := s.svc.GetOpsMetrics(ctx)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(c, row)
}

// toAdminProjectItem 执行相关逻辑。
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
		ProjectToken:     row.ProjectToken,
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

// toAdminRunItem 执行相关逻辑。
func toAdminRunItem(row db.IssueRun) adminRunItem {
	return adminRunItem{
		ID:                 row.ID,
		IssueID:            row.IssueID,
		RunNo:              row.RunNo,
		RunKind:            row.RunKind,
		TriggerType:        row.TriggerType,
		Status:             row.Status,
		AgentRole:          row.AgentRole,
		LoopStep:           row.LoopStep,
		MaxLoopStep:        row.MaxLoopStep,
		QueuedAt:           row.QueuedAt,
		StartedAt:          row.StartedAt,
		FinishedAt:         row.FinishedAt,
		BranchName:         row.BranchName,
		MRIID:              row.MRIID,
		MRURL:              row.MRURL,
		ConflictRetryCount: row.ConflictRetryCount,
		MaxConflictRetry:   row.MaxConflictRetry,
		ErrorSummary:       row.ErrorSummary,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}
