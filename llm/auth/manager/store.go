package manager

import "context"

// Store abstracts persistence of Credential state across restarts.
type Store interface {
	// List returns all credential records stored in the backend.
	List(ctx context.Context) ([]*Credential, error)
	// Save persists the provided credential record, replacing any existing one with the same ID.
	// Returns the storage path or key used.
	Save(ctx context.Context, cred *Credential) (string, error)
	// Delete removes the credential record identified by id.
	Delete(ctx context.Context, id string) error
}
