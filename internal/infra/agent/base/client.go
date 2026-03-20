package base

import "context"

type Client interface {
	Name() string
	Run(ctx context.Context, req InvokeRequest) (*InvokeResult, error)
}
