package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/agent-guide/caddy-agent-gateway/api"
	"github.com/agent-guide/caddy-agent-gateway/gateway"
	routepkg "github.com/agent-guide/caddy-agent-gateway/gateway/route"
	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/cloudwego/eino/schema"
)

// Handler handles OpenAI-format API requests (/v1/chat/completions, etc.).
type Handler struct {
	RouteID string `json:"route_id,omitempty"`

	gateway *gateway.AgentGateway
}

func init() {
	caddy.RegisterModule(Handler{})
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.openai",
		New: func() caddy.Module { return new(Handler) },
	}
}

// NewHandler creates a Handler.
func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) SetRouteID(routeID string) {
	h.RouteID = routeID
}

func (h *Handler) SetAgentGateway(gw *gateway.AgentGateway) {
	h.gateway = gw
}

func (h *Handler) Provision(ctx caddy.Context) error {
	app, err := gateway.GetApp(ctx)
	if err != nil {
		return fmt.Errorf("openai llm api: get agent_gateway app: %w", err)
	}
	h.gateway = app.AgentGateway()
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
	// resolved, err := api.ResolveRequest(api.WithAgentGateway(r, h.gateway), genReq.Model, req.Stream, h.RouteID)
	resolved, err := h.gateway.ResolveProvider(r.Context(), h.RouteID, routepkg.ResolveRequest{
		HTTPRequest: r,
		Model:       genReq.Model,
		Stream:      req.Stream,
	})
	if err != nil {
		writeError(w, api.StatusCode(err), err.Error())
		return nil
	}

	if req.Stream {
		h.serveStream(w, r.Context(), resolved.Provider, genReq)
		return nil
	}

	resp, err := resolved.Provider.Generate(r.Context(), genReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return nil
	}
	writeJSON(w, http.StatusOK, conv.FromInternal(resp, genReq.Model))
	return nil
}

func (h *Handler) serveStream(w http.ResponseWriter, ctx context.Context, prov provider.Provider, genReq *provider.GenerateRequest) {
	stream, err := prov.Stream(ctx, genReq)
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
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
)
