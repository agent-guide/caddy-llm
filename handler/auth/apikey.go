package auth

import "net/http"

// APIKeyAuth authenticates requests using API keys stored in the config.
type APIKeyAuth struct {
	// store config.Store  // TODO: wire in
}

// Authenticate implements Authenticator.
func (a *APIKeyAuth) Authenticate(r *http.Request) (*Identity, error) {
	key := extractAPIKey(r)
	if key == "" {
		return nil, ErrUnauthorized
	}
	// TODO: look up key in config store and return Identity
	return nil, ErrUnauthorized
}
