package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/agent-guide/caddy-llm/llm/auth/manager"
	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/cloudwego/eino/schema"
)

// Handler handles Anthropic-format API requests (/v1/messages).
type Handler struct {
	authManager *manager.Manager
	prov        provider.Provider
}

// NewHandler creates a Handler wired with the given auth manager and provider.
func NewHandler(authMgr *manager.Manager, prov provider.Provider) *Handler {
	return &Handler{authManager: authMgr, prov: prov}
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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var req MessagesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %s", err))
		return
	}

	conv := &Converter{}
	genReq := conv.ToInternal(&req)

	// Inject per-request credential from CLI login (same pattern as OpenAI handler).
	ctx := r.Context()
	if cred, err := h.authManager.Pick(ctx, "codex", genReq.Model, nil); err == nil && cred != nil {
		ctx = provider.WithCredential(ctx, cred)
	}

	if req.Stream {
		h.serveStream(w, ctx, genReq, req.Model)
		return
	}

	resp, err := h.prov.Generate(ctx, genReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conv.FromInternal(resp, req.Model))
}

func (h *Handler) serveStream(w http.ResponseWriter, ctx context.Context, genReq *provider.GenerateRequest, model string) {
	stream, err := h.prov.Stream(ctx, genReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)
	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	writeSSEEvent(w, "message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id": msgID, "type": "message", "role": "assistant",
			"model": model, "content": []any{},
			"stop_reason": nil,
			"usage":       map[string]int{"input_tokens": 0, "output_tokens": 0},
		},
	})
	writeSSEEvent(w, "content_block_start", map[string]any{
		"type": "content_block_start", "index": 0,
		"content_block": map[string]string{"type": "text", "text": ""},
	})
	if canFlush {
		flusher.Flush()
	}

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if text := extractText(chunk); text != "" {
			writeSSEEvent(w, "content_block_delta", map[string]any{
				"type": "content_block_delta", "index": 0,
				"delta": map[string]string{"type": "text_delta", "text": text},
			})
			if canFlush {
				flusher.Flush()
			}
		}
	}

	writeSSEEvent(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
	writeSSEEvent(w, "message_delta", map[string]any{
		"type":  "message_delta",
		"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil},
		"usage": map[string]int{"output_tokens": 0},
	})
	writeSSEEvent(w, "message_stop", map[string]any{"type": "message_stop"})
	if canFlush {
		flusher.Flush()
	}
}

func (h *Handler) handleCountTokens(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "count_tokens is not supported")
}

// extractText returns the text content from a streaming message chunk.
func extractText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	return msg.Content
}

func writeSSEEvent(w http.ResponseWriter, event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
