package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// writeError 执行相关逻辑。
func writeError(c *app.RequestContext, status int, message string) {
	c.JSON(status, map[string]any{
		"error": strings.TrimSpace(message),
	})
}

// bindJSON 执行相关逻辑。
func bindJSON(c *app.RequestContext, dst any) error {
	return json.Unmarshal(c.Request.Body(), dst)
}

// writeOK 执行相关逻辑。
func writeOK(c *app.RequestContext, data any) {
	c.JSON(http.StatusOK, data)
}
