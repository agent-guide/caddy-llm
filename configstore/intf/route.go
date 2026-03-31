package intf

import "context"

// RouteStorer persists gateway route definitions.
type RouteStorer interface {
	List(ctx context.Context) ([]any, error)
	Save(ctx context.Context, id string, obj any) error
	Update(ctx context.Context, id string, obj any) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (any, error)
}
