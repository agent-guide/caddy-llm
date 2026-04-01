package admin

import (
	"encoding/json"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"github.com/agent-guide/caddy-agent-gateway/llm/authmanager/manager"
	"github.com/agent-guide/caddy-agent-gateway/configstore/intf"
)

// Handler handles Admin API requests under /admin/.
type Handler struct {
	authManager      *manager.Manager
	configStore      intf.ConfigStorer
	mux              *http.ServeMux
	logger           *zap.Logger
	loginSessions    sync.Map // cliname -> *loginStatus
	sessions         *sessionStore
	adminUsername    string
	adminPasswordHash string
}

// NewHandler constructs an admin Handler with the given auth manager.
// logger may be nil (a no-op logger is used in that case).
func NewHandler(authMgr *manager.Manager, configStore intf.ConfigStorer, logger *zap.Logger, adminUser, adminPasswordHash string) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	h := &Handler{
		authManager:       authMgr,
		configStore:       configStore,
		logger:            logger,
		sessions:          newSessionStore(),
		adminUsername:     adminUser,
		adminPasswordHash: adminPasswordHash,
	}
	h.mux = http.NewServeMux()
	for _, route := range h.Routes() {
		handler := route.Handler
		if route.RequireAuth {
			handler = h.requireAuth(handler)
		}
		h.mux.HandleFunc(route.Method+" "+route.Path, handler)
	}
	return h
}

// ServeHTTP dispatches admin API requests, including CORS preflight handling.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")
	}
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
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
