package admin

import (
	"context"
	"net/http"
)

// handleCodexLogin triggers the Codex OAuth browser login flow asynchronously.
// POST /admin/clilogin/codex
//
// The handler returns 202 Accepted immediately. In the background it invokes
// the CodexAuthenticator's Login method (which opens a browser for PKCE OAuth
// and listens on localhost:1455 for the callback). On success the returned
// credential is registered with the auth manager.
func (h *Handler) handleCodexLogin(w http.ResponseWriter, r *http.Request) {
	if h.authManager == nil {
		writeError(w, http.StatusServiceUnavailable, "auth manager not configured")
		return
	}

	auth, ok := h.authManager.GetAuthenticator("codex")
	if !ok {
		writeError(w, http.StatusNotFound, "codex authenticator not registered")
		return
	}

	// Run the login flow in the background so the HTTP call returns immediately.
	// The Codex browser flow can take up to 5 minutes waiting for the OAuth callback.
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
		"status":  "login_started",
		"message": "Codex OAuth login initiated. A browser window will open on the server. Complete the login to register the credential.",
	})
}
