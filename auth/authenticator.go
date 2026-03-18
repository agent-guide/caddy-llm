package auth

import (
	"errors"
	"net/http"
	"strings"
)

// ErrUnauthorized is returned when authentication fails.
var ErrUnauthorized = errors.New("unauthorized")

// IdentityType classifies the authenticated principal.
type IdentityType string

const (
	IdentityTypeAPIKey IdentityType = "api_key"
	IdentityTypeUser   IdentityType = "user"
	IdentityTypeAgent  IdentityType = "agent"
)

// Identity represents an authenticated identity.
type Identity struct {
	ID       string
	Type     IdentityType
	Scopes   []string
	Metadata map[string]any
}

// Authenticator authenticates an HTTP request and returns an Identity.
type Authenticator interface {
	Authenticate(r *http.Request) (*Identity, error)
}

// extractAPIKey extracts the API key from x-api-key header or Bearer token.
func extractAPIKey(r *http.Request) string {
	if key := r.Header.Get("x-api-key"); key != "" {
		return key
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
