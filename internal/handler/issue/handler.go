package issue

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/haowen-xu/agent-coder/internal/handler/httputil"
	issuesvc "github.com/haowen-xu/agent-coder/internal/service/issue"
)

// Handler 表示数据结构定义。
type Handler struct {
	issueSvc *issuesvc.Service // issueSvc 字段说明。
}

// New 执行相关逻辑。
func New(issueSvc *issuesvc.Service) *Handler {
	return &Handler{issueSvc: issueSvc}
}

// AdminRetryIssue 是 *Handler 的方法实现。
func (h *Handler) AdminRetryIssue(ctx context.Context, c *app.RequestContext) {
	issueID64, err := strconv.ParseUint(strings.TrimSpace(c.Param("issueID")), 10, 32)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid issue id")
		return
	}
	row, err := h.issueSvc.RetryIssue(ctx, uint(issueID64))
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	httputil.WriteOK(c, map[string]any{
		"issue_id":         row.ID,
		"lifecycle_status": row.LifecycleStatus,
		"close_reason":     row.CloseReason,
		"retried":          true,
	})
}
