package config

import "context"

// Store is the key-value config storage interface.
type Store interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Tx(ctx context.Context, fn func(tx Store) error) error
}
