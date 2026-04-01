package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/agent-guide/caddy-agent-gateway/api"
	"github.com/agent-guide/caddy-agent-gateway/gateway"
	routepkg "github.com/agent-guide/caddy-agent-gateway/gateway/route"
	"github.com/agent-guide/caddy-agent-gateway/internal/utils"
	"github.com/agent-guide/caddy-agent-gateway/llm/provider"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// Handler handles OpenAI-format API requests (/v1/chat/completions, etc.).
type Handler struct {
	RouteID string `json:"route_id,omitempty"`

	gateway *gateway.AgentGateway
	logger  *zap.Logger
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
	return &Handler{logger: zap.NewNop()}
}

func (h *Handler) SetRouteID(routeID string) {
	h.RouteID = routeID
}

func (h *Handler) SetAgentGateway(gw *gateway.AgentGateway) {
	h.gateway = gw
}

func (h *Handler) Provision(ctx caddy.Context) error {
	h.logger = ctx.Logger(h)
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
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusMethodNotAllowed, "method not allowed", fmt.Errorf("method %s not allowed", r.Method),
			zap.String("protocol", "openai"),
			zap.String("route_id", h.RouteID),
		)
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusBadRequest, "failed to read request body", fmt.Errorf("read request body: %w", err),
			zap.String("protocol", "openai"),
			zap.String("route_id", h.RouteID),
		)
		return nil
	}

	var req ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusBadRequest, fmt.Sprintf("invalid request: %s", err), fmt.Errorf("decode request body: %w", err),
			zap.String("protocol", "openai"),
			zap.String("route_id", h.RouteID),
		)
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
		status := api.StatusCode(err)
		_ = utils.WriteLoggedError(h.logger, w, r, status, err.Error(), fmt.Errorf("resolve provider: %w", err),
			zap.String("protocol", "openai"),
			zap.String("route_id", h.RouteID),
			zap.String("model", genReq.Model),
		)
		return nil
	}

	if req.Stream {
		h.serveStream(w, r, resolved.Provider, genReq)
		return nil
	}

	resp, err := resolved.Provider.Generate(r.Context(), genReq)
	if err != nil {
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusBadGateway, err.Error(), fmt.Errorf("generate response: %w", err),
			zap.String("protocol", "openai"),
			zap.String("route_id", h.RouteID),
			zap.String("model", genReq.Model),
		)
		return nil
	}
	_ = utils.WriteJSON(w, http.StatusOK, conv.FromInternal(resp, genReq.Model))
	return nil
}

func (h *Handler) serveStream(w http.ResponseWriter, r *http.Request, prov provider.Provider, genReq *provider.GenerateRequest) {
	ctx := r.Context()
	stream, err := prov.Stream(ctx, genReq)
	if err != nil {
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusBadGateway, err.Error(), fmt.Errorf("start stream: %w", err),
			zap.String("protocol", "openai"),
			zap.String("route_id", h.RouteID),
			zap.String("model", genReq.Model),
		)
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
			utils.LogHTTPError(h.logger, "http request failed", r, http.StatusOK, fmt.Errorf("receive stream chunk: %w", err),
				zap.String("protocol", "openai"),
				zap.String("route_id", h.RouteID),
				zap.String("model", genReq.Model),
			)
			break
		}

		payload, err := json.Marshal(toStreamChunk(genReq.Model, chunk))
		if err != nil {
			utils.LogHTTPError(h.logger, "http request failed", r, http.StatusOK, fmt.Errorf("marshal stream chunk: %w", err),
				zap.String("protocol", "openai"),
				zap.String("route_id", h.RouteID),
				zap.String("model", genReq.Model),
			)
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

var (
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
)
