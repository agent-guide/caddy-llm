package provider

import (
	"context"

	"github.com/agent-guide/caddy-llm/llm/auth/credential"
)

type credentialKey struct{}

// WithCredential attaches a credential to the context for per-request auth override.
// The openaicompat Base reads this in setHeaders to replace the static APIKey.
func WithCredential(ctx context.Context, cred *credential.Credential) context.Context {
	return context.WithValue(ctx, credentialKey{}, cred)
}

// CredentialFromContext retrieves the per-request credential from the context.
func CredentialFromContext(ctx context.Context) (*credential.Credential, bool) {
	cred, ok := ctx.Value(credentialKey{}).(*credential.Credential)
	return cred, ok && cred != nil
}
