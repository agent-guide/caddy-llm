// Package gemini implements the Google Gemini provider.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/agent-guide/caddy-llm/llm/provider"
	"github.com/agent-guide/caddy-llm/llm/provider/httputil"
)

func init() {
	provider.RegisterProvider("gemini", New)
}

type geminiProvider struct {
	config provider.ProviderConfig
	client *http.Client
}

// New creates a new Google Gemini provider.
func New(config provider.ProviderConfig) (provider.Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("gemini: api_key is required")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com"
	}
	config.Network.Defaults()

	return &geminiProvider{
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

// generateURL returns the URL for generateContent or streamGenerateContent.
func (p *geminiProvider) generateURL(model string, stream bool) string {
	// Gemini URL pattern: /v1beta/models/{model}:{action}?key={apiKey}
	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}
	// Strip "models/" prefix if already present in model name.
	modelID := strings.TrimPrefix(model, "models/")
	return fmt.Sprintf("%s/v1beta/models/%s:%s?key=%s",
		p.config.BaseURL, modelID, action, p.config.APIKey)
}

func (p *geminiProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	gemReq := BuildRequest(req)
	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.generateURL(model, false), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var gemResp GenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&gemResp); err != nil {
		return nil, fmt.Errorf("gemini: decode response: %w", err)
	}
	return ConvertResponse(&gemResp, model, resp.Header.Clone()), nil
}

func (p *geminiProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	gemReq := BuildRequest(req)
	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal request: %w", err)
	}

	// Gemini streaming uses alt=sse to get SSE format.
	url := p.generateURL(model, true) + "&alt=sse"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
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
			ch <- provider.StreamChunk{Err: fmt.Errorf("gemini: scan stream: %w", err)}
		}
	}()

	return result, nil
}

// CountTokens uses the native Gemini countTokens endpoint.
func (p *geminiProvider) CountTokens(ctx context.Context, req *provider.GenerateRequest) (*provider.TokenCountResponse, error) {
	model := req.Model
	if model == "" {
		model = p.config.DefaultModel
	}

	gemReq := BuildRequest(req)
	countReq := &CountTokensRequest{Contents: gemReq.Contents}

	body, err := json.Marshal(countReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal countTokens request: %w", err)
	}

	modelID := strings.TrimPrefix(model, "models/")
	url := fmt.Sprintf("%s/v1beta/models/%s:countTokens?key=%s",
		p.config.BaseURL, modelID, p.config.APIKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: build countTokens request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: countTokens request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var countResp CountTokensResponse
	if err := json.NewDecoder(resp.Body).Decode(&countResp); err != nil {
		return nil, fmt.Errorf("gemini: decode countTokens response: %w", err)
	}
	return &provider.TokenCountResponse{InputTokens: countResp.TotalTokens}, nil
}

// ListModels fetches available Gemini models from GET /v1beta/models.
func (p *geminiProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	url := fmt.Sprintf("%s/v1beta/models?key=%s", p.config.BaseURL, p.config.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := httputil.CheckResponse(resp); err != nil {
		return nil, err
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("gemini: decode models: %w", err)
	}

	out := make([]provider.ModelInfo, len(modelsResp.Models))
	for i, m := range modelsResp.Models {
		// Name is "models/gemini-1.5-pro" — strip the prefix for the ID.
		id := strings.TrimPrefix(m.Name, "models/")
		out[i] = provider.ModelInfo{
			ID:          id,
			Name:        m.DisplayName,
			Description: m.Description,
			Capabilities: provider.ProviderCapabilities{
				ContextWindow:   m.InputTokenLimit,
				MaxOutputTokens: m.OutputTokenLimit,
			},
		}
	}
	return out, nil
}

func (p *geminiProvider) Capabilities() provider.ProviderCapabilities {
	return provider.ProviderCapabilities{
		Streaming:       true,
		Tools:           true,
		Vision:          true,
		Embeddings:      true,
		ContextWindow:   1000000,
		MaxOutputTokens: 8192,
	}
}

func (p *geminiProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	for k, v := range p.config.Network.ExtraHeaders {
		req.Header.Set(k, v)
	}
}

var _ provider.Provider = (*geminiProvider)(nil)
