package anthropic

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

// Handler handles Anthropic-format API requests (/v1/messages).
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
		ID:  "http.handlers.anthropic",
		New: func() caddy.Module { return new(Handler) },
	}
}

// NewHandler creates a Handler.
func NewHandler(_ provider.Provider) *Handler {
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
		return fmt.Errorf("anthropic llm api: get agent_gateway app: %w", err)
	}
	h.gateway = app.AgentGateway()
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
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusMethodNotAllowed, "method not allowed", fmt.Errorf("method %s not allowed", r.Method),
			zap.String("protocol", "anthropic"),
			zap.String("route_id", h.RouteID),
		)
		return nil
	}
	if strings.HasSuffix(r.URL.Path, "/count_tokens") {
		h.handleCountTokens(w, r)
		return nil
	}
	h.handleMessages(w, r)
	return nil
}

func (h *Handler) handleMessages(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusBadRequest, "failed to read request body", fmt.Errorf("read request body: %w", err),
			zap.String("protocol", "anthropic"),
			zap.String("route_id", h.RouteID),
		)
		return
	}

	var req MessagesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusBadRequest, fmt.Sprintf("invalid request: %s", err), fmt.Errorf("decode request body: %w", err),
			zap.String("protocol", "anthropic"),
			zap.String("route_id", h.RouteID),
		)
		return
	}

	conv := &Converter{}
	genReq := conv.ToInternal(&req)
	resolved, err := h.gateway.ResolveProvider(r.Context(), h.RouteID, routepkg.ResolveRequest{
		HTTPRequest: r,
		Model:       genReq.Model,
		Stream:      req.Stream,
	})
	if err != nil {
		status := api.StatusCode(err)
		_ = utils.WriteLoggedError(h.logger, w, r, status, err.Error(), fmt.Errorf("resolve provider: %w", err),
			zap.String("protocol", "anthropic"),
			zap.String("route_id", h.RouteID),
			zap.String("model", genReq.Model),
		)
		return
	}

	if req.Stream {
		h.serveStream(w, r, resolved.Provider, genReq, req.Model)
		return
	}

	resp, err := resolved.Provider.Generate(r.Context(), genReq)
	if err != nil {
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusBadGateway, err.Error(), fmt.Errorf("generate response: %w", err),
			zap.String("protocol", "anthropic"),
			zap.String("route_id", h.RouteID),
			zap.String("model", genReq.Model),
		)
		return
	}
	_ = utils.WriteJSON(w, http.StatusOK, conv.FromInternal(resp, req.Model))
}

func (h *Handler) serveStream(w http.ResponseWriter, r *http.Request, prov provider.Provider, genReq *provider.GenerateRequest, model string) {
	ctx := r.Context()
	stream, err := prov.Stream(ctx, genReq)
	if err != nil {
		_ = utils.WriteLoggedError(h.logger, w, r, http.StatusBadGateway, err.Error(), fmt.Errorf("start stream: %w", err),
			zap.String("protocol", "anthropic"),
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
			utils.LogHTTPError(h.logger, "http request failed", r, http.StatusOK, fmt.Errorf("receive stream chunk: %w", err),
				zap.String("protocol", "anthropic"),
				zap.String("route_id", h.RouteID),
				zap.String("model", genReq.Model),
			)
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
	_ = utils.WriteError(w, http.StatusNotImplemented, "count_tokens is not supported")
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

var (
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
)
