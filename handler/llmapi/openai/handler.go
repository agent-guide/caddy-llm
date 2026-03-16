package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/agent-guide/caddy-llm/llm/auth/manager"
	"github.com/agent-guide/caddy-llm/llm/provider"
)

// Handler handles OpenAI-format API requests (/v1/chat/completions, etc.).
type Handler struct {
	authManager *manager.Manager
	prov        provider.Provider
}

// NewHandler creates a Handler with the given auth manager and provider.
func NewHandler(authMgr *manager.Manager, prov provider.Provider) *Handler {
	return &Handler{authManager: authMgr, prov: prov}
}

// ServeHTTP handles OpenAI API requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %s", err))
		return
	}

	conv := &Converter{}
	genReq := conv.ToInternal(&req)

	// Pick a credential and inject into context for per-request auth override.
	ctx := r.Context()
	if cred, err := h.authManager.Pick(ctx, "codex", genReq.Model, nil); err == nil && cred != nil {
		ctx = provider.WithCredential(ctx, cred)
	}

	if req.Stream {
		h.serveStream(w, ctx, genReq)
		return
	}

	resp, err := h.prov.Generate(ctx, genReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conv.FromInternal(resp))
}

func (h *Handler) serveStream(w http.ResponseWriter, ctx context.Context, genReq *provider.GenerateRequest) {
	result, err := h.prov.Stream(ctx, genReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			break
		}
		fmt.Fprintf(w, "data: %s\n\n", chunk.Payload)
		if canFlush {
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
