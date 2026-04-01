package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/agent-guide/caddy-agent-gateway/internal/utils"
	"go.uber.org/zap"
)

// cliAuthStatus tracks the state of an async CLI auth flow.
type cliAuthStatus struct {
	Status       string     `json:"status"` // "running", "succeeded", "failed"
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	CredentialID string     `json:"credential_id,omitempty"`
}

// handleCLIAuth triggers a provider-specific CLI auth flow asynchronously.
// POST /admin/cliauth/{cliname}
//
// The handler returns 202 Accepted immediately. In the background it invokes
// the registered Authenticator's Login method. On success the returned
// credential is registered with the auth manager.
func (h *Handler) handleCLIAuth(w http.ResponseWriter, r *http.Request) {
	if h.cliauthManager == nil {
		_ = utils.WriteError(w, http.StatusServiceUnavailable, "auth manager not configured")
		return
	}

	requestedName := strings.ToLower(strings.TrimSpace(r.PathValue("cliname")))
	if requestedName == "" {
		_ = utils.WriteError(w, http.StatusBadRequest, "cliname is required")
		return
	}

	auth, ok := h.cliauthManager.GetAuthenticator(requestedName)
	if !ok {
		_ = utils.WriteError(w, http.StatusNotFound, requestedName+" authenticator not registered")
		return
	}

	status := &cliAuthStatus{
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	h.storeCLIAuthStatus(requestedName, status)

	// Run the login flow in the background so the HTTP call returns immediately.
	go func() {
		ctx := context.Background()
		cred, err := auth.Login(ctx)
		finished := cliAuthStatusSnapshot(status)
		now := time.Now().UTC()
		finished.FinishedAt = &now
		if err != nil {
			finished.Status = "failed"
			finished.Error = err.Error()
			h.storeCLIAuthStatus(requestedName, &finished)
			h.logger.Error("cli login failed", zap.String("cliname", requestedName), zap.Error(err))
			return
		}
		if regErr := h.cliauthManager.Register(ctx, cred); regErr != nil {
			finished.Status = "failed"
			finished.Error = regErr.Error()
			h.storeCLIAuthStatus(requestedName, &finished)
			h.logger.Error("cli login: register credential failed",
				zap.String("cliname", requestedName), zap.Error(regErr))
			return
		}
		finished.Status = "succeeded"
		finished.CredentialID = cred.ID
		h.storeCLIAuthStatus(requestedName, &finished)
		h.logger.Info("cli login succeeded",
			zap.String("cliname", requestedName),
			zap.String("credential_id", cred.ID))
	}()

	_ = utils.WriteJSON(w, http.StatusAccepted, map[string]string{
		"status":  "login_started",
		"cliname": requestedName,
		"message": "CLI login initiated. Complete the provider authentication flow on the server to register the credential.",
	})
}

// handleCLIAuthStatus returns the current status of an async CLI auth flow.
// GET /admin/cliauth/{cliname}/status
func (h *Handler) handleCLIAuthStatus(w http.ResponseWriter, r *http.Request) {
	cliname := strings.ToLower(strings.TrimSpace(r.PathValue("cliname")))
	if cliname == "" {
		_ = utils.WriteError(w, http.StatusBadRequest, "cliname is required")
		return
	}

	val, ok := h.cliAuthSessions.Load(cliname)
	if !ok {
		_ = utils.WriteError(w, http.StatusNotFound, "no login session found for "+cliname)
		return
	}
	status, ok := val.(cliAuthStatus)
	if !ok {
		_ = utils.WriteError(w, http.StatusInternalServerError, "invalid login session state")
		return
	}
	_ = utils.WriteJSON(w, http.StatusOK, status)
}

func (h *Handler) storeCLIAuthStatus(cliname string, status *cliAuthStatus) {
	if h == nil || status == nil {
		return
	}
	h.cliAuthSessions.Store(cliname, cliAuthStatusSnapshot(status))
}

func cliAuthStatusSnapshot(status *cliAuthStatus) cliAuthStatus {
	if status == nil {
		return cliAuthStatus{}
	}
	snapshot := *status
	if status.FinishedAt != nil {
		finished := *status.FinishedAt
		snapshot.FinishedAt = &finished
	}
	return snapshot
}
