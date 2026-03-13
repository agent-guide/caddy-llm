package admin

import (
	"encoding/json"
	"net/http"
)

// Handler handles Admin API requests under /admin/.
type Handler struct {
	// config config.Manager  // TODO: wire in
	// mcp    mcp.Manager     // TODO: wire in
}

// ServeHTTP dispatches admin API requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: implement route dispatch
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
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
