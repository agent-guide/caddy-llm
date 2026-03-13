package caddyllmrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/agent-guide/caddy-llm/llm"
	"github.com/agent-guide/caddy-llm/llm/models"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// Handler is a Caddy module that proxies Anthropic API requests to LLM providers
type Handler struct {
	// Provider configuration
	Provider      string            `json:"provider,omitempty"`
	APIKeys       map[string]string `json:"api_keys,omitempty"`
	ModelMappings map[string]string `json:"model_mappings,omitempty"`

	// Retry configuration
	MaxRetries int `json:"max_retries,omitempty"`
	RetryDelay int `json:"retry_delay,omitempty"` // in seconds

	// Fallback providers
	FallbackProviders []string `json:"fallback_providers,omitempty"`

	// Logging
	LogLevel    string `json:"log_level,omitempty"`
	LogRequests bool   `json:"log_requests,omitempty"`

	// Internal state
	provider  llm.LLMProvider
	converter *llm.Converter
	logger    *zap.Logger
	ctx       caddy.Context
	next      caddyhttp.Handler
}

// CaddyModule returns the Caddy module information
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.llm_router",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets up the handler
func (h *Handler) Provision(ctx caddy.Context) error {
	h.ctx = ctx
	h.logger = ctx.Logger(h)

	// Set defaults
	if h.Provider == "" {
		h.Provider = "openai"
	}
	if h.MaxRetries == 0 {
		h.MaxRetries = 3
	}
	if h.RetryDelay == 0 {
		h.RetryDelay = 2
	}

	// Initialize provider
	apiKey := ""
	if h.APIKeys != nil {
		if key, ok := h.APIKeys[h.Provider]; ok {
			apiKey = key
		}
	}

	config := llm.ProviderConfig{
		Name:   h.Provider,
		APIKey: apiKey,
		Model:  "default", // Will be overridden per request
	}

	var err error
	h.provider, err = llm.NewProvider(config)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Initialize converter
	h.converter = llm.NewConverter(h.Provider, h.ModelMappings)

	h.logger.Debug("LLM router provisioned",
		zap.String("provider", h.Provider),
		zap.Int("max_retries", h.MaxRetries),
	)

	return nil
}

// Validate ensures the handler's configuration is valid
func (h *Handler) Validate() error {
	if h.Provider == "" {
		return fmt.Errorf("no provider specified")
	}
	return nil
}

// ServeHTTP implements caddyhttp.Handler
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Only handle POST requests
	if r.Method != http.MethodPost {
		return next.ServeHTTP(w, r)
	}

	// Only handle /v1/messages and /v1/messages/count_tokens endpoints
	if !strings.HasPrefix(r.URL.Path, "/v1/messages") {
		return next.ServeHTTP(w, r)
	}

	// Log request if enabled
	if h.LogRequests {
		h.logger.Info("LLM request",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
		)
	}

	// Handle the request
	if strings.HasSuffix(r.URL.Path, "/count_tokens") {
		return h.handleCountTokens(w, r)
	}

	return h.handleMessages(w, r)
}

// handleMessages handles the /v1/messages endpoint
func (h *Handler) handleMessages(w http.ResponseWriter, r *http.Request) error {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return h.error(w, fmt.Errorf("failed to read request body: %w", err), http.StatusBadRequest)
	}
	defer r.Body.Close()

	// Parse Anthropic request
	var anthropicReq models.MessagesRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		return h.error(w, fmt.Errorf("failed to parse request: %w", err), http.StatusBadRequest)
	}

	// Check API key
	apiKey := r.Header.Get("x-api-key")
	if apiKey == "" {
		apiKey = r.Header.Get("Authorization")
		if strings.HasPrefix(apiKey, "Bearer ") {
			apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		}
	}

	if apiKey == "" && len(h.APIKeys) > 0 {
		// Use default API key if configured
		for _, key := range h.APIKeys {
			if key != "" {
				apiKey = key
				break
			}
		}
	}

	// Map the model
	mappedModel := h.converter.MapModel(anthropicReq.Model)
	h.logger.Debug("Model mapping",
		zap.String("original", anthropicReq.Model),
		zap.String("mapped", mappedModel),
	)

	// Convert Anthropic request to gollm prompt
	ctx := r.Context()
	prompt, err := h.converter.AnthropicToGollm(ctx, &anthropicReq)
	if err != nil {
		return h.error(w, fmt.Errorf("failed to convert request: %w", err), http.StatusInternalServerError)
	}

	// Handle streaming or non-streaming
	if anthropicReq.Stream {
		return h.handleStreaming(ctx, w, &anthropicReq, prompt)
	}

	return h.handleNonStreaming(ctx, w, &anthropicReq, prompt)
}

// handleNonStreaming handles non-streaming requests
func (h *Handler) handleNonStreaming(ctx context.Context, w http.ResponseWriter, req *models.MessagesRequest, prompt string) error {
	// Prepare generation options
	opts := []llm.GenerateOption{
		llm.WithMaxTokens(req.MaxTokens),
	}

	if req.Temperature > 0 {
		opts = append(opts, llm.WithTemperature(req.Temperature))
	}
	if req.TopP > 0 {
		opts = append(opts, llm.WithTopP(req.TopP))
	}
	if len(req.StopSequences) > 0 {
		opts = append(opts, llm.WithStopSequences(req.StopSequences))
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		tools, err := h.converter.ConvertTools(req.Tools)
		if err != nil {
			return h.error(w, fmt.Errorf("failed to convert tools: %w", err), http.StatusInternalServerError)
		}
		opts = append(opts, llm.WithTools(tools))
	}

	if req.ToolChoice != nil {
		toolChoice := h.converter.ConvertToolChoice(req.ToolChoice)
		opts = append(opts, llm.WithToolChoice(toolChoice))
	}

	// Generate response with retry logic
	var response string
	var lastErr error

	for attempt := 0; attempt <= h.MaxRetries; attempt++ {
		if attempt > 0 {
			h.logger.Debug("Reting request", zap.Int("attempt", attempt))
		}

		response, lastErr = h.provider.Generate(ctx, prompt, opts...)
		if lastErr == nil {
			break
		}

		// Check if we should retry
		if attempt < h.MaxRetries {
			h.logger.Warn("Generation failed, will retry",
				zap.Int("attempt", attempt+1),
				zap.Error(lastErr),
			)
		}
	}

	if lastErr != nil {
		return h.error(w, fmt.Errorf("failed to generate response after %d attempts: %w", h.MaxRetries+1, lastErr), http.StatusInternalServerError)
	}

	// Convert response back to Anthropic format
	anthropicResp, err := h.converter.GollmToAnthropic(response, req)
	if err != nil {
		return h.error(w, fmt.Errorf("failed to convert response: %w", err), http.StatusInternalServerError)
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(anthropicResp); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	h.logger.Debug("Request completed successfully",
		zap.String("model", anthropicResp.Model),
		zap.Int("input_tokens", anthropicResp.Usage.InputTokens),
		zap.Int("output_tokens", anthropicResp.Usage.OutputTokens),
	)

	return nil
}

// handleStreaming handles streaming requests
func (h *Handler) handleStreaming(ctx context.Context, w http.ResponseWriter, req *models.MessagesRequest, prompt string) error {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Flush headers
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Create stream writer
	streamWriter := llm.NewStreamResponseWriter(w)

	// Send start events
	startEvents := h.converter.GenerateStreamStartEvents(req)
	for _, event := range startEvents {
		if err := streamWriter.WriteRaw(event); err != nil {
			return fmt.Errorf("failed to write start event: %w", err)
		}
	}
	streamWriter.Flush()

	// Prepare generation options
	opts := []llm.GenerateOption{
		llm.WithMaxTokens(req.MaxTokens),
	}

	if req.Temperature > 0 {
		opts = append(opts, llm.WithTemperature(req.Temperature))
	}

	// Start streaming from provider
	chunkChan, err := h.provider.Stream(ctx, prompt, opts...)
	if err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}

	outputTokens := 0

	// Stream chunks
	for chunk := range chunkChan {
		outputTokens += len(chunk) / 4 // Rough estimate

		// Convert chunk to Anthropic SSE format
		events, err := h.converter.ProcessStreamChunk(chunk)
		if err != nil {
			h.logger.Error("Failed to process chunk", zap.Error(err))
			continue
		}

		// Write events
		for _, event := range events {
			if err := streamWriter.WriteRaw(event); err != nil {
				return fmt.Errorf("failed to write chunk: %w", err)
			}
		}
		streamWriter.Flush()
	}

	// Send end events
	endEvents := h.converter.GenerateStreamEndEvents(outputTokens)
	for _, event := range endEvents {
		if err := streamWriter.WriteRaw(event); err != nil {
			return fmt.Errorf("failed to write end event: %w", err)
		}
	}
	streamWriter.Flush()

	return nil
}

// handleCountTokens handles the /v1/messages/count_tokens endpoint
func (h *Handler) handleCountTokens(w http.ResponseWriter, r *http.Request) error {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return h.error(w, fmt.Errorf("failed to read request body: %w", err), http.StatusBadRequest)
	}
	defer r.Body.Close()

	// Parse request
	var req models.TokenCountRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return h.error(w, fmt.Errorf("failed to parse request: %w", err), http.StatusBadRequest)
	}

	// Estimate tokens (rough estimation)
	// In production, use a proper tokenizer
	inputTokens := 0
	for _, msg := range req.Messages {
		for _, block := range msg.Content {
			inputTokens += len(block.Text) / 4
		}
	}

	if req.System != "" {
		inputTokens += len(req.System) / 4
	}

	// Return response
	resp := models.TokenCountResponse{
		InputTokens: inputTokens,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// error writes an error response
func (h *Handler) error(w http.ResponseWriter, err error, status int) error {
	h.logger.Error("Request error", zap.Error(err), zap.Int("status", status))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := models.ErrorResponse{
		Type: "error",
		Error: models.ErrorDetail{
			Type:    "api_error",
			Message: err.Error(),
		},
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Failed to write error response: %v", err)
	}

	return caddyhttp.Error(http.StatusInternalServerError, err)
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for d.NextBlock(0) {
			switch d.Val() {
			case "provider":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.Provider = d.Val()

			case "api_key":
				if !d.NextArg() {
					return d.ArgErr()
				}
				if h.APIKeys == nil {
					h.APIKeys = make(map[string]string)
				}
				// Check if this is for a specific provider
				if d.NextArg() {
					provider := d.Val()
					h.APIKeys[provider] = provider
				} else {
					h.APIKeys[h.Provider] = d.Val()
				}

			case "map":
				if !d.NextArg() {
					return d.ArgErr()
				}
				from := d.Val()
				if !d.NextArg() {
					return d.ArgErr()
				}
				to := d.Val()

				if h.ModelMappings == nil {
					h.ModelMappings = make(map[string]string)
				}
				h.ModelMappings[from] = to

			case "fallback":
				providers := []string{}
				for d.NextArg() {
					providers = append(providers, d.Val())
				}
				h.FallbackProviders = providers

			case "max_retries":
				if !d.NextArg() {
					return d.ArgErr()
				}
				maxRetriesStr := d.Val()
				maxRetries, err := strconv.Atoi(maxRetriesStr)
				if err != nil {
					return d.Errf("invalid max_retries value: %v", err)
				}
				h.MaxRetries = maxRetries

			case "retry_delay":
				if !d.NextArg() {
					return d.ArgErr()
				}
				retryDelayStr := d.Val()
				retryDelay, err := strconv.Atoi(retryDelayStr)
				if err != nil {
					return d.Errf("invalid retry_delay value: %v", err)
				}
				h.RetryDelay = retryDelay

			case "log_level":
				if !d.NextArg() {
					return d.ArgErr()
				}
				h.LogLevel = d.Val()

			case "log_requests":
				h.LogRequests = true

			default:
				return d.Errf("unknown directive: %s", d.Val())
			}
		}
	}
	return nil
}

// Interface guards
var (
	_ caddy.Module                = (*Handler)(nil)
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddy.Validator             = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
)
