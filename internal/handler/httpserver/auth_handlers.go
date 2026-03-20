package httpserver

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

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

func (s *Server) me(_ context.Context, c *app.RequestContext) {
	user := currentUser(c)
	if user == nil {
		writeError(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeOK(c, map[string]any{"user": user})
}
