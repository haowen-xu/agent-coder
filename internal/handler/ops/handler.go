package ops

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/haowen-xu/agent-coder/internal/handler/httputil"
	opssvc "github.com/haowen-xu/agent-coder/internal/service/ops"
)

// Handler 表示数据结构定义。
type Handler struct {
	opsSvc *opssvc.Service // opsSvc 字段说明。
}

// New 执行相关逻辑。
func New(opsSvc *opssvc.Service) *Handler {
	return &Handler{opsSvc: opsSvc}
}

// AdminMetrics 是 *Handler 的方法实现。
func (h *Handler) AdminMetrics(ctx context.Context, c *app.RequestContext) {
	row, err := h.opsSvc.GetMetrics(ctx)
	if err != nil {
		httputil.WriteError(c, http.StatusInternalServerError, err.Error())
		return
	}
	httputil.WriteOK(c, row)
}
