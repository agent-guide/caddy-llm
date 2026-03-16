package intf

import (
	"context"

	"github.com/agent-guide/caddy-llm/llm/provider"
)

// PrivoderConfigStorer abstracts persistence of the config of providers.
type PrivoderConfigStorer interface {
	List(ctx context.Context) ([]*provider.ProviderConfig, error)

	Save(ctx context.Context, cred *provider.ProviderConfig) (string, error)

	Delete(ctx context.Context, id string) error
}
