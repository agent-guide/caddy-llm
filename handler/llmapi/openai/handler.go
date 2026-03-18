package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/agent-guide/caddy-llm/llm/auth/manager"
	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/cloudwego/eino/schema"
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
	writeJSON(w, http.StatusOK, conv.FromInternal(resp, genReq.Model))
}

func (h *Handler) serveStream(w http.ResponseWriter, ctx context.Context, genReq *provider.GenerateRequest) {
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
