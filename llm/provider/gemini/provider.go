// Package gemini implements the Google Gemini provider.
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	einogemini "github.com/cloudwego/eino-ext/components/model/gemini"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"google.golang.org/genai"

	"github.com/agent-guide/caddy-llm/llm/authmanager/credential"
	"github.com/agent-guide/caddy-llm/llm/provider"
)

func init() {
	provider.RegisterProvider("gemini", New)
}

type geminiProvider struct {
	config      provider.ProviderConfig
	genaiClient *genai.Client // cached default client (no credential override)
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

	defaultClient, err := buildGenaiClient(context.Background(), config.APIKey, config.BaseURL, config.Network, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini: init client: %w", err)
	}
	return &geminiProvider{
		config:      config,
		genaiClient: defaultClient,
	}, nil
}

// buildGenaiClient constructs a genai.Client with the given credentials and network config.
// This is the single path for creating Gemini API clients in this package.
func buildGenaiClient(ctx context.Context, apiKey, baseURL string, network provider.NetworkConfig, cred *credential.Credential) (*genai.Client, error) {
	httpClient := provider.BuildHTTPClient(provider.ProviderConfig{Network: network}, nil, cred)
	timeout := network.Timeout()
	return genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:     apiKey,
		HTTPClient: httpClient,
		HTTPOptions: genai.HTTPOptions{
			BaseURL:    baseURL,
			APIVersion: "v1beta",
			Timeout:    &timeout,
		},
	})
}

func (p *geminiProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
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

func (p *geminiProvider) Stream(ctx context.Context, req *provider.GenerateRequest) (*schema.StreamReader[*schema.Message], error) {
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

// ListModels fetches available Gemini models from GET /v1beta/models.
func (p *geminiProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	url := fmt.Sprintf("%s/v1beta/models?key=%s", p.config.BaseURL, p.config.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	p.setHeaders(httpReq)

	httpClient := provider.BuildHTTPClient(p.config, nil, nil)
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := provider.CheckResponse(resp); err != nil {
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

func (p *geminiProvider) newChatModel(ctx context.Context, req *provider.GenerateRequest) (einomodel.ToolCallingChatModel, []*schema.Message, []einomodel.Option, error) {
	state, err := provider.ResolveChatRequest(ctx, p.config, req)
	if err != nil {
		return nil, nil, nil, err
	}

	// Reuse the cached client for the common path (no credential override).
	// Build a new one only when a per-request credential changes the API key or base URL.
	client := p.genaiClient
	if state.Credential != nil {
		client, err = buildGenaiClient(ctx, state.APIKey, state.BaseURL, p.config.Network, state.Credential)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("gemini: build credential client: %w", err)
		}
	}

	cfg := &einogemini.Config{
		Client: client,
		Model:  state.ModelName,
	}
	chatModel, err := einogemini.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	return chatModel, state.Messages, state.Options, nil
}

func (p *geminiProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	for k, v := range p.config.Network.ExtraHeaders {
		req.Header.Set(k, v)
	}
}

var _ provider.Provider = (*geminiProvider)(nil)
