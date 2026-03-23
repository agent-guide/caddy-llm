package admin

import (
	"encoding/json"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	"github.com/agent-guide/caddy-llm/llm/configstore/intf"
)

// Handler handles Admin API requests under /admin/.
type Handler struct {
	authManager   *manager.Manager
	configStore   intf.ConfigStorer
	mux           *http.ServeMux
	logger        *zap.Logger
	loginSessions sync.Map // cliname -> *loginStatus
}

// NewHandler constructs an admin Handler with the given auth manager.
// logger may be nil (a no-op logger is used in that case).
func NewHandler(authMgr *manager.Manager, configStore intf.ConfigStorer, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	h := &Handler{authManager: authMgr, configStore: configStore, logger: logger}
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

func decodeJSON(r *http.Request, dest any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dest)
}
