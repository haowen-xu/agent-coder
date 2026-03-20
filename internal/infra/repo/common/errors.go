package common

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNeedHumanMerge 表示代码托管平台拒绝自动合并，必须转人工合并。
type ErrNeedHumanMerge struct {
	Provider   string // Provider 是仓库平台标识（如 gitlab）。
	StatusCode int    // StatusCode 是平台接口返回的 HTTP 状态码。
	Reason     string // Reason 是平台返回的原始原因摘要。
}

// Error 返回规范化的错误文本，便于日志和 issue note 直接复用。
func (e *ErrNeedHumanMerge) Error() string {
	if e == nil {
		return "need human merge"
	}
	reason := strings.TrimSpace(e.Reason)
	if reason == "" {
		reason = "need human merge"
	}
	if strings.TrimSpace(e.Provider) == "" {
		if e.StatusCode > 0 {
			return fmt.Sprintf("%s (status=%d)", reason, e.StatusCode)
		}
		return reason
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s provider=%s status=%d", reason, e.Provider, e.StatusCode)
	}
	return fmt.Sprintf("%s provider=%s", reason, e.Provider)
}

// IsNeedHumanMerge 判断错误链中是否包含 ErrNeedHumanMerge。
func IsNeedHumanMerge(err error) bool {
	var target *ErrNeedHumanMerge
	return errors.As(err, &target)
}
