package intf

import "context"

type VXApiKeyStorer interface {
	List(ctx context.Context) ([]any, error)

	Save(ctx context.Context, key string, obj any) error

	Delete(ctx context.Context, key string) error

	Get(ctx context.Context, key string) (any, error)
}
