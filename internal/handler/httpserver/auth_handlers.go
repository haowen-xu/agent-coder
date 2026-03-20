package httpserver

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

// loginRequest 表示数据结构定义。
type loginRequest struct {
	Username string `json:"username"` // Username 字段说明。
	Password string `json:"password"` // Password 字段说明。
}

// login 是 *Server 的方法实现。
func (s *Server) login(ctx context.Context, c *app.RequestContext) {
	var req loginRequest
	if err := bindJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid json body")
		return
	}
	token, expiredAt, user, err := s.svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err.Error())
		return
	}

	writeOK(c, map[string]any{
		"token":      token,
		"expired_at": expiredAt,
		"user": authUserView{
			ID:       user.ID,
			Username: user.Username,
			IsAdmin:  user.IsAdmin,
			Enabled:  user.Enabled,
		},
	})
}

// me 是 *Server 的方法实现。
func (s *Server) me(_ context.Context, c *app.RequestContext) {
	user := currentUser(c)
	if user == nil {
		writeError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeOK(c, map[string]any{"user": user})
}
