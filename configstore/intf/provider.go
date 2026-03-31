package intf

import (
	"context"
)

// ProviderConfigStorer abstracts persistence of the config of providers.
type ProviderConfigStorer interface {
	ListByName(ctx context.Context, name string) ([]any, error)

	// create a new provider config
	Save(ctx context.Context, id string, name string, obj any) (string, error)

	// update the config, and `name` is not allowed to be modified
	Update(ctx context.Context, id string, obj any) error

	Delete(ctx context.Context, id string) error

	// return (name, config, error)
	Get(ctx context.Context, id string) (string, any, error)
}
