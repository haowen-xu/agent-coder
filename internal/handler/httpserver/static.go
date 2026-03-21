package httpserver

import (
	"context"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

//go:embed static/* static/**/*
var webuiFS embed.FS

// registerStaticRoutes 是方法实现。
func (s *Server) registerStaticRoutes() {
	s.hz.NoRoute(s.handleNoRoute)
}

// servePrecompressedStaticGzip 是方法实现。
func (s *Server) servePrecompressedStaticGzip() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if !c.IsGet() && !c.IsHead() {
			c.Next(ctx)
			return
		}

		rawPath := string(c.Path())
		if strings.HasPrefix(rawPath, "/api/") || rawPath == "/healthz" {
			c.Next(ctx)
			return
		}

		clean := normalizeStaticPath(rawPath)
		if clean == "/index.html" || path.Ext(clean) != "" {
			if servePrecompressedStaticAsset(c, clean) {
				c.Abort()
				return
			}
		}
		c.Next(ctx)
	}
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
	clean := normalizeStaticPath(rawPath)
	if (clean == "/index.html" || path.Ext(clean) != "") && servePrecompressedStaticAsset(c, clean) {
		return
	}

	contentPath := "static" + clean
	data, err := webuiFS.ReadFile(contentPath)
	if err != nil {
		if servePrecompressedStaticAsset(c, "/index.html") {
			return
		}

		data, err = webuiFS.ReadFile("static/index.html")
		if err != nil {
			writeError(c, http.StatusNotFound, "webui asset not found")
			return
		}
		c.SetStatusCode(http.StatusOK)
		c.Header("Content-Type", "text/html; charset=utf-8")
		if c.IsHead() {
			return
		}
		c.Write(data)
		return
	}

	c.SetStatusCode(http.StatusOK)
	if ct := mime.TypeByExtension(path.Ext(contentPath)); ct != "" {
		c.Header("Content-Type", ct)
	}
	if c.IsHead() {
		return
	}
	c.Write(data)
}

// normalizeStaticPath 执行相关逻辑。
func normalizeStaticPath(rawPath string) string {
	target := strings.TrimSpace(rawPath)
	if target == "" || target == "/" {
		target = "/index.html"
	}

	clean := path.Clean("/" + strings.TrimPrefix(target, "/"))
	if clean == "/" {
		clean = "/index.html"
	}
	return clean
}

// acceptsGzipEncoding 执行相关逻辑。
func acceptsGzipEncoding(c *app.RequestContext) bool {
	encoding := strings.TrimSpace(string(c.GetHeader("Accept-Encoding")))
	return acceptsGzipEncodingHeader(encoding)
}

// acceptsGzipEncodingHeader 执行相关逻辑。
func acceptsGzipEncodingHeader(headerValue string) bool {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(headerValue)), ",")
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}

		name := item
		params := ""
		if idx := strings.Index(item, ";"); idx >= 0 {
			name = strings.TrimSpace(item[:idx])
			params = strings.TrimSpace(item[idx+1:])
		}

		if name != "gzip" && name != "x-gzip" && name != "*" {
			continue
		}

		if params == "" {
			return true
		}

		qValue := 1.0
		for _, param := range strings.Split(params, ";") {
			param = strings.TrimSpace(param)
			if !strings.HasPrefix(param, "q=") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(param, "q=")), 64)
			if err == nil {
				qValue = parsed
			}
			break
		}
		if qValue > 0 {
			return true
		}
	}
	return false
}

// readPrecompressedStaticAsset 执行相关逻辑。
func readPrecompressedStaticAsset(fsys fs.ReadFileFS, cleanPath string) ([]byte, string, bool) {
	gzipPath := "static" + cleanPath + ".gz"
	data, err := fsys.ReadFile(gzipPath)
	if err != nil {
		return nil, "", false
	}

	contentType := mime.TypeByExtension(path.Ext(cleanPath))
	return data, contentType, true
}

// servePrecompressedStaticAsset 执行相关逻辑。
func servePrecompressedStaticAsset(c *app.RequestContext, cleanPath string) bool {
	if !acceptsGzipEncoding(c) {
		return false
	}

	data, contentType, ok := readPrecompressedStaticAsset(webuiFS, cleanPath)
	if !ok {
		return false
	}

	if contentType != "" {
		c.Header("Content-Type", contentType)
	}
	c.Header("Content-Encoding", "gzip")
	c.Header("Vary", "Accept-Encoding")
	c.SetStatusCode(http.StatusOK)
	if c.IsHead() {
		return true
	}
	c.Write(data)
	return true
}
