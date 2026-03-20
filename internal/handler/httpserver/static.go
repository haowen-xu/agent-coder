package httpserver

import (
	"context"
	"embed"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

//go:embed static/**/*
var webuiFS embed.FS

// registerStaticRoutes 是方法实现。
func (s *Server) registerStaticRoutes() {
	s.hz.NoRoute(s.handleNoRoute)
}

// handleNoRoute 是方法实现。
func (s *Server) handleNoRoute(_ context.Context, c *app.RequestContext) {
	p := string(c.Path())
	if strings.HasPrefix(p, "/api/") {
		writeError(c, http.StatusNotFound, "route not found")
		return
	}
	s.serveSPA(c, p)
}

// serveSPA 是方法实现。
func (s *Server) serveSPA(c *app.RequestContext, rawPath string) {
	target := strings.TrimSpace(rawPath)
	if target == "" || target == "/" {
		target = "/index.html"
	}

	clean := path.Clean("/" + strings.TrimPrefix(target, "/"))
	if clean == "/" {
		clean = "/index.html"
	}
	contentPath := "static" + clean
	data, err := webuiFS.ReadFile(contentPath)
	if err != nil {
		data, err = webuiFS.ReadFile("static/index.html")
		if err != nil {
			writeError(c, http.StatusNotFound, "webui asset not found")
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Write(data)
		return
	}

	if ct := mime.TypeByExtension(path.Ext(contentPath)); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Write(data)
}
