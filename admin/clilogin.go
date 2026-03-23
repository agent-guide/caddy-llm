package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// loginStatus tracks the state of an async CLI login flow.
type loginStatus struct {
	Status       string     `json:"status"` // "running", "succeeded", "failed"
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	CredentialID string     `json:"credential_id,omitempty"`
}

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

	status := &loginStatus{
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	h.storeLoginStatus(requestedName, status)

	// Run the login flow in the background so the HTTP call returns immediately.
	go func() {
		ctx := context.Background()
		cred, err := auth.Login(ctx)
		finished := statusSnapshot(status)
		now := time.Now().UTC()
		finished.FinishedAt = &now
		if err != nil {
			finished.Status = "failed"
			finished.Error = err.Error()
			h.storeLoginStatus(requestedName, &finished)
			h.logger.Error("cli login failed", zap.String("cliname", requestedName), zap.Error(err))
			return
		}
		if regErr := h.authManager.Register(ctx, cred); regErr != nil {
			finished.Status = "failed"
			finished.Error = regErr.Error()
			h.storeLoginStatus(requestedName, &finished)
			h.logger.Error("cli login: register credential failed",
				zap.String("cliname", requestedName), zap.Error(regErr))
			return
		}
		finished.Status = "succeeded"
		finished.CredentialID = cred.ID
		h.storeLoginStatus(requestedName, &finished)
		h.logger.Info("cli login succeeded",
			zap.String("cliname", requestedName),
			zap.String("credential_id", cred.ID))
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "login_started",
		"cliname": requestedName,
		"message": "CLI login initiated. Complete the provider authentication flow on the server to register the credential.",
	})
}

// handleCLILoginStatus returns the current status of an async CLI login flow.
// GET /admin/clilogin/{cliname}/status
func (h *Handler) handleCLILoginStatus(w http.ResponseWriter, r *http.Request) {
	cliname := strings.ToLower(strings.TrimSpace(r.PathValue("cliname")))
	if cliname == "" {
		writeError(w, http.StatusBadRequest, "cliname is required")
		return
	}

	val, ok := h.loginSessions.Load(cliname)
	if !ok {
		writeError(w, http.StatusNotFound, "no login session found for "+cliname)
		return
	}
	status, ok := val.(loginStatus)
	if !ok {
		writeError(w, http.StatusInternalServerError, "invalid login session state")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) storeLoginStatus(cliname string, status *loginStatus) {
	if h == nil || status == nil {
		return
	}
	h.loginSessions.Store(cliname, statusSnapshot(status))
}

func statusSnapshot(status *loginStatus) loginStatus {
	if status == nil {
		return loginStatus{}
	}
	snapshot := *status
	if status.FinishedAt != nil {
		finished := *status.FinishedAt
		snapshot.FinishedAt = &finished
	}
	return snapshot
}
