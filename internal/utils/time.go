package utils

import "time"

// NowUTC 返回当前 UTC 时间，供服务内部统一使用。
func NowUTC() time.Time {
	return time.Now().UTC()
}
