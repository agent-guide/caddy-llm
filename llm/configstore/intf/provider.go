package intf

import (
	"context"
)

// ProviderConfigStorer abstracts persistence of the config of providers.
type ProviderConfigStorer interface {
	ListByTag(ctx context.Context, tag string) ([]any, error)

	// create a new provider config
	Save(ctx context.Context, id string, tag string, obj any) (string, error)

	// update the config, and `tag` is not allowed to be modified
	Update(ctx context.Context, id string, obj any) error

	Delete(ctx context.Context, id string) error

	// return (tag, config, error)
	Get(ctx context.Context, id string) (string, any, error)
}
