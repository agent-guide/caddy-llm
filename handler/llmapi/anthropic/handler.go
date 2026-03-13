package anthropic

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Handler handles Anthropic-format API requests (/v1/messages).
type Handler struct {
	// provider provider.Provider  // TODO: wire in
}

// ServeHTTP handles /v1/messages and /v1/messages/count_tokens.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/count_tokens") {
		h.handleCountTokens(w, r)
		return
	}
	h.handleMessages(w, r)
}

func (h *Handler) handleMessages(w http.ResponseWriter, r *http.Request) {
	// TODO: decode request, call provider, encode response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}

func (h *Handler) handleCountTokens(w http.ResponseWriter, r *http.Request) {
	// TODO: implement token counting
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}
