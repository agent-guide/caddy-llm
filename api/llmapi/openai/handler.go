package openai

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

// Handler handles OpenAI-format API requests (/v1/chat/completions, etc.).
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
		ID:  "http.handlers.llm_api.openai",
		New: func() caddy.Module { return new(Handler) },
	}
}

// NewHandler creates a Handler with the given auth manager and provider.
func NewHandler(authMgr *manager.Manager, prov provider.Provider) *Handler {
	return &Handler{Provider: "openai", authManager: authMgr, prov: prov}
}

func (h *Handler) Provision(ctx caddy.Context) error {
	if h.Provider == "" {
		h.Provider = "openai"
	}

	app, err := llm.GetApp(ctx)
	if err != nil {
		return fmt.Errorf("openai llm api: get llm app: %w", err)
	}
	prov, ok := app.Provider(h.Provider)
	if !ok {
		return fmt.Errorf("openai llm api: provider %q is not configured", h.Provider)
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

// ServeHTTP handles OpenAI API requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if !strings.HasPrefix(r.URL.Path, "/v1/chat/completions") &&
		!strings.HasPrefix(r.URL.Path, "/v1/models") &&
		!strings.HasPrefix(r.URL.Path, "/v1/embeddings") {
		return next.ServeHTTP(w, r)
	}
	return h.ServeLLMApi(w, r)
}

// ServeLLMApi handles OpenAI-compatible API requests.
func (h *Handler) ServeLLMApi(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return nil
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %s", err))
		return nil
	}

	conv := &Converter{}
	genReq := conv.ToInternal(&req)

	// Pick a credential and inject into context for per-request auth override.
	ctx := r.Context()
	var cred *credential.Credential
	if c, err := h.authManager.Pick(ctx, "openai", genReq.Model, nil); err == nil && c != nil {
		cred = c
		ctx = provider.WithCredential(ctx, c)
	}

	if req.Stream {
		h.serveStream(w, ctx, genReq, cred)
		return nil
	}

	resp, err := h.prov.Generate(ctx, genReq)
	h.markResult(ctx, cred, genReq.Model, err)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return nil
	}
	writeJSON(w, http.StatusOK, conv.FromInternal(resp, genReq.Model))
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

func (h *Handler) serveStream(w http.ResponseWriter, ctx context.Context, genReq *provider.GenerateRequest, cred *credential.Credential) {
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
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		payload, err := json.Marshal(toStreamChunk(genReq.Model, chunk))
		if err != nil {
			break
		}
		fmt.Fprintf(w, "data: %s\n\n", payload)
		if canFlush {
			flusher.Flush()
		}
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	if canFlush {
		flusher.Flush()
	}
}

type chatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []chunkChoice `json:"choices"`
}

type chunkChoice struct {
	Index        int        `json:"index"`
	Delta        chunkDelta `json:"delta"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

type chunkDelta struct {
	Role      string            `json:"role,omitempty"`
	Content   string            `json:"content,omitempty"`
	ToolCalls []schema.ToolCall `json:"tool_calls,omitempty"`
}

func toStreamChunk(model string, msg *schema.Message) *chatCompletionChunk {
	chunk := &chatCompletionChunk{
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []chunkChoice{{
			Index: 0,
			Delta: chunkDelta{
				Role:    string(msg.Role),
				Content: msg.Content,
			},
		}},
	}
	if len(msg.ToolCalls) > 0 {
		chunk.Choices[0].Delta.ToolCalls = msg.ToolCalls
	}
	if msg.ResponseMeta != nil {
		chunk.Choices[0].FinishReason = msg.ResponseMeta.FinishReason
	}
	return chunk
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
