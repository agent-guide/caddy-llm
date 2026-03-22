package admin

import (
	"context"
	"net/http"
	"strings"
)

// handleCLILogin triggers a provider-specific CLI login flow asynchronously.
// POST /admin/clilogin/{cliname}
//
// The handler returns 202 Accepted immediately. In the background it invokes
// the registered Authenticator's Login method. On success the returned
// credential is registered with the auth manager.
func (h *Handler) handleCLILogin(w http.ResponseWriter, r *http.Request) {
	if h.authManager == nil {
		writeError(w, http.StatusServiceUnavailable, "auth manager not configured")
		return
	}

	requestedName := strings.ToLower(strings.TrimSpace(r.PathValue("cliname")))
	if requestedName == "" {
		writeError(w, http.StatusBadRequest, "cliname is required")
		return
	}

	auth, ok := h.authManager.GetAuthenticator(requestedName)
	if !ok {
		writeError(w, http.StatusNotFound, requestedName+" authenticator not registered")
		return
	}

	// Run the login flow in the background so the HTTP call returns immediately.
	go func() {
		ctx := context.Background()
		cred, err := auth.Login(ctx)
		if err != nil {
			// Nothing to surface back to the caller at this point; errors are
			// visible in server logs when a logger is wired in.
			return
		}
		_ = h.authManager.Register(ctx, cred)
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":    "login_started",
		"cliname":   requestedName,
		"requested": requestedName,
		"message":   "CLI login initiated. Complete the provider authentication flow on the server to register the credential.",
	})
}
