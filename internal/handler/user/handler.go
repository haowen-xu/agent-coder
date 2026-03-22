package user

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/haowen-xu/agent-coder/internal/handler/httputil"
	usersvc "github.com/haowen-xu/agent-coder/internal/service/user"
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

// Handler 表示数据结构定义。
type Handler struct {
	userSvc *usersvc.Service // userSvc 字段说明。
}

// New 执行相关逻辑。
func New(userSvc *usersvc.Service) *Handler {
	return &Handler{userSvc: userSvc}
}

// AdminListUsers 是 *Handler 的方法实现。
func (h *Handler) AdminListUsers(ctx context.Context, c *app.RequestContext) {
	rows, err := h.userSvc.ListUsers(ctx)
	if err != nil {
		httputil.WriteError(c, http.StatusInternalServerError, err.Error())
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
	httputil.WriteOK(c, map[string]any{"items": out})
}

// AdminCreateUser 是 *Handler 的方法实现。
func (h *Handler) AdminCreateUser(ctx context.Context, c *app.RequestContext) {
	var req adminCreateUserRequest
	if err := httputil.BindJSON(c, &req); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid json body")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row, err := h.userSvc.CreateUser(ctx, req.Username, req.Password, req.IsAdmin, enabled)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}

	httputil.WriteOK(c, map[string]any{
		"id":       row.ID,
		"username": row.Username,
		"is_admin": row.IsAdmin,
		"enabled":  row.Enabled,
	})
}

// AdminUpdateUser 是 *Handler 的方法实现。
func (h *Handler) AdminUpdateUser(ctx context.Context, c *app.RequestContext) {
	id64, err := strconv.ParseUint(strings.TrimSpace(c.Param("id")), 10, 32)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req adminUpdateUserRequest
	if err := httputil.BindJSON(c, &req); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid json body")
		return
	}

	row, err := h.userSvc.UpdateUser(ctx, uint(id64), req.Password, req.IsAdmin, req.Enabled)
	if err != nil {
		httputil.WriteError(c, http.StatusBadRequest, err.Error())
		return
	}

	httputil.WriteOK(c, map[string]any{
		"id":       row.ID,
		"username": row.Username,
		"is_admin": row.IsAdmin,
		"enabled":  row.Enabled,
	})
}
