package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	llm "github.com/agent-guide/caddy-llm/llm"
	"github.com/agent-guide/caddy-llm/llm/authmanager/credential"
	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/cloudwego/eino/schema"
)

// Handler handles Anthropic-format API requests (/v1/messages).
type Handler struct {
	Provider string `json:"provider,omitempty"`

	authManager *manager.Manager
	prov        provider.Provider
}

func init() {
	caddy.RegisterModule(Handler{})
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.llm_api.anthropic",
		New: func() caddy.Module { return new(Handler) },
	}
}

// NewHandler creates a Handler wired with the given auth manager and provider.
func NewHandler(authMgr *manager.Manager, prov provider.Provider) *Handler {
	return &Handler{Provider: "anthropic", authManager: authMgr, prov: prov}
}

func (h *Handler) Provision(ctx caddy.Context) error {
	if h.Provider == "" {
		h.Provider = "anthropic"
	}

	app, err := llm.GetApp(ctx)
	if err != nil {
		return fmt.Errorf("anthropic llm api: get llm app: %w", err)
	}
	prov, ok := app.Provider(h.Provider)
	if !ok {
		return fmt.Errorf("anthropic llm api: provider %q is not configured", h.Provider)
	}

	h.authManager = app.AuthManager()
	h.prov = prov
	return nil
}

func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "provider":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.Provider = d.Val()
			default:
				return d.Errf("unknown subdirective: %s", d.Val())
			}
		}
	}
	return nil
}

// ServeHTTP handles /v1/messages and /v1/messages/count_tokens.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if !strings.HasPrefix(r.URL.Path, "/v1/messages") {
		return next.ServeHTTP(w, r)
	}
	return h.ServeLLMApi(w, r)
}

// ServeLLMApi handles Anthropic-compatible API requests.
func (h *Handler) ServeLLMApi(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}
	if strings.HasSuffix(r.URL.Path, "/count_tokens") {
		h.handleCountTokens(w, r)
		return nil
	}
	h.handleMessages(w, r)
	return nil
}

// markResult feeds the outcome of a provider call back to the auth manager so
// quota tracking and credential health are updated accordingly.
func (h *Handler) markResult(ctx context.Context, cred *credential.Credential, model string, err error) {
	if h.authManager == nil || cred == nil {
		return
	}
	result := manager.Result{
		CredentialID: cred.ID,
		Provider:     cred.Provider,
		Model:        model,
		Success:      err == nil,
	}
	if err != nil {
		var se provider.StatusError
		httpStatus := http.StatusBadGateway
		if errors.As(err, &se) {
			httpStatus = se.StatusCode()
		}
		result.Error = &credential.Error{
			Code:       http.StatusText(httpStatus),
			Message:    err.Error(),
			HTTPStatus: httpStatus,
			Retryable:  httpStatus == http.StatusTooManyRequests || httpStatus >= 500,
		}
	}
	h.authManager.MarkResult(ctx, result)
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

	// Pick an anthropic credential and inject into context for per-request auth override.
	ctx := r.Context()
	var cred *credential.Credential
	if c, err := h.authManager.Pick(ctx, "anthropic", genReq.Model, nil); err == nil && c != nil {
		cred = c
		ctx = provider.WithCredential(ctx, c)
	}

	if req.Stream {
		h.serveStream(w, ctx, genReq, req.Model, cred)
		return
	}

	resp, err := h.prov.Generate(ctx, genReq)
	h.markResult(ctx, cred, genReq.Model, err)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, conv.FromInternal(resp, req.Model))
}

func (h *Handler) serveStream(w http.ResponseWriter, ctx context.Context, genReq *provider.GenerateRequest, model string, cred *credential.Credential) {
	stream, err := h.prov.Stream(ctx, genReq)
	h.markResult(ctx, cred, genReq.Model, err)
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

var (
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
)
