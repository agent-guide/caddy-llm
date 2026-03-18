// Package anthropic implements the Anthropic provider (Claude models).
package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	einoclaude "github.com/cloudwego/eino-ext/components/model/claude"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/agent-guide/caddy-llm/llm/provider"
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
	return provider.RetryGenerate(p.config.Network, func() (*provider.GenerateResponse, error) {
		chatModel, messages, opts, err := p.newChatModel(ctx, req)
		if err != nil {
			return nil, err
		}
		msg, err := chatModel.Generate(ctx, messages, opts...)
		if err != nil {
			return nil, provider.WrapEinoError(err)
		}
		return provider.FromEinoMessage(msg), nil
	})
}

func (p *anthropicProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
	chatModel, messages, opts, err := p.newChatModel(ctx, req)
	if err != nil {
		return nil, err
	}
	stream, err := chatModel.Stream(ctx, messages, opts...)
	if err != nil {
		return nil, provider.WrapEinoError(err)
	}
	return stream, nil
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

	if err := provider.CheckResponse(resp); err != nil {
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

func (p *anthropicProvider) newChatModel(ctx context.Context, req *provider.GenerateRequest) (einomodel.ToolCallingChatModel, []*schema.Message, []einomodel.Option, error) {
	state, err := provider.ResolveChatRequest(ctx, p.config, req)
	if err != nil {
		return nil, nil, nil, err
	}

	maxTokens := 0
	if state.CommonOptions.MaxTokens != nil {
		maxTokens = *state.CommonOptions.MaxTokens
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	cfg := &einoclaude.Config{
		APIKey:     state.APIKey,
		Model:      state.ModelName,
		MaxTokens:  maxTokens,
		HTTPClient: provider.BuildHTTPClient(p.config, nil, state.Credential),
	}
	if state.BaseURL != "" {
		cfg.BaseURL = &state.BaseURL
	}

	chatModel, err := einoclaude.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	return chatModel, state.Messages, state.Options, nil
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
