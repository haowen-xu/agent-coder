package issuerun

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/handler/httputil"
	issuerunsvc "github.com/haowen-xu/agent-coder/internal/service/issue_run"
)

// Handler 表示数据结构定义。
type Handler struct {
	issueRunSvc *issuerunsvc.Service // issueRunSvc 字段说明。
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

// New 执行相关逻辑。
func New(issueRunSvc *issuerunsvc.Service) *Handler {
	return &Handler{issueRunSvc: issueRunSvc}
}

// AdminIssueRuns 是 *Handler 的方法实现。
func (h *Handler) AdminIssueRuns(ctx context.Context, c *app.RequestContext) {
	issueID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("issueID")), 10, 32)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid issue id")
		return
	}
	limit := 100
	if q := strings.TrimSpace(string(c.Query("limit"))); q != "" {
		if n, parseErr := strconv.Atoi(q); parseErr == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	rows, err := h.issueRunSvc.ListIssueRuns(ctx, uint(issueID64), limit)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]adminRunItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAdminRunItem(row))
	}
	httputil.WriteOK(c, map[string]any{
		"issue_id": uint(issueID64),
		"items":    out,
	})
}

// AdminRunLogs 是 *Handler 的方法实现。
func (h *Handler) AdminRunLogs(ctx context.Context, c *app.RequestContext) {
	runID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("runID")), 10, 32)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid run id")
		return
	}
	limit := 500
	if q := strings.TrimSpace(string(c.Query("limit"))); q != "" {
		if n, parseErr := strconv.Atoi(q); parseErr == nil && n > 0 && n <= 5000 {
			limit = n
		}
	}
	rows, err := h.issueRunSvc.ListRunLogs(ctx, uint(runID64), limit)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
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
	httputil.WriteOK(c, map[string]any{
		"run_id": uint(runID64),
		"items":  out,
	})
}

// AdminCancelRun 是 *Handler 的方法实现。
func (h *Handler) AdminCancelRun(ctx context.Context, c *app.RequestContext) {
	runID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("runID")), 10, 32)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid run id")
		return
	}
	var req adminRunActionRequest
	if len(c.Request.Body()) > 0 {
		if err := httputil.BindJSON(c, &req); err != nil {
			httputil.WriteError(c, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	row, err := h.issueRunSvc.CancelRun(ctx, uint(runID64), req.Reason)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	httputil.WriteOK(c, toAdminRunItem(*row))
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
