package secret

import "context"

// Manager 定义接口行为。
type Manager interface { // Get 定义接口方法。
	// Get 定义接口方法。
	Get(ctx context.Context, ref string) (string, error)
}
