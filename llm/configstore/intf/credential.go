package intf

import (
	"context"
)

// CredentialStorer abstracts persistence of Credential state across restarts.
type CredentialStorer interface {
	ListByProviderName(ctx context.Context, providerName string) ([]any, error)

	Save(ctx context.Context, id string, providerName string, obj any) (string, error)

	Delete(ctx context.Context, id string) error

	// return (tag, config, error)
	Get(ctx context.Context, id string) (string, any, error)
}
