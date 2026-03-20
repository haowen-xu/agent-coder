package base

import "context"

// Client 定义接口行为。
type Client interface { // Name 定义接口方法。
	Name() string // Run 定义接口方法。
	// Run 定义接口方法。
	Run(ctx context.Context, req InvokeRequest) (*InvokeResult, error)
}
