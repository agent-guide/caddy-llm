package intf

import (
	"context"
)

// CredentialStorer abstracts persistence of Credential state across restarts.
type CredentialStorer interface {
	ListByProviderName(ctx context.Context, providerName string) ([]any, error)

	Create(ctx context.Context, id string, providerName string, obj any) (string, error)

	Update(ctx context.Context, id string, obj any) error

	Delete(ctx context.Context, id string) error

	// return (providerName, obj, error)
	Get(ctx context.Context, id string) (string, any, error)
}
