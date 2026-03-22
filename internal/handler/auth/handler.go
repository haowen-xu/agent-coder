package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/haowen-xu/agent-coder/internal/handler/httputil"
	usersvc "github.com/haowen-xu/agent-coder/internal/service/user"
)

const (
	ctxUserKey = "auth_user"
)

// UserView 表示数据结构定义。
type UserView struct {
	ID       uint   `json:"id"`       // ID 字段说明。
	Username string `json:"username"` // Username 字段说明。
	IsAdmin  bool   `json:"is_admin"` // IsAdmin 字段说明。
	Enabled  bool   `json:"enabled"`  // Enabled 字段说明。
}

// loginRequest 表示数据结构定义。
type loginRequest struct {
	Username string `json:"username"` // Username 字段说明。
	Password string `json:"password"` // Password 字段说明。
}

// Handler 表示数据结构定义。
type Handler struct {
	userSvc *usersvc.Service // userSvc 字段说明。
}

// New 执行相关逻辑。
func New(userSvc *usersvc.Service) *Handler {
	return &Handler{userSvc: userSvc}
}

// Login 是 *Handler 的方法实现。
func (h *Handler) Login(ctx context.Context, c *app.RequestContext) {
	var req loginRequest
	if err := httputil.BindJSON(c, &req); err != nil {
		httputil.WriteError(c, http.StatusBadRequest, "invalid json body")
		return
	}
	token, expiredAt, user, err := h.userSvc.Login(ctx, req.Username, req.Password)
	if err != nil {
		httputil.WriteError(c, http.StatusUnauthorized, err.Error())
		return
	}

	httputil.WriteOK(c, map[string]any{
		"token":      token,
		"expired_at": expiredAt,
		"user": UserView{
			ID:       user.ID,
			Username: user.Username,
			IsAdmin:  user.IsAdmin,
			Enabled:  user.Enabled,
		},
	})
}

// Me 是 *Handler 的方法实现。
func (h *Handler) Me(_ context.Context, c *app.RequestContext) {
	user := CurrentUser(c)
	if user == nil {
		httputil.WriteError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	httputil.WriteOK(c, map[string]any{"user": user})
}

// RequireLogin 是 *Handler 的方法实现。
func (h *Handler) RequireLogin() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authz := strings.TrimSpace(string(c.GetHeader("Authorization")))
		if authz == "" || !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			httputil.WriteError(c, http.StatusUnauthorized, "missing bearer token")
			c.Abort()
			return
		}
		token := strings.TrimSpace(authz[7:])
		if token == "" {
			httputil.WriteError(c, http.StatusUnauthorized, "invalid bearer token")
			c.Abort()
			return
		}

		user, err := h.userSvc.AuthByToken(ctx, token)
		if err != nil {
			httputil.WriteError(c, http.StatusInternalServerError, err.Error())
			c.Abort()
			return
		}
		if user == nil || !user.Enabled {
			httputil.WriteError(c, http.StatusUnauthorized, "invalid or expired session")
			c.Abort()
			return
		}

		c.Set(ctxUserKey, &UserView{
			ID:       user.ID,
			Username: user.Username,
			IsAdmin:  user.IsAdmin,
			Enabled:  user.Enabled,
		})
		c.Next(ctx)
	}
}

// RequireAdmin 是 *Handler 的方法实现。
func (h *Handler) RequireAdmin() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		user := CurrentUser(c)
		if user == nil || !user.IsAdmin {
			httputil.WriteError(c, http.StatusForbidden, "admin required")
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}

// CurrentUser 执行相关逻辑。
func CurrentUser(c *app.RequestContext) *UserView {
	v, ok := c.Get(ctxUserKey)
	if !ok {
		return nil
	}
	row, ok := v.(*UserView)
	if ok {
		return row
	}
	return nil
}
