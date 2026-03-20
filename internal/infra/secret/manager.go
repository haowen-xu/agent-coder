package secret

import "context"

type Manager interface {
	Get(ctx context.Context, ref string) (string, error)
}
