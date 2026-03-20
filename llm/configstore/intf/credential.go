package intf

import (
	"context"
)

// CredentialStorer abstracts persistence of Credential state across restarts.
type CredentialStorer interface {
	ListByTag(ctx context.Context, tag string) ([]any, error)

	Save(ctx context.Context, id string, tag string, obj any) (string, error)

	Delete(ctx context.Context, id string) error

	// return (tag, config, error)
	Get(ctx context.Context, id string) (string, any, error)
}
