package httputil

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// WriteError 执行相关逻辑。
func WriteError(c *app.RequestContext, status int, message string) {
	c.JSON(status, map[string]any{
		"error": strings.TrimSpace(message),
	})
}

// BindJSON 执行相关逻辑。
func BindJSON(c *app.RequestContext, dst any) error {
	return json.Unmarshal(c.Request.Body(), dst)
}

// WriteOK 执行相关逻辑。
func WriteOK(c *app.RequestContext, data any) {
	c.JSON(http.StatusOK, data)
}
