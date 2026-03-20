package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	ctxUserKey = "auth_user"
)

func (s *Server) requireLogin() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authz := strings.TrimSpace(string(c.GetHeader("Authorization")))
		if authz == "" || !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			writeError(c, http.StatusUnauthorized, "missing bearer token")
			c.Abort()
			return
		}
		token := strings.TrimSpace(authz[7:])
		if token == "" {
			writeError(c, http.StatusUnauthorized, "invalid bearer token")
			c.Abort()
			return
		}

		user, err := s.svc.AuthByToken(ctx, token)
		if err != nil {
			writeError(c, http.StatusInternalServerError, err.Error())
			c.Abort()
			return
		}
		if user == nil || !user.Enabled {
			writeError(c, http.StatusUnauthorized, "invalid or expired session")
			c.Abort()
			return
		}

		c.Set(ctxUserKey, &authUserView{
			ID:       user.ID,
			Username: user.Username,
			IsAdmin:  user.IsAdmin,
			Enabled:  user.Enabled,
		})
		c.Next(ctx)
	}
}

func (s *Server) requireAdmin() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		user := currentUser(c)
		if user == nil || !user.IsAdmin {
			writeError(c, http.StatusForbidden, "admin required")
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}

func currentUser(c *app.RequestContext) *authUserView {
	v, ok := c.Get(ctxUserKey)
	if !ok {
		return nil
	}
	row, ok := v.(*authUserView)
	if ok {
		return row
	}
	return nil
}

type authUserView struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	Enabled  bool   `json:"enabled"`
}
