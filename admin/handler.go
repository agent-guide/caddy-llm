package admin

import (
	"encoding/json"
	"net/http"

	"github.com/agent-guide/caddy-llm/llm/auth/manager"
)

// Handler handles Admin API requests under /admin/.
type Handler struct {
	authManager *manager.Manager
	mux         *http.ServeMux
	// mcp    mcp.Manager     // TODO: wire in
}

// NewHandler constructs an admin Handler with the given auth manager.
func NewHandler(authMgr *manager.Manager) *Handler {
	h := &Handler{authManager: authMgr}
	h.mux = http.NewServeMux()
	for _, route := range h.Routes() {
		h.mux.HandleFunc(route.Method+" "+route.Path, route.Handler)
	}
	return h
}

// ServeHTTP dispatches admin API requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
