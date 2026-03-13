// Package anthropic implements the Anthropic provider (Claude models).
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/httputil"
)

const anthropicVersion = "2023-06-01"

func init() {
	provider.RegisterProvider("anthropic", New)
}

type anthropicProvider struct {
	config provider.ProviderConfig
	client *http.Client
}

// New creates a new Anthropic provider.
func New(config provider.ProviderConfig) (provider.Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("anthropic: api_key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	config.Network.Defaults()

	return &anthropicProvider{
		config: config,
		client: &http.Client{
			Timeout: config.Network.Timeout(),
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}, nil
}

func (p *anthropicProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	anthReq := BuildRequest(req, false)

	body, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var messagesResp MessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&messagesResp); err != nil {
		return nil, fmt.Errorf("anthropic: decode response: %w", err)
	}
	return ConvertResponse(&messagesResp, resp.Header.Clone()), nil
}

func (p *anthropicProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	anthReq := BuildRequest(req, true)

	body, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
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
			chunk := make([]byte, len(payload))
			copy(chunk, payload)
			ch <- provider.StreamChunk{Payload: chunk}
		}
		if err := scanner.Err(); err != nil {
			ch <- provider.StreamChunk{Err: fmt.Errorf("anthropic: scan stream: %w", err)}
		}
	}()

	return result, nil
}

// CountTokens uses the native Anthropic /v1/messages/count_tokens endpoint.
func (p *anthropicProvider) CountTokens(ctx context.Context, req *provider.GenerateRequest) (*provider.TokenCountResponse, error) {
	anthReq := BuildRequest(req, false)
	countReq := &CountTokensRequest{
		Model:    anthReq.Model,
		Messages: anthReq.Messages,
		System:   anthReq.System,
		Tools:    anthReq.Tools,
	}

	body, err := json.Marshal(countReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal count_tokens request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.config.BaseURL+"/v1/messages/count_tokens", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build count_tokens request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: count_tokens request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var countResp CountTokensResponse
	if err := json.NewDecoder(resp.Body).Decode(&countResp); err != nil {
		return nil, fmt.Errorf("anthropic: decode count_tokens response: %w", err)
	}
	return &provider.TokenCountResponse{InputTokens: countResp.InputTokens}, nil
}

// ListModels fetches available Claude models from GET /v1/models.
func (p *anthropicProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.config.BaseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("anthropic: decode models: %w", err)
	}

	out := make([]provider.ModelInfo, len(modelsResp.Data))
	for i, m := range modelsResp.Data {
		out[i] = provider.ModelInfo{ID: m.ID, Name: m.DisplayName}
	}
	return out, nil
}

func (p *anthropicProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming:       true,
		Tools:           true,
		Vision:          true,
		ContextWindow:   200000,
		MaxOutputTokens: 8192,
	}
}

func (p *anthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.config.APIKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	for k, v := range p.config.Network.ExtraHeaders {
		req.Header.Set(k, v)
	}
}

var _ provider.Provider = (*anthropicProvider)(nil)
