package openaicompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/httputil"
)

// Base implements provider.Provider for OpenAI-compatible APIs.
// Embed *Base in a provider-specific struct and call NewBase to initialize.
// The embedding struct must supply Capabilities() to satisfy provider.Provider.
//
// Features provided out of the box:
//   - Proxy support via config.Network.ProxyURL
//   - Automatic retry with linear back-off via config.Network.MaxRetries / RetryDelaySeconds
//   - Vision (image content blocks)
//   - Streaming with usage reporting (stream_options.include_usage)
//   - Structured output and reasoning-effort via req.Metadata
//   - Extra request headers via config.Network.ExtraHeaders
type Base struct {
	config provider.ProviderConfig
	client *http.Client
}

// NewBase creates a Base using the supplied config.
// Call config.Network.Defaults() before passing it here.
// Proxy is configured automatically from config.Network.ProxyURL when non-empty.
func NewBase(config provider.ProviderConfig) *Base {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	if config.Network.ProxyURL != "" {
		if proxyURL, err := url.Parse(config.Network.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return &Base{
		config: config,
		client: &http.Client{
			Timeout:   config.Network.Timeout(),
			Transport: transport,
		},
	}
}

// Generate performs a non-streaming chat completion with automatic retry.
func (b *Base) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	chatReq, err := BuildChatRequest(req, false)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= b.config.Network.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(b.config.Network.RetryDelaySeconds) * time.Second * time.Duration(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := b.doGenerate(ctx, chatReq)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isRetryable(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("openaicompat: max retries exceeded: %w", lastErr)
}

func (b *Base) doGenerate(ctx context.Context, chatReq *ChatRequest) (*provider.GenerateResponse, error) {
	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openaicompat: build request: %w", err)
	}
	b.setHeaders(httpReq)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("openaicompat: decode response: %w", err)
	}
	return ConvertResponse(&chatResp, resp.Header.Clone()), nil
}

// Stream performs a streaming chat completion and returns raw SSE chunks.
func (b *Base) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	chatReq, err := BuildChatRequest(req, true)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openaicompat: build request: %w", err)
	}
	b.setHeaders(httpReq)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: request failed: %w", err)
	}
	if err := httputil.CheckResponse(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}

	ch := make(chan provider.StreamChunk, 16)
	result := &provider.StreamResult{
		Headers: resp.Header.Clone(),
		Chunks:  ch,
	}

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := httputil.NewSSEScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				ch <- provider.StreamChunk{Err: ctx.Err()}
				return
			default:
			}
			payload, isDone, ok := httputil.ParseSSELine(scanner.Bytes())
			if !ok {
				continue
			}
			if isDone {
				return
			}
			// Copy payload: scanner.Bytes() is reused on next iteration.
			chunk := make([]byte, len(payload))
			copy(chunk, payload)
			ch <- provider.StreamChunk{Payload: chunk}
		}
		if err := scanner.Err(); err != nil {
			ch <- provider.StreamChunk{Err: fmt.Errorf("openaicompat: scan stream: %w", err)}
		}
	}()

	return result, nil
}

// CountTokens returns a heuristic token estimate (4 chars ≈ 1 token).
// OpenAI-compatible APIs generally do not expose a count_tokens endpoint.
func (b *Base) CountTokens(_ context.Context, req *provider.GenerateRequest) (*provider.TokenCountResponse, error) {
	total := len(req.System) / 4
	for _, msg := range req.Messages {
		for _, block := range msg.Content {
			total += len(block.Text) / 4
		}
	}
	return &provider.TokenCountResponse{InputTokens: total}, nil
}

// ListModels fetches the model list from GET /v1/models.
func (b *Base) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		b.config.BaseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: build request: %w", err)
	}
	b.setHeaders(httpReq)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("openaicompat: decode models: %w", err)
	}

	out := make([]provider.ModelInfo, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		out[i] = provider.ModelInfo{ID: m.ID, Name: m.ID}
	}
	return out, nil
}

// Embed generates vector embeddings via POST /v1/embeddings.
func (b *Base) Embed(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResponse, error) {
	model := req.Model
	if model == "" {
		model = b.config.DefaultModel
	}

	embedReq := &EmbedRequest{Model: model, Input: req.Texts}
	body, err := json.Marshal(embedReq)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: marshal embed request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.config.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openaicompat: build embed request: %w", err)
	}
	b.setHeaders(httpReq)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openaicompat: embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("openaicompat: decode embed response: %w", err)
	}

	out := &provider.EmbedResponse{
		Model: embedResp.Model,
		Usage: provider.Usage{
			InputTokens:  embedResp.Usage.PromptTokens,
			OutputTokens: embedResp.Usage.CompletionTokens,
		},
	}
	for _, d := range embedResp.Data {
		out.Embeddings = append(out.Embeddings, d.Embedding)
	}
	return out, nil
}

func (b *Base) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	// Per-request credential override: use OAuth access_token from CLI login.
	if cred, ok := provider.CredentialFromContext(req.Context()); ok {
		if token, _ := cred.Metadata["access_token"].(string); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			for k, v := range b.config.Network.ExtraHeaders {
				req.Header.Set(k, v)
			}
			return
		}
	}

	if b.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.config.APIKey)
	}
	for k, v := range b.config.Network.ExtraHeaders {
		req.Header.Set(k, v)
	}
}

// isRetryable reports whether an error warrants an automatic retry.
// Network/transport errors are always retryable; HTTP 429 and 5xx are retryable.
func isRetryable(err error) bool {
	se, ok := err.(provider.StatusError)
	if !ok {
		return true // network/transport errors
	}
	code := se.StatusCode()
	return code == http.StatusTooManyRequests ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout ||
		code >= 500
}
