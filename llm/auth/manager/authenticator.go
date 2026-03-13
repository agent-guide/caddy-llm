package manager

import "context"

// Authenticator handles the CLI login flow for a specific provider.
// Each concrete implementation covers one CLI tool (e.g. Codex, Claude CLI).
type Authenticator interface {
	// Provider returns the unique provider name this authenticator handles (e.g. "openai", "anthropic").
	Provider() string
	// Login initiates the interactive CLI login flow and returns a new Credential on success.
	Login(ctx context.Context) (*Credential, error)
	// RefreshLead attempts to refresh the given credential before it expires.
	// Returns nil to indicate no refresh is needed; returns an updated Credential on success.
	RefreshLead(ctx context.Context, cred *Credential) (*Credential, error)
}
