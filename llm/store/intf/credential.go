package intf

import (
	"context"

	"github.com/agent-guide/caddy-llm/llm/auth/credential"
)

// CredentialStorer abstracts persistence of Credential state across restarts.
type CredentialStorer interface {
	// List returns all credential records stored in the backend.
	List(ctx context.Context) ([]*credential.Credential, error)
	// Save persists the provided credential record, replacing any existing one with the same ID.
	// Returns the storage path or key used.
	Save(ctx context.Context, cred *credential.Credential) (string, error)
	// Delete removes the credential record identified by id.
	Delete(ctx context.Context, id string) error
}
